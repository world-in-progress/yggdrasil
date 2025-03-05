package threading

type Worker struct {
	ID string
}

func NewWorker(workerID string, taskChan chan *TaskEntry, firstEntry *TaskEntry) *Worker {

	w := &Worker{
		ID: workerID,
	}

	// start worker
	GoSafe(func() {

		if firstEntry != nil && !firstEntry.IsCancelled() {
			firstEntry.task.Handler()
		}

		for entry := range taskChan {
			if entry.IsCancelled() {
				return
			}
			entry.task.Handler()
		}
	})
	return w
}
