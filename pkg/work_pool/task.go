package work_pool

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

type Task struct {
	Name string `json:"name"`

	retryCount    int
	maxRetryCount int
	group         *TaskGroup
	resultChan    chan interface{}

	Run func(ctx context.Context, task *Task) error `json:"-"`
}

func NewTask(g *TaskGroup, name string, maxRetryCount int) *Task {
	if g == nil {
		panic("TaskGroup must not be nil")
	}

	t := &Task{
		Name:          name,
		group:         g,
		maxRetryCount: maxRetryCount,
		resultChan:    make(chan interface{}, 1),
	}

	return t
}

func (t *Task) run() error {
	if t.Run != nil {
		return t.Run(t.group.ctx, t)
	}
	return nil
}

func (t *Task) Result() interface{} {
	return <-t.resultChan
}

func (t *Task) ResultChan() chan<- interface{} {
	return t.resultChan
}

func (t *Task) Output(result interface{}) {
	t.resultChan <- result
}

func (t *Task) Retry(p *WorkPool) error {
	t.retryCount++
	if t.retryCount > t.maxRetryCount {
		return errors.Errorf("task %s exceed max retry times", t.Name)
	}
	return p.Submit(t)
}

// 任务组，每个任务都有一个任务组，控制该任务组下所有任务的执行，取消等等
type TaskGroup struct {
	ctx           context.Context
	cancelFunc    context.CancelFunc
	errorChan     chan error
	doneTaskCount uint64
	taskNumber    uint64
	once          *sync.Once

	RunBefore  func(ctx context.Context, task *Task) bool
	RunSuccess func(ctx context.Context, task *Task)
	RunFail    func(ctx context.Context, task *Task, err error)
}

func NewTaskGroup(ctx context.Context, taskNumber int) *TaskGroup {
	if ctx == nil {
		ctx = context.Background()
	}

	g := &TaskGroup{
		errorChan:     make(chan error),
		doneTaskCount: 0,
		taskNumber:    uint64(taskNumber),
		once:          &sync.Once{},
	}
	g.ctx, g.cancelFunc = context.WithCancel(ctx)

	return g
}

func (g *TaskGroup) Wait() error {
	select {
	case err, ok := <-g.errorChan:
		if ok {
			return err
		}
	case <-g.ctx.Done():
		return g.ctx.Err()
	}
	return nil
}

func (g *TaskGroup) Cancel() {
	g.cancelFunc()
}

func (g *TaskGroup) Fail(err error) {
	g.once.Do(func() {
		g.errorChan <- err
		g.Cancel()
		close(g.errorChan)
	})
}

func (g *TaskGroup) Chan() <-chan struct{} {
	return g.ctx.Done()
}

func (g *TaskGroup) done() {
	if newValue := atomic.AddUint64(&g.doneTaskCount, 1); newValue == g.taskNumber {
		close(g.errorChan)
	}
}

// beforeRun 运行之前的检查，主要用在检查是否需要继续执行task任务
func (g *TaskGroup) beforeRun(task *Task) bool {
	select {
	case <-g.ctx.Done():
		return false
	default:
	}
	if g.RunBefore != nil {
		return g.RunBefore(g.ctx, task)
	}
	return true
}

func (g *TaskGroup) afterRun(task *Task, err error) {
	if err == nil {
		if g.RunSuccess != nil {
			g.RunSuccess(g.ctx, task)
		}
		g.done()
		return
	}
	if g.RunFail != nil {
		g.RunFail(g.ctx, task, err)
	}
}
