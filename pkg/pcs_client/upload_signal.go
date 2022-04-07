package pcs_client

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"

	"backup/consts"
	"backup/pkg/byte_pool"
	"backup/pkg/group"
	"backup/pkg/logger"
)

func pcsUploadWithSignal(ctx context.Context, uploadId string, path string, partSeq []int, filename string, signal chan struct{}) error {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithFields(map[string]interface{}{
		"filename":    filename,
		"server_path": path,
		"upload_id":   uploadId,
		"part_seq":    partSeq,
	}).Info("pcs upload start")

	if len(partSeq) == 0 {
		partSeq = append(partSeq, 0)
	}
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		baseLogger.WithError(err).WithField("filename", filename).Error("open file fail")
		return err
	}
	defer file.Close()

	sendGroup := group.NewMyGroup(ctx, 3, 3).WithErrorStrategy(&RetryStrategyWithSignal{MaxTime: RetryCount})
	sendGroup.Start()
	for _, seq := range partSeq {
		chunk := byte_pool.DefaultBytePool.Get()
		n, err := file.Read(chunk)

		params := &UploadParams{
			UploadId: uploadId,
			Path:     path,
			PartSeq:  seq,
			File:     chunk[:n],
		}

		task := &UploadTaskWithSignal{
			Name:   fmt.Sprintf("%s_%d", path, seq),
			Params: params,
			Signal: signal,
			client: &http.Client{},
			SpanId: uuid.New().String(),
		}

		baseLogger.WithField("task", task).Info("task submit")
		err = sendGroup.Submit(task)
		if err != nil {
			baseLogger.WithContext(ctx).WithField("task", task).WithError(err).Warn("task submit fail")
			sendGroup.Cancel()
			return err
		}
	}

	if err = sendGroup.Wait(); err != nil {
		baseLogger.WithError(err).Error("pcs upload fail")
		return err
	}
	baseLogger.WithField("filename", filename).Info("pcs upload success")
	return nil
}

func uploadSliceWithSignal(ctx context.Context, params *UploadParams, client *http.Client, signal chan struct{}) error {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithFields(map[string]interface{}{
		"params":        params,
		"file_size(B)":  len(params.File),
		"file_size(KB)": len(params.File) / 1024,
		"file_size(MB)": len(params.File) / 1024 / 1024,
	}).Info("pcs upload slice start")

	request, err := params.GenerateRequest(ctx, params.Path)
	if err != nil {
		baseLogger.WithError(err).Error("generate request fail")
		return err
	}
	request.WithContext(ctx)
	response, err := client.Do(request)
	if err != nil {
		baseLogger.WithError(err).Error("upload request fail")
		return err
	}
	defer response.Body.Close()

	baseLogger.WithFields(map[string]interface{}{
		"header": response.Header,
		"status": response.Status,
	}).Info("upload slice: response info")
	if response.StatusCode != http.StatusOK {
		baseLogger.WithField("response", response).Error("response status code is not 200")
		return errors.Errorf("response status code is not 200")
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		baseLogger.WithError(err).Error("read response data fail")
		return err
	}
	baseLogger.WithField("response_body", string(data)).Info("pcs upload slice: response body")

	if string(data) == "" {
		baseLogger.WithField("response", response).Error("response body data is empty")
	}
	var resp = &UploadResponse{}
	err = jsoniter.Unmarshal(data, &resp)
	if err != nil {
		baseLogger.WithError(err).Error("unmarshal response fail")
		return err
	}

	if resp.Errno != consts.ErrnoSuccess || resp.ErrorCode != consts.ErrnoSuccess {
		err = errors.Errorf("pcs upload slice: upload fail")
		baseLogger.WithError(err).WithField("response", resp).Error("pcs upload slice: upload fail")
		return err
	}

	if signal != nil {
		signal <- struct{}{}
	}

	byte_pool.DefaultBytePool.Put(params.File)
	baseLogger.WithError(err).WithField("response", resp).Info("pcs upload slice: upload success")
	return nil
}

type UploadTaskWithSignal struct {
	Name       string        `json:"name"`
	RetryCount int           `json:"retry_count"`
	Params     *UploadParams `json:"params"`
	Signal     chan struct{} `json:"-"`
	SpanId     string        `json:"span_id"`
	client     *http.Client
}

func (r *UploadTaskWithSignal) Run(ctx context.Context) error {
	ctx = context.WithValue(ctx, consts.LogSpanId, r.SpanId)
	return uploadSliceWithSignal(ctx, r.Params, r.client, r.Signal)
}

type RetryStrategyWithSignal struct {
	MaxTime int
}

func (s RetryStrategyWithSignal) ErrorDeal(group *group.MyGroup, err error, task group.TaskInterface) {
	// 如果任务为空，返回
	// 如果err是上下文取消的err，返回
	if task == nil || err == context.Canceled {
		return
	}
	retryTask, ok := task.(*UploadTaskWithSignal)
	if ok && retryTask.RetryCount < s.MaxTime {
		retryTask.RetryCount++
		logger.Logger.WithError(err).WithField("taskName", retryTask.Name).Errorf("task retry %v times", retryTask.RetryCount)
		go func() {
			err = group.Submit(task)
			if err != nil {
				s.ErrorDeal(group, err, task)
			}
		}()
	} else {
		logger.Logger.WithError(err).WithField("taskName", retryTask.Name).Errorf("task execute fail")
		group.Once.Do(func() {
			group.Err = err
			group.Stop()
		})
	}
}
