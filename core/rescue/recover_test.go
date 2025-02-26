package rescue

import (
	"testing"
)

func division(a float64, b float64) float64 {
	defer Recover()

	if b == 0 {
		panic("division by zero")
	}

	return a / b
}

func TestRecover(t *testing.T) {
	division(1, 0)
}
