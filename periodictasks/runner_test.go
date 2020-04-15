package periodictasks

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestRunSingleTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := rand.Intn(50) + 1

	wg := sync.WaitGroup{}
	wg.Add(1)

	ts := []PeriodicTask{
		{
			Interval: time.Millisecond,
			Run: func(ctx context.Context) error {
				n--
				if n == 0 {
					wg.Done()
				}
				return nil
			},
		},
	}

	RunTasks(ctx, ts)
	wg.Wait()
}

func TestRunMultiTask(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n0 := rand.Intn(50) + 1
	n1 := rand.Intn(50) + 1

	wg0 := sync.WaitGroup{}
	wg0.Add(1)

	wg1 := sync.WaitGroup{}
	wg1.Add(1)

	ts := []PeriodicTask{
		{
			Interval: time.Millisecond,
			Run: func(ctx context.Context) error {
				n0--
				if n0 == 0 {
					wg0.Done()
				}
				return nil
			},
		},
		{
			Interval: time.Millisecond + 1,
			Run: func(ctx context.Context) error {
				n1--
				if n1 == 0 {
					wg1.Done()
				}
				return nil
			},
		},
	}

	RunTasks(ctx, ts)

	wg0.Wait()
	wg1.Wait()
}
