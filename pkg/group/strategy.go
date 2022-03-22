package group

type ErrorStrategyInterface interface {
	ErrorDeal(*MyGroup, error, TaskInterface)
}

type AbortStrategy struct{}

func (s AbortStrategy) ErrorDeal(group *MyGroup, err error, task TaskInterface) {
	group.Once.Do(func() {
		group.Err = err
		group.Stop()
	})
}
