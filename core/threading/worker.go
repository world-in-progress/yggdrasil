package threading

import (
	"fmt"
	"time"
)

type Worker struct {
	ID         string
	taskChan   chan ITask
	lastActive time.Time
}

func NewWorker(taskChan chan ITask, tokenChan chan struct{}, firstEntry ITask) *Worker {

	w := &Worker{
		taskChan:   taskChan,
		lastActive: time.Now(),
	}

	// start worker
	GoSafe(func() {
		// set timer
		idleTimeout := 30 * time.Second
		timer := time.NewTimer(idleTimeout)
		defer timer.Stop()

		if firstEntry != nil && !firstEntry.IsIgnoreable() {
			firstEntry.Process()
			firstEntry.Complete()
			firstEntry = nil // cut off reference
		}

		for {
			select {
			case task, ok := <-taskChan:
				if !ok {
					<-tokenChan // remove token
					return      // channel closed, shutdown
				}
				w.processTask(task)

			case <-timer.C:
				if time.Since(w.lastActive) >= 30*time.Second {
					<-tokenChan // remove token
					return      // idle timeout reached, shutdown
				}
				timer.Reset(idleTimeout)
			}
		}
	})
	return w
}

func (w *Worker) processTask(task ITask) {
	if task.IsIgnoreable() {
		fmt.Printf("Task (ID: %s) has been canceled or done\n", task.GetID())
		return
	}
	w.lastActive = time.Now()
	task.Process()
	task.Complete()
}
