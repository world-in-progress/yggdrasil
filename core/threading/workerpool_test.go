package threading

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
)

var (
	taskNum        int = 1000000
	bufferSize     int = runtime.NumCPU() * 1000
	maxWorkerNum   int = runtime.NumCPU() * 3
	spawnWorkerNum int = runtime.NumCPU() * 2
)

func BenchmarkWorkerPool(b *testing.B) {
	for b.Loop() {
		wg.Add(taskNum)
		wp := NewWorkerPool(spawnWorkerNum, maxWorkerNum, bufferSize)
		for taskIndex := range taskNum {
			wp.Submit(NewMockedWorkerTask(strconv.Itoa(taskIndex)))
		}
		wg.Wait()
		wp.Shutdown()
	}
}

func TestTaskCancel(t *testing.T) {
	wp := NewWorkerPool(spawnWorkerNum, maxWorkerNum, bufferSize)
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
