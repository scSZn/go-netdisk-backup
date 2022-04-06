package group

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

var (
	ErrorGoroutinePoolIsStopped = errors.New("pool is stopped")
	ErrorTaskInvalid            = errors.New("task can't be nil")
)

// RecoveryFunc panic时候的处理函数
// 返回值代表是否启动新的协程执行任务
type RecoveryFunc func(*MyGroup, interface{}, TaskInterface)

//func MyRecoveryFunc(g *MyGroup, errInterface interface{}, task *Task) {
//	logger := g.Logger.(*logrus.Logger)
//	logger.WithError(errors.Errorf("panic: %+v", errInterface)).WithField("taskName", task.Name).Errorf("task execute fail")
//}

type MyGroup struct {
	tasks chan TaskInterface // 任务队列
	size  int                // 并发数量

	ctx          context.Context    // 传入的context，如果没有传入，默认会有一个
	cancel       context.CancelFunc // 取消函数
	wg           *sync.WaitGroup    // 控制等待
	recoveryFunc RecoveryFunc       // 自定义的recover函数，如果不指定则不会影响协程池运行

	Once *sync.Once // 保证err只有一个
	Err  error      // 执行过程中保存错误

	stopOnce *sync.Once // 保证只关闭一次tasks channel和stopChan

	mux     *sync.RWMutex // 保护状态
	isClose bool          // 协程池是否已关闭

	errorStrategy ErrorStrategyInterface // 执行过程中出现错误如何解决，默认是保存错误并关闭协程池
}

func NewMyGroup(ctx context.Context, size int, queueSize int) *MyGroup {
	tasks := make(chan TaskInterface)
	if queueSize > 0 {
		tasks = make(chan TaskInterface, queueSize)
	}
	group := &MyGroup{
		size:          size,
		tasks:         tasks,
		wg:            &sync.WaitGroup{},
		Once:          &sync.Once{},
		stopOnce:      &sync.Once{},
		mux:           &sync.RWMutex{},
		errorStrategy: AbortStrategy{},
	}

	group.ctx, group.cancel = context.WithCancel(ctx)
	return group
}

// 启动协程池
func (g *MyGroup) Start() {
	for i := 0; i < g.size; i++ {
		go g.work()
	}
}
func (g *MyGroup) work() {
	for {
		select {
		case task, ok := <-g.tasks: // 从任务队列中获取任务，执行任务
			if !ok {
				continue
			}
			g.executeTask(task)
		case <-g.ctx.Done(): // 如果Context超时或者取消
			g.Stop()
			g.errorStrategy.ErrorDeal(g, g.ctx.Err(), nil)
		}
	}
}

// executeTask 执行任务
// 1. 执行任务
// 2. 如果任务返回为error，则执行
func (g *MyGroup) executeTask(task TaskInterface) {
	defer func() {
		g.wg.Done()
		// 捕获异常，如果有自定义recoveryFunc，就使用自定义异常恢复函数
		if errInterface := recover(); errInterface != nil {
			if g.recoveryFunc != nil {
				g.recoveryFunc(g, errInterface, task)
			}
		}
	}()
	err := task.Run(g.ctx)
	if err != nil {
		g.errorStrategy.ErrorDeal(g, err, task)
	}
}

// Submit 提交任务
// 1. 判断协程池是否关闭，如果关闭，则不提交任务
// 2. 如果协程池没有关闭，则加入任务列表
func (g *MyGroup) Submit(task TaskInterface) error {
	g.mux.RLock()
	if g.isClose {
		return ErrorGoroutinePoolIsStopped
	}

	// 如果task为nil，消除它
	if task == nil {
		return ErrorTaskInvalid
	}
	g.mux.RUnlock()

	g.wg.Add(1)
	select {
	case g.tasks <- task:
	case <-g.ctx.Done():
		g.Stop()
	}
	return nil
}

// Wait 等任务执行完成，返回error
func (g *MyGroup) Wait() error {
	g.wg.Wait()
	g.Stop()
	return g.Err
}

// Cancel 如果有Context Cancel函数，则调用
func (g *MyGroup) Cancel() {
	if g.cancel != nil {
		g.cancel()
	}
}

func (g *MyGroup) Context() context.Context {
	return g.ctx
}

// Stop 停止协程池
// 1. 修改状态
// 2. 关闭任务队列
// 3. 通知子协程退出
func (g *MyGroup) Stop() {
	// 必须先cancel，如果先关闭tasks通道，会导致AddItem的时候往一个关闭的通道塞数据，报错
	g.stopOnce.Do(func() {
		g.cancel()
		g.isClose = true
		close(g.tasks) // 保证仅关闭一次
	})
}

//func (g *MyGroup) WithLogger(logger interface{}) *MyGroup {
//	g.Logger = logger
//	return g
//}

func (g *MyGroup) WithContext(ctx context.Context) *MyGroup {
	g.ctx, g.cancel = context.WithCancel(ctx)
	return g
}

func (g *MyGroup) WithErrorStrategy(strategy ErrorStrategyInterface) *MyGroup {
	g.errorStrategy = strategy
	return g
}
