package threading

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

var (
	taskNum      int = 1000000
	bufferSize   int = runtime.NumCPU() * 1000
	minWorkerNum int = runtime.NumCPU() * 2
	maxWorkerNum int = runtime.NumCPU() * 3
)

func BenchmarkWorkerPool(b *testing.B) {
	for b.Loop() {
		wg.Add(taskNum)
		wp := NewWorkerPool(minWorkerNum, maxWorkerNum, bufferSize)
		for taskIndex := range taskNum {
			wp.Submit(NewMockedWorkerTask(strconv.Itoa(taskIndex)))
		}
		wg.Wait()
		wp.Shutdown()
	}
}

func TestTaskCancel(t *testing.T) {
	wp := NewWorkerPool(minWorkerNum, maxWorkerNum, bufferSize)
	wg.Add(taskNum)
	for taskIndex := range taskNum {
		cancel, _ := wp.Submit(NewMockedWorkerTask(strconv.Itoa(taskIndex)))
		if taskIndex%200000 == 0 {
			if cancel() {
				wg.Done()
			} else {
				fmt.Printf("Task (ID: %s) has been done\n", strconv.Itoa(taskIndex))
			}
		}
	}
	wg.Wait()
	wp.Shutdown()
}

func TestWorkerTimeout(t *testing.T) {

	wp := NewWorkerPool(minWorkerNum, maxWorkerNum, bufferSize)
	defer wp.Shutdown()

	fmt.Printf("Current worker count is %d\n", wp.GetWorkerCount())

	var wg sync.WaitGroup

	// submit initial tasks to activate workers
	wg.Add(minWorkerNum)
	for i := range minWorkerNum {
		task := NewMockTerminateTask("test-"+string(rune(i)), &wg)
		wp.Submit(task)
	}
	wg.Wait()

	// get initial worker count
	workerCount := wp.GetWorkerCount()
	fmt.Printf("Current worker count is %d\n", workerCount)

	// wait for timeout period plus some buffer
	timeoutDuration := 30*time.Second + 2*time.Second
	time.Sleep(timeoutDuration)

	// check if workers have shut down
	workerCount = wp.GetWorkerCount()

	// since no new tasks were submitted and timeout has passed
	// all workers should have shut down
	if workerCount != 0 {
		t.Errorf("Expected 0 workers after timeout, got %d\n", workerCount)
	} else {
		fmt.Printf("Current worker count is %d\n", wp.GetWorkerCount())
	}
}

type mockTerminateTask struct {
	BaseTask
	wg *sync.WaitGroup
}

func NewMockTerminateTask(id string, wg *sync.WaitGroup) *mockTerminateTask {
	return &mockTerminateTask{
		BaseTask: BaseTask{
			ID: id,
		},
		wg: wg,
	}
}

func (m *mockTerminateTask) Process() {
	m.wg.Done()
}
