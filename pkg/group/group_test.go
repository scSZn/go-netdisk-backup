package group

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

type MockTask struct {
	Name     string
	RunField func() error
}

func (t *MockTask) Run(ctx context.Context) error {
	return t.RunField()
}

func TestMyGroup_Stop(t *testing.T) {
	tests := []struct {
		name  string
		size  int
		count int
	}{
		{
			name:  "testStop",
			size:  5,
			count: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewMyGroup(context.Background(), tt.size, 0)
			for i := 0; i < tt.count; i++ {
				task := &MockTask{
					Name: "testStop",
					RunField: func() error {
						fmt.Println(101010)
						time.Sleep(5 * time.Second)
						return nil
					},
				}
				g.Submit(task)
				g.wg.Done()
			}
			g.Start()
			fmt.Println(g.Wait())
		})
	}
}

func TestMyGroup_Submit(t *testing.T) {
	type args struct {
		task *MockTask
	}
	tests := []struct {
		name  string
		size  int
		args  args
		count int
	}{
		{
			name: "testSubmit",
			size: 5,
			args: struct{ task *MockTask }{
				task: &MockTask{
					Name: "testSubmit",
					RunField: func() error {
						fmt.Println(101010)
						return nil
					},
				},
			},
			count: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewMyGroup(context.Background(), tt.size, 0)
			for i := 0; i < tt.count; i++ {
				g.Submit(tt.args.task)
			}
			g.Start()
			fmt.Println(g.Wait())
		})
	}
}

func TestMyGroup_WithContext(t *testing.T) {
	type args struct {
		task *MockTask
	}
	tests := []struct {
		name  string
		size  int
		args  args
		count int
	}{
		{
			name: "testContext",
			size: 5,
			args: struct{ task *MockTask }{
				task: &MockTask{
					Name: "testContext",
					RunField: func() error {
						fmt.Println(101010)
						time.Sleep(5 * time.Second)
						return nil
					},
				},
			},
			count: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := context.WithTimeout(context.Background(), time.Second*7)
			g := NewMyGroup(context.Background(), tt.size, 0).WithContext(ctx)
			for i := 0; i < tt.count; i++ {
				g.Submit(tt.args.task)
			}
			g.Start()
			fmt.Println(g.Wait())
		})
	}
}

func TestMyGroup_Panic(t *testing.T) {
	type args struct {
		task *MockTask
	}
	tests := []struct {
		name  string
		size  int
		args  args
		count int
	}{
		{
			name: "testPanic",
			size: 5,
			args: struct{ task *MockTask }{
				task: &MockTask{
					Name: "testPanic",
					RunField: func() error {
						fmt.Println("panic before")
						if rand.Intn(10) < 7 {
							panic("test panic")
						}
						fmt.Println("panic after")
						return nil
					},
				},
			},
			count: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewMyGroup(context.Background(), tt.size, 0)
			for i := 0; i < tt.count; i++ {
				g.Submit(tt.args.task)
			}
			g.Start()
			fmt.Println(g.Wait())
		})
	}
}

//func TestMyGroup_work(t *testing.T) {
//	type fields struct {
//		tasks        chan func() error
//		size         int
//		once         *sync.Once
//		err          error
//		ctx          context.Context
//		cancel       context.CancelFunc
//		wg           *sync.WaitGroup
//		stopChan     chan struct{}
//		recoveryFunc func(interface{})
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			g := &MyGroup{
//				tasks:        tt.fields.tasks,
//				size:         tt.fields.size,
//				once:         tt.fields.once,
//				err:          tt.fields.err,
//				ctx:          tt.fields.ctx,
//				cancel:       tt.fields.cancel,
//				wg:           tt.fields.wg,
//				stopChan:     tt.fields.stopChan,
//				recoveryFunc: tt.fields.recoveryFunc,
//			}
//			g.work()
//		})
//	}
//}
//
//func TestNewMyGroup(t *testing.T) {
//	type args struct {
//		size int
//	}
//	tests := []struct {
//		name string
//		args args
//		want *MyGroup
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := NewMyGroup(tt.args.size); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewMyGroup() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
