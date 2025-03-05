package threading

import (
	"fmt"
)

type Worker struct {
	ID string
}

func NewWorker(workerID string, taskChan chan Task, firstEntry Task) *Worker {

	w := &Worker{
		ID: workerID,
	}

	// start worker
	GoSafe(func() {

		if firstEntry != nil && !firstEntry.IsIgnoreable() {
			firstEntry.Process()
			firstEntry.Complete()
			firstEntry = nil // cut off reference
		}

		for task := range taskChan {

			if task.IsIgnoreable() {
				fmt.Printf("Task (ID: %s) has been canceled or done\n", task.GetID())
				continue
			}
			task.Process()
			task.Complete()
		}
	})
	return w
}
