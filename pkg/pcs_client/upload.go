package pcs_client

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"backup/consts"
	"backup/internal/token"
	"backup/pkg/byte_pool"
	"backup/pkg/logger"
	"backup/pkg/work_pool"
)

func pcsUpload(ctx context.Context, uploadReq *uploadRequest) error {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("uploadReq", uploadReq).Info("pcs upload start")

	if len(uploadReq.PartSeq) == 0 {
		uploadReq.PartSeq = append(uploadReq.PartSeq, 0)
	}
	file, err := os.OpenFile(uploadReq.Filename, os.O_RDONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "open file fail")
	}
	defer file.Close()

	var group = work_pool.NewTaskGroup(ctx, len(uploadReq.PartSeq))
	group.RunFail = func(ctx context.Context, task *work_pool.Task, err error) {
		baseLogger.WithFields(map[string]interface{}{
			"task":          task,
			logrus.ErrorKey: err,
		}).WithError(err).Errorf("task execute fail, retry")

		err = task.Retry(p)
		if err != nil {
			baseLogger.WithFields(map[string]interface{}{
				"task":          task,
				logrus.ErrorKey: err,
			}).Errorf("task retry fail")
			group.Fail(err)
		}
	}
	group.RunSuccess = func(ctx context.Context, task *work_pool.Task) {
		uploadReq.RefreshFunc()
	}
	for _, seq := range uploadReq.PartSeq {
		chunk := byte_pool.DefaultBytePool.Get()
		n, err := file.Read(chunk)
		if err != nil {
			return errors.Wrap(err, "read chunk from file")
		}

		params := &uploadTaskParams{
			UploadId:   uploadReq.UploadId,
			ServerPath: uploadReq.ServerPath,
			PartSeq:    seq,
			Content:    chunk[:n],
		}

		task := work_pool.NewTask(group, fmt.Sprintf("%s_%d", uploadReq.ServerPath, seq), consts.MaxRetryCount)
		task.Run = func(ctx context.Context, task *work_pool.Task) error {
			return uploadChunk(ctx, params, http.DefaultClient)
		}

		baseLogger.WithField("task", task).Info("task submit")

		err = p.Submit(task)
		if err != nil {
			return errors.Wrap(err, "submit task fail")
		}
	}

	if err = group.Wait(); err != nil {
		return errors.Wrapf(err, "group task run fail")
	}

	baseLogger.WithField("filename", uploadReq.Filename).Info("pcs upload success")
	return nil
}

func uploadChunk(ctx context.Context, params *uploadTaskParams, client *http.Client) error {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithFields(map[string]interface{}{
		"params":        params,
		"file_size(B)":  len(params.Content),
		"file_size(KB)": len(params.Content) / 1024,
		"file_size(MB)": len(params.Content) / 1024 / 1024,
	}).Info("pcs upload chunk start")

	request, err := params.GenerateRequest(ctx, params.ServerPath)
	if err != nil {
		return errors.Wrap(err, "generate upload request fail")
	}
	request.WithContext(ctx)
	response, err := client.Do(request)
	if err != nil {
		return errors.Wrap(err, "upload request fail")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("response status code is %v", response.StatusCode)
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrap(err, "read response data fail")
	}
	baseLogger.WithField("response_body", string(data)).Info()

	if string(data) == "" {
		return errors.New("response body data is empty")
	}
	var resp = &uploadResponse{}
	err = jsoniter.Unmarshal(data, &resp)
	if err != nil {
		return errors.Wrap(err, "unmarshal response fail")
	}

	if resp.Errno != consts.ErrnoSuccess || resp.ErrorCode != consts.ErrnoSuccess {
		return errors.Errorf("upload chunk fail")
	}

	byte_pool.DefaultBytePool.Put(params.Content)
	baseLogger.WithError(err).WithField("response", resp).Info("upload chunk success")
	return nil
}

type uploadRequest struct {
	UploadId    string `json:"upload_id"`
	ServerPath  string `json:"server_path"`
	PartSeq     []int  `json:"part_seq"`
	Filename    string `json:"filename"`
	RefreshFunc func() `json:"-"`
}

func NewUploadRequest(uploadId string, serverPath string, partSeq []int, filename string, refreshFunc func()) *uploadRequest {
	return &uploadRequest{UploadId: uploadId, ServerPath: serverPath, PartSeq: partSeq, Filename: filename, RefreshFunc: refreshFunc}
}

type uploadResponse struct {
	Errno     int    `json:"errno"`
	Md5       string `json:"md5"`
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
	RequestId int64  `json:"request_id"`
}

type uploadTaskParams struct {
	ServerPath string `json:"path"`
	UploadId   string `json:"uploadid"`
	PartSeq    int    `json:"partseq"`
	Content    []byte `json:"-"`
}

func (p *uploadTaskParams) GenerateRequest(ctx context.Context, filename string) (*http.Request, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("filename", filename).Info("generate request start")

	address := fmt.Sprintf("https://d.pcs.baidu.com/rest/2.0/pcs/superfile2?method=%s&access_token=%s", consts.MethodUpload, token.AccessToken)

	encodeString, err := p.GenEncodeString(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "generate encode string fail")
	}

	address = fmt.Sprintf("%s&%s", address, encodeString)
	buffer := bytes.NewBufferString("")
	writer := multipart.NewWriter(buffer)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, errors.Wrap(err, "create form file fail")
	}

	_, err = part.Write(p.Content)
	if err != nil {
		return nil, errors.Wrap(err, "write to form file fail")
	}

	err = writer.Close()
	if err != nil {
		return nil, errors.Wrap(err, "close form file fail")
	}

	req, err := http.NewRequest("POST", address, buffer)
	if err != nil {
		return nil, errors.Wrap(err, "request generate fail")
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())
	baseLogger.Info("generate request success")
	return req, nil
}

func (p *uploadTaskParams) GenEncodeString(ctx context.Context) (string, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("params", p).Info("generate encode string start")

	body, err := jsoniter.Marshal(p)
	if err != nil {
		return "", errors.Wrap(err, "marshal params fail")
	}

	var param = map[string]interface{}{}
	err = jsoniter.Unmarshal(body, &param)
	if err != nil {
		return "", errors.Wrap(err, "unmarshal params fail")
	}
	var values = url.Values{}
	for key, value := range param {
		values.Set(key, fmt.Sprintf("%+v", value))
	}
	values.Set("type", "tmpfile")
	res := values.Encode()

	baseLogger.WithField("result", res).Info("generate encoding string end")
	return res, nil
}
