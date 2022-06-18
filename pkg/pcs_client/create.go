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
	"github.com/pkg/errors"

	"backup/consts"
	"backup/internal/token"
	"backup/pkg/logger"
)

func pcsCreate(ctx context.Context, createReq *createRequest) (*createResponse, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.WithField("createReq", createReq).Info("pcs create start")

	address := fmt.Sprintf("https://pan.baidu.com/rest/2.0/xpan/file?method=%s&access_token=%s", consts.MethodCreate, token.AccessToken)

	encodeString, err := createReq.GenEncodeString(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "construct encode string fail")
	}

	req, err := http.NewRequest(http.MethodPost, address, bytes.NewBufferString(encodeString))
	if err != nil {
		return nil, errors.Wrap(err, "construct request fail")
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "request fail")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("response status code is %+v", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	baseLogger.WithField("response_body", string(data)).Info("pcs create: response body")

	var createResp = &createResponse{}
	err = jsoniter.Unmarshal(data, resp)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal createResp fail")
	}

	if createResp.Errno != consts.ErrnoSuccess {
		return nil, errors.Errorf("errno is not zero")
	}

	baseLogger.WithField("createResp", createResp).Info("pcs create: create success")
	return createResp, nil
}

type createRequest struct {
	Path         string   `json:"path,omitempty" bind:"required"` // 上传文件的绝对路径
	Size         int64    `json:"size,omitempty" bind:"required"` // 文件大小，单位为B
	IsDir        uint8    `json:"isdir" bind:"required"`          // 0 文件，1 目录
	BlockList    []string `json:"-"`
	BlockListStr string   `json:"block_list" bind:"required"` // 分块的md5数组（32位小写）
	RType        uint8    `json:"rtype,omitempty"`            // 文件名冲突策略，0 表示不进行重命名，若云端存在同名文件返回错误
	UploadId     string   `json:"uploadid,omitempty"`         // 上传ID
	LocalCTime   string   `json:"local_ctime,omitempty"`      // 客户端创建时间
	LocalMTime   string   `json:"local_mtime,omitempty"`      // 客户端修改时间
	ZipQuality   int      `json:"zip_quality,omitempty"`      // 图片压缩程度
	ZipSign      string   `json:"zip_sign,omitempty"`         // 未压缩原始图片的MD5
	IsRevision   uint8    `json:"is_revision,omitempty"`      // 是否开启多版本，1开启，0不开启
	Mode         uint8    `json:"mode,omitempty"`             // 上传模式
	//ExifInfo     ExifInfo `json:"exif_info"`                  // 图片的ExifInfo信息
}

type createResponse struct {
	Errno          int    `json:"errno"`
	FsId           int64  `json:"fs_id"`
	Md5            string `json:"md5"`
	ServerFilename string `json:"server_filename"`
	Category       int    `json:"category"`
	Path           string `json:"path"`
	Size           int    `json:"size"`
	Ctime          int    `json:"ctime"`
	Mtime          int    `json:"mtime"`
	IsDir          int    `json:"isdir"`
	Name           string `json:"name"`
}

func (c *createRequest) GenEncodeString(ctx context.Context) (string, error) {
	baseLogger := logger.Logger.WithContext(ctx)
	baseLogger.Info("generate encode string start")

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
	baseLogger.WithField("result", res).Info("pcs create: generate encode string success")
	return res, nil
}
