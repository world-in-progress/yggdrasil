package threading

import (
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

var wg sync.WaitGroup
var dataSum = new(atomic.Uint32)
var taskCount = new(atomic.Uint32)

type mockedWorkerTask struct {
	BaseTask
	data uint32
}

func NewMockedWorkerTask(taskID string) ITask {
	return &mockedWorkerTask{
		BaseTask: BaseTask{
			ID: taskID,
		},
	}
}

func (mwt *mockedWorkerTask) Process() error {
	defer wg.Done()

	for range 1000 {
		mwt.data++
	}

	taskCount.Add(1)
	dataSum.Add(mwt.data)

	return nil
}

func BenchmarkWorker(b *testing.B) {

	var taskNum int = 10000000

	if useWorker := false; useWorker {

		// benchmark for workers
		workerNum := runtime.NumCPU() * 2
		workers := make([]*Worker, workerNum)
		taskChan := make(chan ITask, workerNum*100)
		for index := range workerNum {
			workers[index] = NewWorker(strconv.Itoa(index), taskChan, nil)
		}

		for b.Loop() {
			wg.Add(taskNum)
			for taskIndex := range taskNum {
				taskChan <- NewMockedWorkerTask(strconv.Itoa(taskIndex))
			}
			wg.Wait()
		}
	} else {
		// benchmark for pure goroutines
		for b.Loop() {
			wg.Add(taskNum)
			for taskIndex := range taskNum {
				task := NewMockedWorkerTask(strconv.Itoa(taskIndex))
				go func() {
					task.Process()
				}()
			}
			wg.Wait()
		}
	}
}
