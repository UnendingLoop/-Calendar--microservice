package notifier_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
	"github.com/UnendingLoop/-Calendar--microservice/internal/notifier"
	"github.com/stretchr/testify/require"
)

type mockRep struct {
	// GetNextEventTime behavior
	nextTime            time.Time
	nextTimeErr         error
	nextTimeCallCounter int32

	// MarkEventDone behavior
	eventFound             bool
	eventMarkedCallCounter int32

	// PopNearestEvent behavior
	popNearestEvent            *model.Event
	popNearestEventErr         error
	popNearestEventCallCounter int32

	// Control behavior with counter
	popCounterThreshold int32 // threshold for returning error
	popCounter          int32
}

func (mr *mockRep) GetNextEventTime() (time.Time, error) {
	atomic.AddInt32(&mr.nextTimeCallCounter, 1)
	return mr.nextTime, mr.nextTimeErr
}

func (mr *mockRep) MarkEventDone(uid uint, eid string) bool {
	atomic.AddInt32(&mr.eventMarkedCallCounter, 1)
	return mr.eventFound
}

func (mr *mockRep) PopNearestEvent() (*model.Event, error) {
	atomic.AddInt32(&mr.popNearestEventCallCounter, 1)
	return mr.popNearestEvent, mr.popNearestEventErr
}

