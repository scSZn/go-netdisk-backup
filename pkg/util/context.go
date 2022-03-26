package util

import (
	"context"

	"github.com/google/uuid"

	"backup/consts"
)

func NewContext() context.Context {
	baseContext := context.Background()
	ctx := context.WithValue(baseContext, consts.LogTraceKey, uuid.New().String())
	return ctx
}
