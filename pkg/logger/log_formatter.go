package logger

import (
	"context"
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"

	"backup/consts"
)

type LogFormatter struct {
}

func (l *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	level := strings.ToUpper(entry.Level.String())
	time := entry.Time.Format(consts.LogTimeLayout)

	caller := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)

	// 获取TraceID
	traceId := getTraceInfo(entry.Context)
	var slice = make([]string, 0, len(entry.Data)+2)
	slice = append(slice, fmt.Sprintf("%s=%v", consts.LogTraceKey, traceId))

	// 获取SpanID
	spanId := getSpanId(entry.Context)
	slice = append(slice, fmt.Sprintf("%s=%v", consts.LogSpanId, spanId))

	// 拼装消息
	for k, v := range entry.Data {
		var vStr string
		var err error
		if k == logrus.ErrorKey { // 这样可以把errors包中的cause给打印出来
			vStr = fmt.Sprintf("%+v", v)
		} else {
			vStr, err = jsoniter.MarshalToString(v)
		}
		if err != nil {
			return nil, err
		}
		slice = append(slice, fmt.Sprintf("%s=%s", k, vStr))
	}

	slice = append(slice, fmt.Sprintf("%s=%v", "message", entry.Message))
	return []byte(fmt.Sprintf("[%s][%s][%v] %s\n", level, time, caller, strings.Join(slice, "||"))), nil
}

// 如果是gin.Context，从gin.Context中获取traceId，否则从context中获取traceId
func getTraceInfo(ctx context.Context) interface{} {
	if ctx != nil {
		return ctx.Value(consts.LogTraceKey)
	}
	return ""
}

// 如果是gin.Context，从gin.Context中获取traceId，否则从context中获取traceId
func getSpanId(ctx context.Context) interface{} {
	if ctx != nil {
		return ctx.Value(consts.LogSpanId)
	}
	return ""
}
