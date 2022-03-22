package pcs_client

import (
	"context"
	"fmt"
	"testing"

	"backup/consts"
	"backup/pkg/util"
)

func Test_create(t *testing.T) {
	type args struct {
		ctx    context.Context
		params *CreateParams
	}
	tests := []struct {
		name    string
		args    args
		want    *CreateResponse
		wantErr bool
	}{
		{
			name: "create",
			args: args{
				ctx: context.TODO(),
				params: &CreateParams{
					Path:       "/apps/宝宝专用文件夹/dump.txt",
					Size:       20,
					IsDir:      0,
					UploadId:   "N1-MTExLjIwNC4xODIuMTAwOjE2NDc2NjA3Mzc6ODcxNDUyNjM3NzY2NDkzNTgzMQ==",
					RType:      1,
					Mode:       consts.ModeManual,
					IsRevision: consts.EnableMultiVersion,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md5, err := util.Md5(tt.args.ctx, []byte("HelloHelloHelloHello"))
			if err != nil {
				t.Errorf("%+v", err)
			}
			tt.args.params.BlockList = []string{md5}
			got, err := Create(tt.args.ctx, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("%+v", got)
		})
	}
}
