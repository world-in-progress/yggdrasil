package threading

import (
	"context"

	"github.com/world-in-progress/yggdrasil/core/rescue"
)

// RunSafe runs the provided function recovers if function panics.
func RunSafe(fn func()) {
	defer rescue.Recover()

	fn()
}

// RunSafeCtx runs the provided function, recovers if function panics with context.
func RunSafeCtx(ctx context.Context, fn func()) {
	defer rescue.RecoverCtx(ctx)

	fn()
}

// GoSafe runs the provided function in a goroutine, recovers if function panics.
func GoSafe(fn func()) {
	go RunSafe(fn)
}

// GoSafeCtx runs the provided function in a goroutine, recovers if function panics with context.
func GoSafeCtx(ctx context.Context, fn func()) {
	go RunSafeCtx(ctx, fn)
}
