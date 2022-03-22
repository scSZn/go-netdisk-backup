package scan

import (
	"context"
	"testing"
)

func TestScanner_Scan(t *testing.T) {
	type fields struct {
		ctx  context.Context
		root string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "scan",
			fields: fields{
				ctx:  context.Background(),
				root: "/Users/didi/study/backup",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewScanner(tt.fields.ctx, tt.fields.root)
			s.Scan()
		})
	}
}
