package pcs_client

import (
	"context"
	"testing"
)

func TestUpload(t *testing.T) {
	type args struct {
		ctx      context.Context
		uploadId string
		path     string
		partSeq  []int
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "upload",
			args: args{
				ctx:      context.TODO(),
				path:     "/apps/宝宝专用文件夹/test.txt",
				uploadId: "N1-MTExLjIwNC4xODIuMTAwOjE2NDc2NjA3Mzc6ODcxNDUyNjM3NzY2NDkzNTgzMQ==",
				partSeq:  []int{0},
				filename: "/Users/didi/dump.rdb",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Upload(tt.args.ctx, tt.args.uploadId, tt.args.path, tt.args.partSeq, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("Upload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
