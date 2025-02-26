package rescue

import (
	"context"
	"fmt"
)

// TODO: Log the error to a log file
func Recover(cleanups ...func()) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup()
		}
		fmt.Println("Recovered from panic:", r)
	}
}

// TODO: Log the error to a log file with context info
func RecoverCtx(ctx context.Context, cleanups ...func()) {
	if r := recover(); r != nil {
		for _, cleanup := range cleanups {
			cleanup()
		}
		fmt.Println("Recovered from panic:", r)
	}
}
