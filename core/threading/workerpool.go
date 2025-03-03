package threading

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
)

type (
	// TaskDoneMessage is the structure to hold a task done message.
	TaskDoneMessage struct {
		WorkerID int
		TaskID   string
		Result   any
		Err      error
	}

	// Task is the interface for a worker task.
	Task interface {
		GetID() string   // allow worker pool to get the task ID
		SetID(id string) // allow worker pool to set the task ID
		Handler(workerID int, taskID string) *TaskDoneMessage
	}

	TaskCallbackFunc = func(*TaskDoneMessage)

	// WorkerPool is the structure for a worker pool.
	WorkerPool struct {
		wg   sync.WaitGroup
		quit chan struct{}

		callbacks sync.Map
		taskMu    sync.Mutex
		tasks     []*taskEntry
		taskChan  chan *taskEntry
	}

	// taskEntry is the structure to hold a task and its context.
	taskEntry struct {
		task   Task
		ctx    context.Context
		cancel context.CancelFunc
	}

	// callbackTask is the structure to hold a callback task.
	callbackTask struct {
		msg      *TaskDoneMessage
		callback TaskCallbackFunc
	}
)

// NewWorkerPool creates a worker pool with the provided number of workers.
func NewWorkerPool(workerCount ...int) *WorkerPool {
	// calculate worker count
	count := runtime.NumCPU() * 2
	if len(workerCount) > 0 {
		count = workerCount[0]
	}
	if count <= 0 {
		count = runtime.NumCPU() * 2
	}

	// init worker pool
	wp := &WorkerPool{
		quit:     make(chan struct{}),
		tasks:    make([]*taskEntry, 0, 10000),
		taskChan: make(chan *taskEntry, count),
	}

	// start worker pool
	wp.startTaskDispatcher(count)
	wp.startWorkers(count)
	return wp
}

// Submit submits a task to the worker pool. It is recommended to pass a pointer to the task
// (e.g., &simpleTask{}) to avoid unnecessary copies and improve memory efficiency.
func (wp *WorkerPool) Submit(t Task, callback TaskCallbackFunc) (cancel func(), err error) {
	// generate and set task ID
	taskID, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate task ID: %v", err)
	}
	t.SetID(taskID)

	// register callback
	if callback != nil {
		wp.callbacks.Store(taskID, callback)
	}

	// create task entry
	ctx, cancel := context.WithCancel(context.Background())
	entry := &taskEntry{
		task:   t,
		ctx:    ctx,
		cancel: cancel,
	}

	// add to task queue if worker pool is not shutting down
	wp.taskMu.Lock()
	if isClosed(wp.quit) {
		wp.taskMu.Unlock()
		return cancel, fmt.Errorf("worker pool is shutting down")
	}
	wp.tasks = append(wp.tasks, entry)
	wp.taskMu.Unlock()
	return cancel, nil
}

// Shutdown shuts down the worker pool.
func (wp *WorkerPool) Shutdown() {
	close(wp.quit)
	wp.taskMu.Lock()

	// cancel all tasks in task queue
	for _, entry := range wp.tasks {
		entry.cancel()
		wp.maybeAddCallbackTask(entry.task, &TaskDoneMessage{
			TaskID: entry.task.GetID(),
			Err:    fmt.Errorf("worker pool shutting down"),
		})
	}

	wp.tasks = nil
	wp.taskMu.Unlock()
	wp.wg.Wait()
	close(wp.taskChan)
}

func (wp *WorkerPool) startTaskDispatcher(workerCount int) {
	wp.wg.Add(1)
	GoSafe(func() {
		defer wp.wg.Done()
		for {
			wp.taskMu.Lock()

			// when no tasks can be executed, give up power of executive
			if len(wp.tasks) == 0 {
				if isClosed(wp.quit) {
					wp.taskMu.Unlock()
					return
				}
				wp.taskMu.Unlock()
				runtime.Gosched() // avoid busy waiting
				continue
			}

			// fetch tasks by batch
			batch := wp.tasks[:min(len(wp.tasks), workerCount)]
			wp.tasks = wp.tasks[len(batch):]
			wp.taskMu.Unlock()

			// notify workers to perform tasks
			for _, entry := range batch {
				select {
				case wp.taskChan <- entry:

				case <-wp.quit:
					// case when worker pool is shutting down
					wp.maybeAddCallbackTask(entry.task, &TaskDoneMessage{
						TaskID: entry.task.GetID(),
						Err:    fmt.Errorf("worker pool shutting down"),
					})
					return

				case <-entry.ctx.Done():
					// case when task has been canceled
					wp.maybeAddCallbackTask(entry.task, &TaskDoneMessage{
						TaskID: entry.task.GetID(),
						Err:    context.Canceled,
					})
				}
			}
		}
	})
}

func (wp *WorkerPool) startWorkers(workerCount int) {
	for workerID := range workerCount {
		wp.wg.Add(1)
		GoSafe(func() {
			defer wp.wg.Done()
			for {
				select {
				case <-wp.quit:
					return
				case entry, ok := <-wp.taskChan:
					if !ok { // taskChan is closed
						return
					}
					// make task done message
					msg := wp.executeTask(workerID, entry)
					wp.maybeAddCallbackTask(entry.task, msg)
				}
			}
		})
	}
}

func (wp *WorkerPool) executeTask(workerID int, entry *taskEntry) *TaskDoneMessage {
	select {
	case <-entry.ctx.Done():
		return &TaskDoneMessage{
			WorkerID: workerID,
			TaskID:   entry.task.GetID(),
			Err:      context.Canceled,
		}
	default:
		return entry.task.Handler(workerID, entry.task.GetID())
	}
}

func (wp *WorkerPool) maybeAddCallbackTask(t Task, msg *TaskDoneMessage) {
	if _, isCallback := t.(*callbackTask); isCallback {
		return
	}
	if cb, loaded := wp.callbacks.LoadAndDelete(t.GetID()); loaded {
		wp.taskMu.Lock()
		if !isClosed(wp.quit) {
			wp.tasks = append(wp.tasks, &taskEntry{
				task: &callbackTask{
					callback: cb.(TaskCallbackFunc),
					msg:      msg,
				},
				ctx:    context.Background(),
				cancel: func() {},
			})
		}
		wp.taskMu.Unlock()
	}
}

func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func generateID() (string, error) {
	n := rand.Uint64() % 1e18
	id := strconv.FormatUint(n, 36)
	if len(id) > 10 {
		id = id[:10]
	}
	return id, nil
}

func (ct *callbackTask) GetID() string   { return ct.msg.TaskID }
func (ct *callbackTask) SetID(id string) { /* ID cannot be set in a callback task */ }
func (ct *callbackTask) Handler(workerID int, taskID string) *TaskDoneMessage {
	ct.callback(ct.msg)
	return nil // callback task will not return new done message
}
