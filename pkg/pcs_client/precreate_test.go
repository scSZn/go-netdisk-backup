package pcs_client

import (
	"context"
	"fmt"
	"testing"

	"backup/pkg/util"
)

func TestPreCreate(t *testing.T) {
	type args struct {
		context context.Context
		params  *PreCreateParams
	}
	tests := []struct {
		name    string
		args    args
		want    *PreCreateResponse
		wantErr bool
	}{
		{
			name: "precreate",
			args: args{
				context: context.TODO(),
				params: &PreCreateParams{
					Path:  "/apps/宝宝专用文件夹/dump.txt",
					Size:  20,
					IsDir: 0,
					RType: 0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md5, err := util.Md5(tt.args.context, []byte("HelloHelloHelloHello"))
			if err != nil {
				t.Errorf("%+v", err)
			}
			tt.args.params.BlockList = []string{md5}
			got, err := PreCreate(tt.args.context, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("preCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("%+v", got)
		})
	}
}
