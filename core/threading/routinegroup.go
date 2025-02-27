package threading

import "sync"

// RoutineGroup is a group of routines, where all routines will be waited for before the group is considered done.
type RoutineGroup struct {
	waitGroup sync.WaitGroup
}

// NewRoutineGroup creates a new RoutineGroup.
func NewRoutineGroup() *RoutineGroup {
	return &RoutineGroup{}
}

// Run runs the provided fn in a RoutineGroup.
func (g *RoutineGroup) Run(fn func()) {
	g.waitGroup.Add(1)

	go func() {
		defer g.waitGroup.Done()
		fn()
	}()
}

// RunSafe runs the provided fn in a RoutineGroup, recovers if function panics.
func (g *RoutineGroup) RunSafe(fn func()) {
	g.waitGroup.Add(1)

	GoSafe(func() {
		defer g.waitGroup.Done()
		fn()
	})
}

// Wait waits all routines in the RoutineGroup to finish.
func (g *RoutineGroup) Wait() {
	g.waitGroup.Wait()
}
