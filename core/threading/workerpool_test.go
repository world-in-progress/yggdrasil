package threading

import (
	"runtime"
	"strconv"
	"testing"
)

func BenchmarkWorkerPool(b *testing.B) {

	var taskNum int = 10000000

	bufferSize := runtime.NumCPU() * 1000
	maxWorkerNum := runtime.NumCPU() * 2
	wp := NewWorkerPool(maxWorkerNum, bufferSize, runtime.NumCPU()*2)
	for b.Loop() {
		wg.Add(taskNum)
		for taskIndex := range taskNum {
			wp.Submit(NewMockedWorkerTask(strconv.Itoa(taskIndex)))
		}
		wg.Wait()
	}
}
