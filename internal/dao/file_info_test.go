package dao

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/gorm"

	"backup/internal/model"
	"backup/pkg/database"
	"backup/pkg/util"
)

func TestFileInfoDao_Add(t *testing.T) {
	type fields struct {
		ctx context.Context
		DB  *gorm.DB
	}
	type args struct {
		info *model.FileInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "addTest",
			fields: fields{
				ctx: context.Background(),
				DB:  database.DB,
			},
			args: args{
				info: &model.FileInfo{
					AbsPath: "/Users/didi/dump.rdb",
					Size:    0,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.info.Md5, _ = util.GetFileMd5(tt.fields.ctx, tt.args.info.AbsPath)
			d := &FileInfoDao{
				ctx: tt.fields.ctx,
				DB:  tt.fields.DB,
			}
			if err := d.Add(tt.args.info); (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileInfoDao_QueryByAbsPath(t *testing.T) {
	type fields struct {
		ctx context.Context
		DB  *gorm.DB
	}
	type args struct {
		absPath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *model.FileInfo
		wantErr bool
	}{
		{
			name: "queryTest",
			fields: fields{
				ctx: context.Background(),
				DB:  database.DB,
			},
			args: args{
				absPath: "/Users/didi/dump.rdb",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &FileInfoDao{
				ctx: tt.fields.ctx,
				DB:  tt.fields.DB,
			}
			got, err := d.QueryByAbsPath(tt.args.absPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryByAbsPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("%+v\n", got)
		})
	}
}

func TestFileInfoDao_Update(t *testing.T) {
	type fields struct {
		ctx context.Context
		DB  *gorm.DB
	}
	type args struct {
		filename string
		updates  map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "queryUpdate",
			fields: fields{
				ctx: context.Background(),
				DB:  database.DB,
			},
			args: args{
				filename: "/Users/didi/dump.rdb",
				updates: map[string]interface{}{
					"size": 1000,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &FileInfoDao{
				ctx: tt.fields.ctx,
				DB:  tt.fields.DB,
			}
			if err := d.Update(tt.args.updates, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
