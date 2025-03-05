package threading

import "fmt"

type Worker struct {
	ID string
}

func NewWorker(workerID string, taskChan chan *TaskEntry, firstEntry *TaskEntry) *Worker {

	w := &Worker{
		ID: workerID,
	}

	// start worker
	GoSafe(func() {

		if firstEntry != nil && !firstEntry.IsIgnoreable() {
			firstEntry.task.Process()
			firstEntry.Complete()
			firstEntry = nil // cut off reference
		}

		for entry := range taskChan {
			if entry.IsIgnoreable() {
				fmt.Printf("Task (ID: %s) has been canceled or done\n", entry.task.GetID())
				continue
			}
			entry.task.Process()
			entry.Complete()
		}
	})
	return w
}
