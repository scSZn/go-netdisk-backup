package watch

import (
	"context"
	"testing"
)

func TestWatcher_Start(t *testing.T) {
	type args struct {
		ctx  context.Context
		root string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				ctx:  context.Background(),
				root: "./test",
			},
		},
	}
	signal := make(chan struct{})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := NewWatcher(tt.args.ctx, tt.args.root)
			if err != nil {
				t.Errorf("new watcher fail, error: %+v", err)
			}
			w.Start()
			<-signal
		})
	}
}
