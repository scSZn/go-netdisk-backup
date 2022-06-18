package pcs_client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"

	"backup/consts"
	"backup/internal/config"
	"backup/internal/token"
	"backup/pkg/logger"
	"backup/pkg/util"
)

func pcsPreCreate(ctx context.Context, preCreateReq *preCreateRequest) (*preCreateResponse, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("preCreateReq", preCreateReq).Info("pcs precreate start")

	preCreateReq.AutoInit = consts.AutoInitConstant
	encodeString, err := preCreateReq.GenEncodeString(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "construct encode string fail")
	}

	address := fmt.Sprintf("http://pan.baidu.com/rest/2.0/xpan/file?method=%s&access_token=%s", consts.MethodPrecreate, token.AccessToken)
	req, err := http.NewRequest(http.MethodPost, address, bytes.NewBufferString(encodeString))
	if err != nil {
		return nil, errors.Wrap(err, "construct request fail")
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "precreate request fail")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("response status code is %+v", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	baseLogger.WithField("response_body", string(data)).Info("pcs precreate response")

	var preCreateResp = &preCreateResponse{}
	err = jsoniter.Unmarshal(data, preCreateResp)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal params fail")
	}

	if preCreateResp.Errno != consts.ErrnoSuccess {
		return preCreateResp, errors.Errorf("errno isn't 0, preCreateResp is [%+v]", resp)
	}

	baseLogger.WithField("preCreateResp", preCreateResp).Info("pcs precreate success")
	return preCreateResp, nil
}

type preCreateRequest struct {
	Path         string   `json:"path" bind:"required"`  // 上传文件的绝对路径
	Size         int64    `json:"size" bind:"required"`  // 文件大小，单位为B
	IsDir        uint8    `json:"isdir" bind:"required"` // 0 文件，1 目录
	BlockList    []string `json:"-"`
	BlockListStr string   `json:"block_list" bind:"required"`         // 分块的md5数组（32位小写）
	AutoInit     uint8    `json:"autoinit,omitempty" bind:"required"` // 固定值为1
	RType        uint8    `json:"rtype,omitempty"`                    // 文件名冲突策略，0 表示不进行重命名，若云端存在同名文件返回错误
	UploadId     string   `json:"uploadid,omitempty"`                 // 上传ID
	ContentMd5   string   `json:"content-md5,omitempty"`              // 文件MD5，32位小写
	SliceMd5     string   `json:"slice-md5,omitempty"`                // 文件校验段的MD5，32位小写，校验段对应文件前256KB
	LocalCTime   string   `json:"local_ctime,omitempty"`              // 客户端创建时间
	LocalMTime   string   `json:"local_mtime,omitempty"`              // 客户端修改时间
}

type preCreateResponse struct {
	Errno      int    `json:"errno"`
	Path       string `json:"path"`        // 文件的绝对路径
	UploadId   string `json:"uploadid"`    // 上传ID
	ReturnType uint8  `json:"return_type"` // 返回类型，1 文件在云端不存在，2 文件在云端已存在
	BlockList  []int  `json:"block_list"`  // 需要上传的分片序号列表，索引从0开始
}

func NewPreCreateRequest(ctx context.Context, filename, serverPath string) (*preCreateRequest, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithFields(map[string]interface{}{
		"filename":   filename,
		"serverPath": serverPath,
	})

	baseLogger.Infof("construct preCreateRequest start")
	if serverPath == "" {
		return nil, errors.Errorf("serverFilename is empty, filename is %s", filename)
	}
	list, err := util.GetBlockList(ctx, filename)
	if err != nil {
		return nil, errors.Wrap(err, "get block list fail")
	}

	serverPath = path.Join(config.Config.PcsConfig.PathPrefix, serverPath)
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, errors.Wrap(err, "get file stat fail")
	}

	var isDir uint8 = 0
	if stat.IsDir() {
		isDir = 1
	}

	request := &preCreateRequest{
		Path:      serverPath,
		Size:      stat.Size(),
		IsDir:     isDir,
		BlockList: list,
		RType:     consts.RTypeOverride,
	}

	baseLogger.WithField("result", request).Infof("construct preCreateRequest end")
	return request, nil
}

func (c *preCreateRequest) GenEncodeString(ctx context.Context) (string, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.Infof("generate encode string start")

	tempListStr := make([]string, 0, len(c.BlockListStr))
	for _, str := range c.BlockList {
		tempListStr = append(tempListStr, fmt.Sprintf("\"%s\"", str))
	}
	c.BlockListStr = fmt.Sprintf("[%s]", strings.Join(tempListStr, ","))

	body, err := jsoniter.Marshal(c)
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
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float64, float32:
			str, _ := jsoniter.MarshalToString(value)
			values.Set(key, str)
		default:
			values.Set(key, fmt.Sprintf("%+v", value))
		}
	}

	var res = values.Encode()
	baseLogger.WithField("result", res).Info("generate encoding string end")
	return res, nil
}
