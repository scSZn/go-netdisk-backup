package pcs_client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	jsoniter "github.com/json-iterator/go"

	"backup/consts"
	"backup/internal/token"
	"backup/pkg/logger"
)

type PreCreateParams struct {
	Path         string   `json:"path,omitempty" bind:"required"` // 上传文件的绝对路径
	Size         int64    `json:"size,omitempty" bind:"required"` // 文件大小，单位为B
	IsDir        uint8    `json:"isdir" bind:"required"`          // 0 文件，1 目录
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

type PreCreateResponse struct {
	Errno      int    `json:"errno"`
	Path       string `json:"path"`        // 文件的绝对路径
	UploadId   string `json:"uploadid"`    // 上传ID
	ReturnType uint8  `json:"return_type"` // 返回类型，1 文件在云端不存在，2 文件在云端已存在
	BlockList  []int  `json:"block_list"`  // 需要上传的分片序号列表，索引从0开始
}

func PreCreate(context context.Context, params *PreCreateParams) (*PreCreateResponse, error) {
	params.AutoInit = consts.AutoInitConstant

	encodeString, err := params.GenEncodeString()
	if err != nil {
		logger.Logger.WithContext(context).WithField("params", fmt.Sprintf("%+v", params)).WithField("error", fmt.Sprintf("%+v", err)).Errorf("construct encode string fail")
		return nil, err
	}

	address := fmt.Sprintf("http://pan.baidu.com/rest/2.0/xpan/file?method=%s&access_token=%s", consts.MethodPrecreate, token.AccessToken)
	req, err := http.NewRequest(http.MethodPost, address, bytes.NewBufferString(encodeString))
	if err != nil {
		logger.Logger.WithContext(context).WithField("params", fmt.Sprintf("%+v", params)).WithField("error", fmt.Sprintf("%+v", err)).Errorf("construct request fail")
		return nil, err
	}
	req = req.WithContext(context)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	logger.Logger.WithContext(context).WithField("request", fmt.Sprintf("%+v", req)).Info("start request precreate")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Logger.WithContext(context).WithField("params", fmt.Sprintf("%+v", params)).WithField("error", fmt.Sprintf("%+v", err)).Errorf("precreate fail")
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	logger.Logger.WithContext(context).WithField("response", string(data)).Info("request precreate finished")

	var resp = &PreCreateResponse{}
	err = jsoniter.Unmarshal(data, resp)
	if err != nil {
		logger.Logger.WithContext(context).
			WithField("params", fmt.Sprintf("%+v", params)).
			WithField("error", fmt.Sprintf("%+v", err)).
			Errorf("unmarshal params fail")
		return nil, err
	}

	return resp, nil
}

func (c *PreCreateParams) GenEncodeString() (string, error) {
	tempListStr := make([]string, 0, len(c.BlockListStr))
	for _, str := range c.BlockList {
		tempListStr = append(tempListStr, fmt.Sprintf("\"%s\"", str))
	}
	c.BlockListStr = fmt.Sprintf("[%s]", strings.Join(tempListStr, ","))

	body, err := jsoniter.Marshal(c)
	if err != nil {
		logger.Logger.WithField("params", fmt.Sprintf("%+v", c)).
			WithField("error", fmt.Sprintf("%+v", err)).
			Errorf("marshal params fail")
		return "", err
	}

	var param = map[string]interface{}{}
	err = jsoniter.Unmarshal(body, &param)
	if err != nil {
		logger.Logger.WithField("params", fmt.Sprintf("%+v", c)).
			WithField("error", fmt.Sprintf("%+v", err)).
			Errorf("unmarshal params fail")
		return "", err
	}
	var values = url.Values{}
	for key, value := range param {
		values.Set(key, fmt.Sprintf("%+v", value))
	}

	return values.Encode(), nil
}
