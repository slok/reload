package reload

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"
)

type reloaderGroup struct {
	priority  int
	reloaders []Reloader
}

// NewManager returns a new manager.
func NewManager() Manager {
	return Manager{
		reloaders: map[int]reloaderGroup{},
	}
}

// Manager handles the reload mechanism.
// The will be listening to the trigger of any of the notifiers,
// when this process is triggered it will call to all the reloaders
// based on the priority groups.
type Manager struct {
	reloaders map[int]reloaderGroup
	notifiers []Notifier
	lock      uint32 // Mutex based on atomic integer.
}

// On registers a notifier that will execute all reloaders when
// any of the notifiers returns.
//
// The notifier should stay waiting until the reload needs to take place.
// The notifier should be able to be called multiple times.
//
// When a notifier ends its execution triggering the reload process
// the notifier will start again and keep waiting along with the ones
// already waiting.
//
// This process will be repeated forever until the manager stops.
func (m *Manager) On(n Notifier) {
	m.notifiers = append(m.notifiers, n)
}

// Add a reloader to the manager.
//
// The reloader will be called when any of the notifiers end the execution.
//
// When adding a reloader, the reloader will have a priority. All the reloaders
// with the same priority will be batched and executed in parallel. When the
// reloaders batch executing ends, if there is not error, it will execute the next
// priority batch. This pricess will continue until all priority batches have been
// executed.
//
// The priority order is ascendant (e.g 0, 42, 100, 250, 999...).
func (m *Manager) Add(priority int, r Reloader) {
	rg, ok := m.reloaders[priority]
	if !ok {
		rg = reloaderGroup{priority: priority}
	}
	rg.reloaders = append(rg.reloaders, r)
	m.reloaders[priority] = rg
}

type notifierResult struct {
	Result string
	Err    error
}

// Run will start the manager. This starts all the notifiers and wait until
// any of them returns a result, then it will call the notifiers in priority
// batches. All the triggered notifiers will start again.
//
// If any of the notifiers returns an error, the execution will end with
// an error.
//
// If the context is cancelled, the manager Run will end without error.
// If any of the reloaders reload process ends with an error, run will
// end its execution and return an error.
func (m *Manager) Run(ctx context.Context) error {
	signal := make(chan notifierResult, len(m.notifiers))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // This will stop all running notifiers.

	// Run all notifiers and wait for any of them sends a signal signals.
	for _, n := range m.notifiers {
		go func(n Notifier) {
			// Prepare notifier to be executed and map results to
			// our internal notification result.
			fn := func(ctx context.Context) notifierResult {
				res, err := n.Notify(ctx)
				return notifierResult{Result: res, Err: err}
			}
			// Notifiers will rerun once they end executing and
			// notify. This will be forever or until the context
			// ends.
			for {
				select {
				case signal <- fn(ctx):
				case <-ctx.Done():
					return // End notifier.
				}
			}
		}(n)
	}

	// Wait until the context ends or we receive a signal from
	// the first notifier, then stop all the other notifiers we
	// are waiting for.
	for {
		select {
		case notifierSignal := <-signal:
			// If signal has an error then stop everything.
			if notifierSignal.Err != nil {
				return fmt.Errorf("notifier failed: %w", notifierSignal.Err)
			}

			// Start reload process..
			err := m.reloadGroups(ctx, notifierSignal.Result)
			if err != nil {
				return fmt.Errorf("reload process failed: %w", err)
			}
		case <-ctx.Done():
			// We need to end.
			return nil
		}
	}
}

const (
	unlockedState uint32 = 0
	lockedState   uint32 = 1
)

// reloadGroups will start the reload process on all the
// reloaders and will wait until all have finished.
//
// While the reload process is being executed, if any other
// reload start trigger happens, it will be ignored.
//
// If any of the reloaders returns an error, it will automatically
// stop the reload process and end with an error.
//
// Reload process can be triggered any number of times.
func (m *Manager) reloadGroups(ctx context.Context, id string) error {
	if len(m.reloaders) == 0 {
		return nil
	}

	// Are we already in a reload process?
	if !atomic.CompareAndSwapUint32(&m.lock, unlockedState, lockedState) {
		return nil
	}
	defer atomic.StoreUint32(&m.lock, unlockedState)

	// Sort groups.
	reloderGroups := make([]reloaderGroup, 0, len(m.reloaders))
	for _, rg := range m.reloaders {
		reloderGroups = append(reloderGroups, rg)
	}
	sort.SliceStable(reloderGroups, func(x, y int) bool { return reloderGroups[x].priority < reloderGroups[y].priority })

	// Reload all groups secuentially.
	for _, rg := range reloderGroups {
		err := m.reloadGroup(ctx, rg, id)
		if err != nil {
			return fmt.Errorf("error on priority %d group reload: %w", rg.priority, err)
		}
	}

	return nil
}

func (m *Manager) reloadGroup(ctx context.Context, rg reloaderGroup, id string) error {
	reloaders := rg.reloaders

	errors := make(chan error, len(reloaders))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // This will stop all running goroutines.
	for _, r := range reloaders {
		go func(r Reloader) {
			// Wait until we finish reloading or we have signaled to stop.
			select {
			case errors <- r.Reload(ctx, id):
			case <-ctx.Done():
			}
		}(r)
	}

	// Wait until all have been reloaded or we receive an error.
	for i := 0; i < len(reloaders); i++ {
		err := <-errors
		if err != nil {
			return err
		}
	}

	return nil
}
