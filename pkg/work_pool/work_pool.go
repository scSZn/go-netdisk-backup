package work_pool

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
)

type workMode uint8

const (
	WorkModeFull workMode = iota
	WorkModeSlowStart
)

var (
	ErrorGoroutinePoolIsStopped = errors.New("pool is stopped")
	ErrorTaskInvalid            = errors.New("task can't be nil")
)

// RecoveryFunc panic时候的处理函数
type RecoveryFunc func(*WorkPool, interface{}, *Task)

type WorkPool struct {
	tasks              chan *Task    // 传递给worker的任务
	workerCount        int           // 并发数量
	queue              chan *Task    // 暂存队列
	signal             chan struct{} // 信号，将task从queue送到tasks
	currentSignalCount int64         // 当前的信号数量，主要作用是防止所有的worker都不工作了
	mode               workMode

	groupCtx     context.Context
	cancel       context.CancelFunc // 取消函数
	recoveryFunc RecoveryFunc       // 自定义的recover函数，当协程panic时会调用，如果不指定则不会影响协程池运行

	stopOnce *sync.Once // 保证只关闭一次tasks channel和stopChan

	mux     *sync.RWMutex // 保护状态
	isClose bool          // 协程池是否已关闭
}

// NewWorkPool 创建一个工作池
func NewWorkPool(ctx context.Context, workerCount, queueSize int, mode workMode) *WorkPool {
	p := &WorkPool{
		workerCount: workerCount,
		tasks:       make(chan *Task),
		queue:       make(chan *Task, queueSize),
		signal:      make(chan struct{}, 1),
		stopOnce:    &sync.Once{},
		mux:         &sync.RWMutex{},
		mode:        mode,
	}

	p.groupCtx, p.cancel = context.WithCancel(ctx)
	return p
}

// 启动协程池
func (p *WorkPool) Start() {
	for i := 0; i < p.workerCount; i++ {
		go p.work()
	}
	go func() {
		for {
			select {
			case <-p.signal:
				if task, ok := <-p.queue; ok {
					p.tasks <- task
				}
			case <-p.groupCtx.Done(): // 如果Context超时或者取消
				return
			}
		}
	}()
	switch p.mode {
	case WorkModeFull:
		for i := 0; i < p.workerCount; i++ {
			p.signal <- struct{}{}
		}
	case WorkModeSlowStart:
		p.signal <- struct{}{}
	}
}

func (p *WorkPool) work() {
	for {
		select {
		case task, ok := <-p.tasks: // 从任务队列中获取任务，执行任务
			if !ok {
				continue
			}
			p.addSignalCount(1)
			p.executeTask(task)
		case <-p.groupCtx.Done(): // 如果Context超时或者取消
			p.Stop()
		}
	}
}

// executeTask 执行任务
// 1. 执行任务
// 2. 如果任务返回为error，则执行
func (p *WorkPool) executeTask(task *Task) {
	var err error
	defer func() {
		p.addSignalCount(-1)
		if err != nil && p.mode == WorkModeSlowStart { // 慢启动模式下，会进行扣除
			if atomic.LoadInt64(&p.currentSignalCount) == 0 {
				p.wakeWorker(1)
			}
		} else { // 不是慢启动模式，会往signal中再启动一个task
			p.wakeWorker(1)
		}
		// 捕获异常，如果有自定义recoveryFunc，就使用自定义异常恢复函数
		if errInterface := recover(); errInterface != nil {
			if p.recoveryFunc != nil {
				p.recoveryFunc(p, errInterface, task)
			}
		}
	}()

	if !task.group.beforeRun(task) {
		return
	}

	err = task.run()
	task.group.afterRun(task, err)
	if err != nil {
		return
	}

	// 慢启动模式下，如果任务执行完成，会尝试再启动两个goroutine，前提是不超过pool中workerCount的数量
	if p.mode == WorkModeSlowStart {
		p.wakeWorker(1)
	}
}

func (p *WorkPool) wakeWorker(num int) {
	for i := 0; i < num; i++ {
		select {
		case p.signal <- struct{}{}:
		default:
		}
	}
}

// signal的计数器
func (p *WorkPool) addSignalCount(delta int64) int64 {
	return atomic.AddInt64(&p.currentSignalCount, delta)
}

// Submit 提交任务
// 1. 判断协程池是否关闭，如果关闭，则不提交任务
// 2. 如果协程池没有关闭，则加入任务列表
func (p *WorkPool) Submit(task *Task) error {
	p.mux.RLock()
	if p.isClose {
		return ErrorGoroutinePoolIsStopped
	}

	// 如果task为nil，消除它
	if task == nil || task.Run == nil {
		return ErrorTaskInvalid
	}
	p.mux.RUnlock()

	// 如果没有空闲的信号，尝试入队列
	select {
	case p.queue <- task:
	case <-p.groupCtx.Done():
		p.Stop()
	}
	return nil
}

// Cancel 如果有Context Cancel函数，则调用
func (p *WorkPool) Cancel() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *WorkPool) Context() context.Context {
	return p.groupCtx
}

// Stop 停止协程池
// 1. 修改状态
// 2. 关闭任务队列
// 3. 通知子协程退出
func (p *WorkPool) Stop() {
	// 必须先cancel，如果先关闭tasks通道，会导致AddItem的时候往一个关闭的通道塞数据，报错
	p.stopOnce.Do(func() {
		p.cancel()
		p.isClose = true
		close(p.tasks) // 保证仅关闭一次
	})
}

func (p *WorkPool) WithContext(ctx context.Context) *WorkPool {
	p.groupCtx, p.cancel = context.WithCancel(ctx)
	return p
}
