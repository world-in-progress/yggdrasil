package threading

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type simpleTask struct {
	id   string
	data uint32
}

func (t *simpleTask) GetID() string   { return t.id }
func (t *simpleTask) SetID(id string) { t.id = id }
func (t *simpleTask) Handler(workerID int, taskID string) *TaskDoneMessage {

	for range 1000 {
		t.data = uint32(t.data) + 1
	}

	return &TaskDoneMessage{
		WorkerID: workerID,
		TaskID:   taskID,
		Result:   t.data,
		Err:      nil,
	}
}

func TestWorkerPool(t *testing.T) {
	wp := NewWorkerPool()
	wg := sync.WaitGroup{}

	taskNum := 1000000
	cancelNum := int(0)
	cbCount := new(uint32)
	dataSum := new(uint32)

	// task callback
	callback := func(message *TaskDoneMessage) {
		if message.Err != nil {
			fmt.Printf("Task_%s failed: %v\n", message.TaskID, message.Err)
			return
		}
		atomic.AddUint32(cbCount, 1)
		atomic.AddUint32(dataSum, message.Result.(uint32))
	}

	start := time.Now()
	for i := range taskNum {
		task := &simpleTask{data: uint32(i)}
		cancel, err := wp.Submit(task, callback)
		if err != nil {
			t.Fatalf("Submit failed: %v\n", err)
		}

		// test cancel
		if i%200 == 0 {
			cancel()
			cancelNum++
		}
	}
	fmt.Printf("Submitted %v tasks in %v\n", taskNum, time.Since(start))

	// wait for all tasks done
	wg.Add(1)
	GoSafe(func() {
		defer wg.Done()

		for {
			if int(atomic.LoadUint32(cbCount)) == taskNum-cancelNum {
				break
			}
		}
		fmt.Printf("Tasks completed in %v\nCallback count: %v\nData sum: %v\n", time.Since(start), *cbCount, *dataSum)
	})
	wg.Wait()

	wp.Shutdown()
}
