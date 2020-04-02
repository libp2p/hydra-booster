package periodictasks

import (
	"context"
	"fmt"
	"time"
)

// PeriodicTask describes a task that should be run periodically
type PeriodicTask struct {
	Interval time.Duration
	Run      func(ctx context.Context) error
}

// RunTasks immeidately begins to periodically run the passed tasks.
func RunTasks(ctx context.Context, tasks []PeriodicTask) {
	for _, task := range tasks {
		go func(t PeriodicTask) {
			timer := time.NewTimer(t.Interval)
			defer timer.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					err := t.Run(ctx)
					if err != nil {
						fmt.Println(fmt.Errorf("failed to run periodic task: %w", err))
					}
					timer.Reset(t.Interval)
				}
			}
		}(task)
	}
}