func TestRunNotifier(t *testing.T) {
	testCases := []struct {
		caseName              string
		mockSetup             func() *mockRep
		contextSetup          func() context.Context
		channelSetup          func() <-chan struct{}
		wantGetNextEventCalls int // minimum expected calls
		wantPopEventCalls     int // minimum expected calls
		wantMarkEventCalls    int // minimum expected calls
		maxRunDuration        time.Duration
	}{
		{
			caseName: "context cancelled before initialization completes",
			mockSetup: func() *mockRep {
				return &mockRep{
					nextTimeErr: errors.New("repo is empty"),
				}
			},
			contextSetup: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			channelSetup: func() <-chan struct{} {
				return make(chan struct{})
			},
			wantGetNextEventCalls: 1,
			wantPopEventCalls:     0,
			wantMarkEventCalls:    0,
			maxRunDuration:        100 * time.Millisecond,
		},
		{
			caseName: "closed update channel exits gracefully during initialization",
			mockSetup: func() *mockRep {
				return &mockRep{
					nextTimeErr: errors.New("repo is empty"),
				}
			},
			contextSetup: func() context.Context {
				return context.Background()
			},
			channelSetup: func() <-chan struct{} {
				ch := make(chan struct{})
				go func() {
					time.Sleep(30 * time.Millisecond)
					close(ch)
				}()
				return ch
			},
			wantGetNextEventCalls: 1,
			wantPopEventCalls:     0,
			wantMarkEventCalls:    0,
			maxRunDuration:        200 * time.Millisecond,
		},
		{
			caseName: "channel signal triggers retry during initialization",
			mockSetup: func() *mockRep {
				return &mockRep{
					nextTimeErr: errors.New("repo is empty"),
				}
			},
			contextSetup: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			channelSetup: func() <-chan struct{} {
				ch := make(chan struct{}, 2)
				ch <- struct{}{}
				time.AfterFunc(100*time.Millisecond, func() {
					ch <- struct{}{}
				})
				return ch
			},
			wantGetNextEventCalls: 2, // at least 1 initial + 1 after signal
			wantPopEventCalls:     0,
			wantMarkEventCalls:    0,
			maxRunDuration:        1 * time.Second,
		},
		{
			caseName: "PopNearestEvent error does not process event",
			mockSetup: func() *mockRep {
				return &mockRep{
					nextTime:           time.Now().UTC().Add(50 * time.Millisecond),
					nextTimeErr:        nil,
					popNearestEvent:    nil,
					popNearestEventErr: errors.New("db error"),
					eventFound:         false,
				}
			},
			contextSetup: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			channelSetup: func() <-chan struct{} {
				return make(chan struct{})
			},
			wantGetNextEventCalls: 1, // initial
			wantPopEventCalls:     1, // attempted but failed
			wantMarkEventCalls:    0, // not called due to pop error
			maxRunDuration:        1 * time.Second,
		},
		{
			caseName: "event marked as done even when not found in repo",
			mockSetup: func() *mockRep {
				return &mockRep{
					nextTime:           time.Now().UTC().Add(100 * time.Millisecond),
					nextTimeErr:        nil,
					eventFound:         false, // not found in repo
					popNearestEvent:    &model.Event{UID: 1, EID: "evt1", Description: "Test"},
					popNearestEventErr: nil,
				}
			},
			contextSetup: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			channelSetup: func() <-chan struct{} {
				return make(chan struct{})
			},
			wantGetNextEventCalls: 1, // initial
			wantPopEventCalls:     1, // event popped
			wantMarkEventCalls:    1, // attempted to mark as done
			maxRunDuration:        1 * time.Second,
		},
		{
			caseName: "update signal during main loop reschedules timer",
			mockSetup: func() *mockRep {
				return &mockRep{
					nextTime:           time.Now().UTC().Add(2 * time.Second),
					nextTimeErr:        nil,
					eventFound:         true,
					popNearestEvent:    &model.Event{UID: 99, EID: "delayed", Description: "Delayed Event"},
					popNearestEventErr: nil,
				}
			},
			contextSetup: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			channelSetup: func() <-chan struct{} {
				ch := make(chan struct{})
				go func() {
					time.Sleep(100 * time.Millisecond)
					ch <- struct{}{} // trigger reschedule
				}()
				return ch
			},
			wantGetNextEventCalls: 2, // init + after signal
			wantPopEventCalls:     0, // event not triggered yet
			wantMarkEventCalls:    0, // event not triggered yet
			maxRunDuration:        2 * time.Second,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.caseName, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			mock := tt.mockSetup()
			ctx := tt.contextSetup()
			ch := tt.channelSetup()

			// Start notifier
			notifier.RunNotifier(ctx, wg, mock, ch)

			// Wait for goroutine completion
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			// Assert completion within timeout
			start := time.Now()
			select {
			case <-done:
				duration := time.Since(start)
				require.LessOrEqual(t, duration, tt.maxRunDuration,
					"notifier execution exceeded max duration")
			case <-time.After(tt.maxRunDuration + 1*time.Second):
				t.Fatal("notifier did not exit within expected timeout")
			}

			// Verify minimum call counts for GetNextEventTime (may be called many times in hot loop)
			calls := atomic.LoadInt32(&mock.nextTimeCallCounter)
			require.GreaterOrEqual(t, int(calls), tt.wantGetNextEventCalls,
				"GetNextEventTime should be called at least %d times, got %d", tt.wantGetNextEventCalls, calls)

			// Verify exact call counts for PopNearestEvent (should only happen on timer fire)
			popCalls := atomic.LoadInt32(&mock.popNearestEventCallCounter)
			if tt.wantPopEventCalls > 0 {
				require.GreaterOrEqual(t, int(popCalls), tt.wantPopEventCalls,
					"PopNearestEvent should be called at least %d times, got %d", tt.wantPopEventCalls, popCalls)
			} else {
				require.Equal(t, int32(0), popCalls,
					"PopNearestEvent should not be called, got %d", popCalls)
			}

			// Verify exact call counts for MarkEventDone (depends on PopNearestEvent)
			markCalls := atomic.LoadInt32(&mock.eventMarkedCallCounter)
			if tt.wantMarkEventCalls > 0 {
				require.GreaterOrEqual(t, int(markCalls), tt.wantMarkEventCalls,
					"MarkEventDone should be called at least %d times, got %d", tt.wantMarkEventCalls, markCalls)
			} else {
				require.Equal(t, int32(0), markCalls,
					"MarkEventDone should not be called, got %d", markCalls)
			}
		})
	}
}
