package pcs_client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"

	"backup/consts"
	"backup/internal/token"
	"backup/pkg/byte_pool"
	"backup/pkg/group"
	"backup/pkg/logger"
)

const RetryCount = 3

type UploadParams struct {
	Path     string `json:"path"`
	UploadId string `json:"uploadid"`
	PartSeq  int    `json:"partseq"`
	File     []byte `json:"-"`
}

type UploadResponse struct {
	Errno     int    `json:"errno"`
	Md5       string `json:"md5"`
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
	RequestId int64  `json:"request_id"`
}

func Upload(ctx context.Context, uploadId string, path string, partSeq []int, filename string) error {
	logger.Logger.WithContext(ctx).WithField("filename", filename).Info("begin upload")
	if len(partSeq) == 0 {
		partSeq = append(partSeq, 0)
	}
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).WithField("filename", filename).Error("open file fail")
		return err
	}
	defer file.Close()

	sendGroup := group.NewMyGroup(ctx, 3, 3).WithContext(ctx).WithErrorStrategy(&RetryStrategy{MaxTime: RetryCount})
	sendGroup.Start()
	for _, seq := range partSeq {
		chunk := byte_pool.DefaultBytePool.Get()
		n, err := file.Read(chunk)
		if err != nil && err != io.EOF {
			logger.Logger.WithContext(ctx).WithError(err).WithField("filename", filename).Error("read file fail")
			return err
		}

		params := &UploadParams{
			UploadId: uploadId,
			Path:     path,
			PartSeq:  seq,
			File:     chunk[:n],
		}

		task := &UploadTask{
			Name:   fmt.Sprintf("%s_%d", path, seq),
			Params: params,
		}

		var retryCounter = 0
		for retryCounter < RetryCount {
			err = sendGroup.Submit(task)
			if err == nil {
				break
			}
			logger.Logger.WithError(err).WithContext(ctx).WithField("taskName", task.Name).Warn("task submit fail")
			retryCounter++
		}
	}

	if err = sendGroup.Wait(); err != nil {
		logger.Logger.WithError(err).WithField("filename", filename).Error("upload fail")
		return err
	}
	logger.Logger.WithContext(ctx).WithField("filename", filename).Info("end upload")
	return nil
}

func uploadSlice(ctx context.Context, params *UploadParams) error {
	request, err := params.GenerateRequest(ctx, params.Path)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("generate request fail")
		return err
	}

	logger.Logger.WithContext(ctx).WithField("params", params).Info("upload file start")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("upload file fail")
		return err
	}

	defer response.Body.Close()
	logger.Logger.WithContext(ctx).WithField("params", params).WithField("response", fmt.Sprintf("%+v", response)).Info("upload file end")

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).Error("read response data fail")
		return err
	}

	var resp = &UploadResponse{}
	err = jsoniter.Unmarshal(data, &resp)
	if err != nil {
		logger.Logger.WithContext(ctx).WithError(err).WithField("resp", string(data)).Error("unmarshal response fail")
		return err
	}

	if resp.Errno != consts.ErrnoSuccess || resp.ErrorCode != consts.ErrnoSuccess {
		err = errors.Errorf("upload fail")
		logger.Logger.WithContext(ctx).WithError(err).WithField("response", resp).Error("upload fail")
		return err
	}

	byte_pool.DefaultBytePool.Put(params.File)

	return nil
}

func (p *UploadParams) GenerateRequest(ctx context.Context, filename string) (*http.Request, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("filename", filename).Info("pcs upload slice: request generate start")

	address := fmt.Sprintf("https://d.pcs.baidu.com/rest/2.0/pcs/superfile2?method=%s&access_token=%s", consts.MethodUpload, token.AccessToken)

	encodeString, err := p.GenEncodeString(ctx)
	if err != nil {
		baseLogger.WithError(err).Error("pcs upload slice: generate encode string fail")
		return nil, err
	}

	address = fmt.Sprintf("%s&%s", address, encodeString)

	buffer := bytes.NewBufferString("")
	writer := multipart.NewWriter(buffer)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		baseLogger.WithError(err).Error("pcs upload slice: create file fail")
		return nil, err
	}

	_, err = part.Write(p.File)
	if err != nil {
		baseLogger.WithError(err).Error("pcs upload slice: write to file fail")
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		baseLogger.WithError(err).Error("pcs upload slice: close file fail")
		return nil, err
	}

	req, err := http.NewRequest("POST", address, buffer)
	if err != nil {
		baseLogger.WithError(err).Error("pcs upload slice: request generate fail")
		return nil, err
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())
	baseLogger.Info("pcs upload slice: request generate success")
	return req, nil
}

func (p *UploadParams) GenEncodeString(ctx context.Context) (string, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("params", p).Info("pcs upload slice: generate encode string start")

	body, err := jsoniter.Marshal(p)
	if err != nil {
		baseLogger.WithField("params", p).WithError(err).Errorf("marshal params fail")
		return "", err
	}

	var param = map[string]interface{}{}
	err = jsoniter.Unmarshal(body, &param)
	if err != nil {
		baseLogger.WithField("params", p).WithError(err).Errorf("unmarshal params fail")
		return "", err
	}
	var values = url.Values{}
	for key, value := range param {
		values.Set(key, fmt.Sprintf("%+v", value))
	}
	values.Set("type", "tmpfile")
	res := values.Encode()

	baseLogger.WithField("result", res).Info("pcs upload slice: generate encoding string success")
	return res, nil
}

type UploadTask struct {
	Name       string        `json:"name"`
	RetryCount int           `json:"retry_count"`
	Params     *UploadParams `json:"params"`
}

func (r *UploadTask) Run(ctx context.Context) error {
	return uploadSlice(ctx, r.Params)
}

type RetryStrategy struct {
	MaxTime int
}

func (s RetryStrategy) ErrorDeal(group *group.MyGroup, err error, task group.TaskInterface) {
	retryTask, ok := task.(*UploadTask)
	if ok && retryTask.RetryCount < s.MaxTime {
		retryTask.RetryCount++
		logger.Logger.WithError(err).WithField("taskName", retryTask.Name).Errorf("task retry %v times", retryTask.RetryCount)
		err = group.Submit(task)
		if err != nil {
			s.ErrorDeal(group, err, task)
		}
	} else {
		logger.Logger.WithError(err).WithField("taskName", retryTask.Name).Errorf("task execute fail")
		group.Once.Do(func() {
			group.Err = err
			group.Stop()
		})
	}
}
