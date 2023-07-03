package rapidshorthair

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type Worker interface {
	Identity() string
	Status() bool
	Run(context.Context) error
	Downcall(interface{}) (interface{}, error)
}

type WorkerManager struct {
	name       string
	count      int32
	workerList []Worker
}

func NewWorkerManager(name string, count int, f func() (Worker, error)) (*WorkerManager, error) {
	workerList := make([]Worker, 0)

	if f != nil {
		for i := 0; i < int(count); i++ {
			w, err := f()
			if err != nil {
				return nil, err
			}

			workerList = append(workerList, w)
		}
	} else {
		count = 0
	}

	return &WorkerManager{
		name:       name,
		count:      int32(count),
		workerList: workerList,
	}, nil
}

func (m *WorkerManager) Add(w Worker) {
	m.workerList = append(m.workerList, w)
	atomic.AddInt32(&m.count, 1)
}

func (m *WorkerManager) Count() int32 {
	return atomic.LoadInt32(&m.count)
}

func (m *WorkerManager) GetWorker(id int32) (Worker, error) {
	if id >= m.count || id < 0 {
		return nil, fmt.Errorf("id invalid %v %v", id, m.count)
	}

	return m.workerList[id], nil
}

func (m *WorkerManager) Run(ctx context.Context, notifier chan error) error {
	ch := make(chan error, m.count)
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	wctx, cancel := context.WithCancel(context.Background())
	for _, w := range m.workerList {
		wg.Add(1)
		go func(w Worker, wg *sync.WaitGroup, ch chan error) {
			err := w.Run(wctx)
			ch <- err
			wg.Done()
		}(w, wg, ch)
	}

	finCount := 0

FOR:
	for {
		select {
		case <-ctx.Done():
			cancel()
			break FOR
		case err := <-ch:
			finCount += 1
			if err != nil {
				cancel()
				if notifier != nil {
					notifier <- err
				}
				return err
			}

			if int32(finCount) >= atomic.LoadInt32(&m.count) {
				cancel()
				if notifier != nil {
					notifier <- err
				}
				return nil
			}
		}
	}

	if notifier != nil {
		notifier <- nil
	}
	return nil
}
