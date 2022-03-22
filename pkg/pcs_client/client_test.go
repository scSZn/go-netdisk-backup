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
				filename:       "/Users/didi/Downloads/productid_map",
				serverFilename: "productid_map",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UploadFile(tt.args.ctx, tt.args.filename, tt.args.serverFilename); (err != nil) != tt.wantErr {
				t.Errorf("UploadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
