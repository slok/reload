package reload_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/reload"
	"github.com/slok/reload/reloadmock"
)

type priorityMockReloader struct {
	priority int
	m        *reloadmock.Reloader
}

func TestManager(t *testing.T) {
	tests := map[string]struct {
		reloaders func() []priorityMockReloader
		triggerID string
		expErr    bool
	}{
		"Single reloader should be called with the expected trigger ID.": {
			reloaders: func() []priorityMockReloader {
				m1 := priorityMockReloader{0, &reloadmock.Reloader{}}
				m1.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)
				return []priorityMockReloader{m1}
			},
			triggerID: "test-id",
		},

		"Single reloader error should get the error.": {
			reloaders: func() []priorityMockReloader {
				m1 := priorityMockReloader{0, &reloadmock.Reloader{}}
				m1.m.On("Reload", mock.Anything, mock.Anything).Once().Return(fmt.Errorf("something"))
				return []priorityMockReloader{m1}
			},
			triggerID: "test-id",
			expErr:    true,
		},

		"Multiple reloaders should be called with the expected trigger ID.": {
			reloaders: func() []priorityMockReloader {
				m1 := priorityMockReloader{0, &reloadmock.Reloader{}}
				m1.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)

				m2 := priorityMockReloader{0, &reloadmock.Reloader{}}
				m2.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)

				m3 := priorityMockReloader{0, &reloadmock.Reloader{}}
				m3.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)
				return []priorityMockReloader{m1, m2, m3}
			},
			triggerID: "test-id",
		},

		"Multiple reloaders with different priority should be called with the expected trigger ID.": {
			reloaders: func() []priorityMockReloader {
				m1 := priorityMockReloader{2, &reloadmock.Reloader{}}
				m1.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)

				m2 := priorityMockReloader{0, &reloadmock.Reloader{}}
				m2.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)

				m3 := priorityMockReloader{1, &reloadmock.Reloader{}}
				m3.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)
				return []priorityMockReloader{m1, m2, m3}
			},
			triggerID: "test-id",
		},

		"Having multiple reloaders with different priority, if a lower priority errors, shouldn't call the next ones.": {
			reloaders: func() []priorityMockReloader {
				m1 := priorityMockReloader{10, &reloadmock.Reloader{}}
				m1.m.On("Reload", mock.Anything, "test-id").Once().Return(fmt.Errorf("something"))

				m2 := priorityMockReloader{4, &reloadmock.Reloader{}}
				m2.m.On("Reload", mock.Anything, "test-id").Once().Return(nil)

				m3 := priorityMockReloader{25, &reloadmock.Reloader{}}

				m4 := priorityMockReloader{20, &reloadmock.Reloader{}}

				m5 := priorityMockReloader{25, &reloadmock.Reloader{}}

				return []priorityMockReloader{m1, m2, m3, m4, m5}
			},
			triggerID: "test-id",
			expErr:    true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			reloaders := test.reloaders()

			// Prepare.
			m := reload.NewManager()
			for _, r := range reloaders {
				m.Add(r.priority, r.m)
			}
			triggerC := make(chan string)
			m.On(reload.NotifierFunc(func(context.Context) string {
				triggerID := <-triggerC
				return triggerID
			}))

			// Execute.
			ctx, cancel := context.WithCancel(context.Background())
			checksFinished := make(chan struct{})
			go func() {
				err := m.Run(ctx)

				// Check.
				if test.expErr {
					assert.Error(err)
				} else {
					assert.NoError(err)
				}

				for _, r := range reloaders {
					r.m.AssertExpectations(t)
				}

				close(checksFinished)
			}()

			// Release the trigger to start the execution and checks.
			triggerC <- test.triggerID

			// Wait for until the reloaders handle the trigger.
			// Then cancel the context in case the reloaders didn't
			// error.
			time.Sleep(10 * time.Millisecond)
			cancel()

			// Wait until everything has been checked.
			<-checksFinished
		})
	}
}
