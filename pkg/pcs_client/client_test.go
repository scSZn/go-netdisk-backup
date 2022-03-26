package pcs_client

import (
	"context"
	"testing"
)

func TestUploadFile(t *testing.T) {
	type args struct {
		ctx            context.Context
		filename       string
		serverFilename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "UploadFileTest",
			args: args{
				ctx:            context.TODO(),
				filename:       "E:\\Temp\\2020年12月四级真题\\2020年12月英语四级解析第1套.pdf",
				serverFilename: "/apps/test/Temp\\2020年12月四级真题\\2020年12月英语四级解析第1套.pdf",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UploadFileWithSignal(tt.args.ctx, tt.args.filename, tt.args.serverFilename, nil); (err != nil) != tt.wantErr {
				t.Errorf("UploadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
