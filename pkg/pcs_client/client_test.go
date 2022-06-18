package pcs_client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync/atomic"
	"testing"
)

func TestUpload(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:8080", nil))
	}()
	type args struct {
		ctx    context.Context
		params *UploadParams
	}
	var count int64
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				ctx: context.Background(),
				params: NewUploadParams("E:\\Test\\Hello.txt", "hello.txt", func() {
					newValue := atomic.AddInt64(&count, 1)
					fmt.Printf("当前进度为 %d\n", newValue)
				}, func() {
					fmt.Printf("上传完成")
				}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Upload(tt.args.ctx, tt.args.params); (err != nil) != tt.wantErr {
				t.Errorf("Upload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
