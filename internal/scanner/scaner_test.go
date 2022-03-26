package scanner

import (
	"context"
	"testing"
)

func TestScanner_ScanAndUpload(t *testing.T) {
	type fields struct {
		ctx  context.Context
		root string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "scanner",
			fields: fields{
				ctx:  context.Background(),
				root: "/Users/didi/study/backup",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewScanner(tt.fields.ctx, tt.fields.root)
			s.ScanAndUpload()
		})
	}
}
