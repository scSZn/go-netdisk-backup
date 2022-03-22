package util

import (
	"context"
	"fmt"
	"testing"
)

func TestGetSubDir(t *testing.T) {
	type args struct {
		ctx     context.Context
		dirname string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "get_sub_dir",
			args: args{
				ctx:     context.TODO(),
				dirname: "/Users/didi/study/backup",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSubDir(tt.args.ctx, tt.args.dirname)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSubDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("%+v", got)
		})
	}
}

func TestGetSubDirV2(t *testing.T) {
	type args struct {
		ctx     context.Context
		dirname string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "get_sub_dir",
			args: args{
				ctx:     context.TODO(),
				dirname: "/Users/didi/study/backup",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSubDirV2(tt.args.ctx, tt.args.dirname)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSubDirV2() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("%+v\n", got)
		})
	}
}
