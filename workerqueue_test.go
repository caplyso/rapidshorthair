package rapidshorthair

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type TestWorker struct {
	id string
}

func NewTestWorker() (Worker, error) {
	return &TestWorker{
		id: "testWorker",
	}, nil
}

func (w *TestWorker) Identity() string {
	return w.id
}
func (w *TestWorker) Status() bool {
	return true
}
func (w *TestWorker) Run(ctx context.Context) error {
	for {
		<-ctx.Done()
		break
	}
	return nil
}

func (w *TestWorker) Downcall(interface{}) (interface{}, error) {
	return nil, nil
}

func TestWorkerManagerCreate(t *testing.T) {
	wm, err := NewWorkerManager("testManager", 2, NewTestWorker)
	require.NotEqual(t, wm, nil, "")
	require.Equal(t, err, nil, "")

	w1 := &TestWorker{
		id: "testWorker1",
	}
	wm.Add(w1)

	count := wm.Count()
	require.Equal(t, count, int32(3), "")

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func(ctx context.Context, ch chan error) {
		err := wm.Run(ctx, ch)
		require.Equal(t, err, nil, "")
	}(ctx, ch)

	time.Sleep(1 * time.Second)
	cancel()
}
