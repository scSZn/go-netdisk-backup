package work_pool

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"testing"

	"github.com/pkg/errors"
)

func TestWorkPool_work(t *testing.T) {
	p := NewWorkPool(context.Background(), 5, 10, WorkModeSlowStart)
	p.Start()
	group := NewTaskGroup(context.Background(), 10000)
	group.RunSuccess = func(ctx context.Context, task *Task) {
		fmt.Printf("任务 %s 执行成功success\n", task.Name)
	}
	group.RunFail = func(ctx context.Context, task *Task, err error) {
		fmt.Printf("任务 %s 执行失败fail\n", task.Name)
	}
	for i := 0; i < 100000; i++ {
		task := NewTask(group, strconv.Itoa(i), 1)
		task.Run = func(ctx context.Context, task *Task) error {
			if rand.Int31n(2) == 1 {
				return errors.New("")
			}
			return nil
		}
		err := p.Submit(task)
		if err != nil {
			log.Fatalf("task %d 提交失败", i)
		}
	}
}

func TestTaskGroup_Wait(t *testing.T) {
	p := NewWorkPool(context.Background(), 5, 10, WorkModeSlowStart)
	p.Start()
	group := NewTaskGroup(context.Background(), 10000)
	group.RunSuccess = func(ctx context.Context, task *Task) {
		fmt.Printf("任务 %s 执行成功success\n", task.Name)
	}
	group.RunFail = func(ctx context.Context, task *Task, err error) {
		fmt.Printf("任务 %s 执行失败fail\n", task.Name)
	}
	go func() {
		for i := 0; i < 100000; i++ {
			task := NewTask(group, strconv.Itoa(i), 1)
			task.Run = func(ctx context.Context, task *Task) error {
				if rand.Int31n(10) == 1 {
					return errors.New("")
				}
				return nil
			}
			err := p.Submit(task)
			if err != nil {
				log.Fatalf("task %d 提交失败", i)
			}
		}
	}()

	group.Wait()
}
