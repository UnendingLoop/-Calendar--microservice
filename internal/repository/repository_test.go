package repository

import (
	"container/heap"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"

	"github.com/stretchr/testify/require"
)

func TestNewEventRepository(t *testing.T) {
	type mock struct {
		name       string
		mockEmap   map[uint][]model.HeapEntity
		mockArch   map[uint][]model.Event
		mockCh     chan struct{}
		wantErr    bool
		wantSecMap *SecureEventsMap
		heapLen    int
	}

	cases := []mock{
		func() mock {
			ch := make(chan struct{})

			return mock{
				name:     "Nil emap and arch",
				mockEmap: nil,
				mockArch: nil,
				mockCh:   ch,
				wantErr:  false,
				wantSecMap: &SecureEventsMap{
					eventMap: map[uint][]model.HeapEntity{},
					archive:  map[uint][]model.Event{},
					eh:       model.EventHeap{},
					updateCh: ch,
				},
				heapLen: 0}
		}(),
		{
			name:       "Nil channel",
			mockEmap:   nil,
			mockArch:   nil,
			mockCh:     nil,
			wantErr:    true,
			wantSecMap: nil,
			heapLen:    0},
		func() mock {
			ch := make(chan struct{})
			ct := model.CustomTime{}
			ct.Time = time.Now()

			emap := map[uint][]model.HeapEntity{
				1: {
					{Event: &model.Event{Scheduled: &ct}},
					{Event: &model.Event{Scheduled: &ct}},
				},
				2: {
					{Event: &model.Event{Scheduled: &ct}},
					{Event: &model.Event{Scheduled: &ct}},
				}}
			amap := make(map[uint][]model.Event)

			hl := 0
			for _, v := range emap {
				hl += len(v)
			}

			return mock{
				name:     "Positive - no nils in input",
				mockEmap: emap,
				mockArch: amap,
				mockCh:   ch,
				wantErr:  false,
				wantSecMap: &SecureEventsMap{
					eventMap: emap,
					archive:  amap,
					eh:       model.EventHeap{},
					updateCh: ch,
				},
				heapLen: hl}
		}(),
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := NewEventRepository(tt.mockCh, tt.mockEmap, tt.mockArch)
			if tt.wantErr {
				require.NotEqual(t, nil, err, "received nil, but expected error")
				return
			}

			require.Equal(t, tt.heapLen, len(res.eh), fmt.Sprintf("heap size is %d instead of expected %d", len(res.eh), tt.heapLen))
			require.Equal(t, tt.wantSecMap.archive, res.archive)
			require.Equal(t, tt.wantSecMap.eventMap, res.eventMap)
			require.Equal(t, tt.wantSecMap.updateCh, res.updateCh)

		})

	}
}
func newMockSecureMap() *SecureEventsMap {
	return &SecureEventsMap{
		eventMap: map[uint][]model.HeapEntity{},
		archive:  map[uint][]model.Event{},
		eh:       model.EventHeap{},
		mu:       sync.RWMutex{},
		updateCh: make(chan<- struct{}, 1),
	}
}

func TestCreateEvent(t *testing.T) {
	cases := []struct {
		name      string
		mockRepo  *SecureEventsMap
		mockEvent model.Event
		diff      int
	}{
		{
			name:     "creating event",
			mockRepo: newMockSecureMap(),
			mockEvent: func() model.Event {
				ct := model.CustomTime{}
				ct.Time = time.Now()
				return model.Event{Scheduled: &ct}
			}(),
			diff: 1,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			before := len(tt.mockRepo.eh)
			tt.mockRepo.CreateEvent(tt.mockEvent)
			after := len(tt.mockRepo.eh)

			require.Equal(t, tt.diff, after-before)

		})
	}
}

func TestUpdateEvent(t *testing.T) {
	cases := []struct {
		name     string
		mockRepo *SecureEventsMap
		updEvent model.Event
		wantRes  *model.Event
	}{
		{name: "pos - time updated",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Date(2000, time.January, 1, 1, 1, 1, 0, time.Local)

				mockMap := newMockSecureMap()
				mockMap.CreateEvent(model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "some description",
				})
				return mockMap
			}(),

			updEvent: model.Event{
				EID: "someUUID",
				UID: 300,
				Scheduled: func() *model.CustomTime {
					ct := model.CustomTime{}
					ct.Time = time.Date(3000, time.January, 1, 1, 1, 1, 0, time.Local)
					return &ct
				}(),
			},
			wantRes: func() *model.Event {
				ct := model.CustomTime{}
				ct.Time = time.Date(3000, time.January, 1, 1, 1, 1, 0, time.Local)
				return &model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "some description",
				}
			}(),
		},
		{name: "pos - descr updated",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Date(2000, time.January, 1, 1, 1, 1, 0, time.Local)

				mockMap := newMockSecureMap()
				mockMap.CreateEvent(model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "some description",
				})
				return mockMap
			}(),

			updEvent: model.Event{
				EID:         "someUUID",
				UID:         300,
				Description: "new description",
			},
			wantRes: func() *model.Event {
				ct := model.CustomTime{}
				ct.Time = time.Date(2000, time.January, 1, 1, 1, 1, 0, time.Local)
				return &model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "new description",
				}
			}(),
		},
		{name: "pos - time & descr updated",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Date(2000, time.January, 1, 1, 1, 1, 0, time.Local)

				mockMap := newMockSecureMap()
				mockMap.CreateEvent(model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "some description",
				})
				return mockMap
			}(),

			updEvent: model.Event{
				EID:         "someUUID",
				UID:         300,
				Description: "new description",
				Scheduled: func() *model.CustomTime {
					ct := model.CustomTime{}
					ct.Time = time.Date(3000, time.January, 1, 1, 1, 1, 0, time.Local)
					return &ct
				}(),
			},
			wantRes: func() *model.Event {
				ct := model.CustomTime{}
				ct.Time = time.Date(3000, time.January, 1, 1, 1, 1, 0, time.Local)
				return &model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "new description",
				}
			}(),
		},
		{name: "neg - user not found",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Date(2000, time.January, 1, 1, 1, 1, 0, time.Local)

				mockMap := newMockSecureMap()
				mockMap.CreateEvent(model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "some description",
				})
				return mockMap
			}(),

			updEvent: model.Event{
				EID: "someUUID",
				UID: 404,
				Scheduled: func() *model.CustomTime {
					ct := model.CustomTime{}
					ct.Time = time.Date(3000, time.January, 1, 1, 1, 1, 0, time.Local)
					return &ct
				}(),
			},
			wantRes: nil,
		},
		{name: "neg - event not found",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Date(2000, time.January, 1, 1, 1, 1, 0, time.Local)

				mockMap := newMockSecureMap()
				mockMap.CreateEvent(model.Event{
					EID:         "someUUID",
					UID:         300,
					Scheduled:   &ct,
					Description: "some description",
				})
				return mockMap
			}(),

			updEvent: model.Event{
				EID: "otherUUID",
				UID: 300,
				Scheduled: func() *model.CustomTime {
					ct := model.CustomTime{}
					ct.Time = time.Date(3000, time.January, 1, 1, 1, 1, 0, time.Local)
					return &ct
				}(),
			},
			wantRes: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.mockRepo.UpdateEvent(tt.updEvent.UID, tt.updEvent)

			if tt.wantRes == nil {
				require.Equal(t, tt.wantRes, res)
				return
			}
			require.Equal(t, tt.wantRes.Description, res.Description, fmt.Sprintf("Description expected: %q, got %q", tt.wantRes.Description, res.Description))
			require.Equal(t, tt.wantRes.EID, res.EID, fmt.Sprintf("EID expected: %q, got %q", tt.wantRes.EID, res.EID))
			require.Equal(t, tt.wantRes.UID, res.UID, fmt.Sprintf("UID expected: %d, got %d", tt.wantRes.UID, res.UID))
			require.Equal(t, tt.wantRes.Scheduled, res.Scheduled, fmt.Sprintf("Scheduled expected: %v, got %v", tt.wantRes.Scheduled, res.Scheduled))
		})
	}
}

func TestDeleteEvent(t *testing.T) {
	cases := []struct {
		name     string
		mockRepo *SecureEventsMap
		eid      string
		uid      uint
		wantRes  bool
	}{
		{name: "pos - delete success main map",
			mockRepo: func() *SecureEventsMap {
				mockHeap := model.EventHeap{}
				heap.Init(&mockHeap)
				ct := model.CustomTime{}
				ct.Time = time.Now()
				mockEvent := model.Event{
					EID:       "someUUID",
					UID:       300,
					Scheduled: &ct,
				}

				mockRepo := SecureEventsMap{
					eventMap: map[uint][]model.HeapEntity{},
					archive:  map[uint][]model.Event{},
					eh:       mockHeap,
					updateCh: make(chan<- struct{}, 1),
					mu:       sync.RWMutex{},
				}
				mockRepo.CreateEvent(mockEvent)
				return &mockRepo
			}(),
			eid:     "someUUID",
			uid:     300,
			wantRes: true,
		},
		{name: "pos - delete success archive",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Now()
				mockEvent := model.Event{
					EID:       "someUUID",
					UID:       300,
					Scheduled: &ct,
				}
				mockRepo := SecureEventsMap{
					eventMap: map[uint][]model.HeapEntity{},
					archive:  map[uint][]model.Event{300: {mockEvent}},
					eh:       model.EventHeap{},
					updateCh: make(chan<- struct{}, 1),
					mu:       sync.RWMutex{},
				}
				return &mockRepo
			}(),
			eid:     "someUUID",
			uid:     300,
			wantRes: true,
		},
		{name: "neg - UID not exist",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Now()
				mockEvent := model.Event{
					EID:       "someUUID",
					UID:       404,
					Scheduled: &ct,
				}
				mockRepo := SecureEventsMap{
					eventMap: map[uint][]model.HeapEntity{},
					archive:  map[uint][]model.Event{404: {mockEvent}},
					eh:       model.EventHeap{},
					updateCh: make(chan<- struct{}),
					mu:       sync.RWMutex{},
				}
				return &mockRepo
			}(),
			eid:     "someUUID",
			uid:     300,
			wantRes: false,
		},
		{name: "neg - EID not exist",
			mockRepo: func() *SecureEventsMap {
				ct := model.CustomTime{}
				ct.Time = time.Now()
				mockEvent := model.Event{
					EID:       "someUUID",
					UID:       300,
					Scheduled: &ct,
				}
				mockRepo := SecureEventsMap{
					eventMap: map[uint][]model.HeapEntity{},
					archive:  map[uint][]model.Event{300: {mockEvent}},
					eh:       model.EventHeap{},
					updateCh: make(chan<- struct{}),
					mu:       sync.RWMutex{},
				}
				return &mockRepo
			}(),
			eid:     "404UUID",
			uid:     300,
			wantRes: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.mockRepo.DeleteEvent(tt.uid, tt.eid)
			require.Equal(t, tt.wantRes, res)
		})
	}

}

func TestGetPeriodEvents(t *testing.T) {
	mockRepoBuilder := func() *SecureEventsMap {
		timePtr := func(t time.Time) *model.CustomTime {
			ct := model.CustomTime{}
			ct.Time = t
			return &ct
		}
		currentTime := time.Now().UTC()
		evMap := map[uint][]model.HeapEntity{300: {
			{Index: 0, Event: &model.Event{
				EID:       "someUUID_0",
				UID:       300,
				Scheduled: timePtr(currentTime.AddDate(0, 0, 1)),
			}},
			{Index: 1, Event: &model.Event{
				EID:       "someUUID_1",
				UID:       300,
				Scheduled: timePtr(currentTime.AddDate(0, 0, 2)),
			}},
			{Index: 2, Event: &model.Event{
				EID:       "someUUID_2",
				UID:       404,
				Scheduled: timePtr(currentTime.AddDate(0, 0, 3)),
			}},
		}}
		archMap := map[uint][]model.Event{300: {
			{
				EID:       "someUUID_0",
				UID:       300,
				Scheduled: timePtr(currentTime.AddDate(0, 0, -1)),
			},
			{
				EID:       "someUUID_1",
				UID:       300,
				Scheduled: timePtr(currentTime.AddDate(0, 0, -2)),
			},
			{
				EID:       "someUUID_2",
				UID:       404,
				Scheduled: timePtr(currentTime.AddDate(0, 0, -3)),
			},
		}}

		mr := &SecureEventsMap{
			eventMap: evMap,
			archive:  archMap,
			eh:       nil, //в этом методе куча не участвует
			updateCh: nil, // как и канал
			mu:       sync.RWMutex{},
		}
		return mr
	}
	cases := []struct {
		name     string
		mockRepo *SecureEventsMap
		start    time.Time
		end      time.Time
		uid      uint
		wantResN int
	}{
		{name: "pos - future events",
			mockRepo: mockRepoBuilder(),
			uid:      300,
			start:    time.Now().UTC(),
			end:      time.Now().UTC().AddDate(0, 0, 3),
			wantResN: 2},
		{name: "pos - past events",
			mockRepo: mockRepoBuilder(),
			uid:      300,
			start:    time.Now().UTC().AddDate(0, 0, -3),
			end:      time.Now().UTC(),
			wantResN: 2},
		{name: "neg - uid not found",
			mockRepo: mockRepoBuilder(),
			uid:      666,
			start:    time.Now().UTC().AddDate(0, 0, 3),
			end:      time.Now().UTC().AddDate(0, 1, 0),
			wantResN: 0},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.mockRepo.GetPeriodEvents(tt.uid, tt.start, tt.end)
			if tt.wantResN == 0 {
				require.Nil(t, res, fmt.Sprintf("Result slice is not nil: %v", res))
			} else {
				require.Equal(t, tt.wantResN, len(res), fmt.Sprintf("Expected slice len is %d, but got %d", tt.wantResN, len(res)))
			}
		})
	}
}

func TestArchiveExpired(t *testing.T) {
	mockRepoBuilder := func() *SecureEventsMap {
		evMap := map[uint][]model.HeapEntity{300: {
			{Index: 0, Event: &model.Event{
				IsDone: true,
			}},
			{Index: 1, Event: &model.Event{
				IsDone: true,
			}},
			{Index: 2, Event: &model.Event{
				IsDone: false,
			}},
		}}

		mr := &SecureEventsMap{
			eventMap: evMap,
			archive:  map[uint][]model.Event{},
			eh:       nil, //в этом методе куча не участвует
			updateCh: nil, // как и канал
			mu:       sync.RWMutex{},
		}
		return mr
	}
	cases := []struct {
		name     string
		mockRepo *SecureEventsMap
		wantResN int
	}{
		{name: "pos - 2 archived events",
			mockRepo: mockRepoBuilder(),
			wantResN: 2},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.mockRepo.ArchiveExpired()
			require.Equal(t, tt.wantResN, res, fmt.Sprintf("Expected N is %d, but got %d", tt.wantResN, res))
			require.Equal(t, tt.wantResN, len(tt.mockRepo.archive[300]), fmt.Sprintf("Expected N of events in archive is %d, but got %d", tt.wantResN, len(tt.mockRepo.archive[300])))
		})
	}
}

func TestMarkEventDone(t *testing.T) {
	mockRepoBuilder := func() *SecureEventsMap {
		evMap := map[uint][]model.HeapEntity{300: {
			{Index: 0, Event: &model.Event{
				EID:    "someUUID_0",
				UID:    300,
				IsDone: false,
			}},
			{Index: 1, Event: &model.Event{
				EID:    "someUUID_1",
				UID:    300,
				IsDone: true,
			}},
			{Index: 2, Event: &model.Event{
				EID:    "someUUID_2",
				UID:    300,
				IsDone: false,
			}},
		}}

		mr := &SecureEventsMap{
			eventMap: evMap,
			mu:       sync.RWMutex{},
		}
		return mr
	}

	cases := []struct {
		name     string
		mockRepo *SecureEventsMap
		eid      string
		uid      uint
		wantRes  bool
	}{
		{
			name:     "pos - event marked 1",
			mockRepo: mockRepoBuilder(),
			uid:      300,
			eid:      "someUUID_0",
			wantRes:  true},
		{
			name:     "pos - event marked 2",
			mockRepo: mockRepoBuilder(),
			uid:      300,
			eid:      "someUUID_1",
			wantRes:  true},
		{
			name:     "neg - event not found",
			mockRepo: mockRepoBuilder(),
			uid:      300,
			eid:      "non-existing_eid",
			wantRes:  false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.mockRepo.MarkEventDone(tt.uid, tt.eid)
			require.Equal(t, tt.wantRes, res, fmt.Sprintf("Expected res is %v, but got %v", tt.wantRes, res))
		})
	}
}

func TestGetNextEventTime(t *testing.T) {
	globalStartTime := time.Now()
	mockRepoBuilder := func(n int) *SecureEventsMap {
		startTime := globalStartTime
		evHeap := model.EventHeap{}
		for i := range n {
			evSchTime := model.CustomTime{}
			evSchTime.Time = startTime
			evHeap = append(evHeap, &model.HeapEntity{
				Index: i,
				Event: &model.Event{
					Scheduled: &evSchTime,
				},
			})
			startTime = startTime.AddDate(0, 0, 1)
		}
		heap.Init(&evHeap)

		mr := &SecureEventsMap{
			eh: evHeap,
			mu: sync.RWMutex{},
		}
		return mr
	}
	cases := []struct {
		name     string
		mockRepo *SecureEventsMap
		wantRes  bool
	}{
		{
			name:     "Neg - no events in heap",
			mockRepo: mockRepoBuilder(0),
			wantRes:  false,
		},
		{
			name:     "Pos - 1 event in heap",
			mockRepo: mockRepoBuilder(1),
			wantRes:  true,
		},
		{
			name:     "Pos - 2 events in heap",
			mockRepo: mockRepoBuilder(2),
			wantRes:  true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.mockRepo.GetNextEventTime()
			if tt.wantRes {
				require.NoError(t, err, fmt.Sprintf("Expected no error, but got this %v", err))
				require.Equal(t, globalStartTime, res, fmt.Sprintf("Expected time %v, got %v", globalStartTime, res))
				return
			}
			require.Error(t, err, "Expected error, but got nil")

		})
	}
}

func TestPopNearestEvent(t *testing.T) {
	globalStartTime := time.Now()
	mockRepoBuilder := func(n int) *SecureEventsMap {
		startTime := globalStartTime
		var evHeap model.EventHeap
		evHeap = []*model.HeapEntity{}
		for i := range n {
			evSchTime := model.CustomTime{}
			evSchTime.Time = startTime
			evHeap = append(evHeap, &model.HeapEntity{
				Index: i,
				Event: &model.Event{
					Scheduled: &evSchTime,
				},
			})
			startTime = startTime.AddDate(0, 0, 1)
		}
		heap.Init(&evHeap)

		mr := &SecureEventsMap{
			eh: evHeap,
			mu: sync.RWMutex{},
		}
		return mr
	}
	cases := []struct {
		name     string
		mockRepo *SecureEventsMap
		wantRes  bool
	}{
		{
			name:     "Neg - no events in heap",
			mockRepo: mockRepoBuilder(0),
			wantRes:  false,
		},
		{
			name:     "Pos - 1 event in heap",
			mockRepo: mockRepoBuilder(1),
			wantRes:  true,
		},
		{
			name:     "Pos - 2 events in heap",
			mockRepo: mockRepoBuilder(2),
			wantRes:  true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.mockRepo.PopNearestEvent()
			if tt.wantRes {
				require.NoError(t, err, fmt.Sprintf("Expected no error, but got this %v", err))
				require.NotNil(t, res, "Expected event ptr, but got nil")
				require.Equal(t, globalStartTime, res.Scheduled.Time, fmt.Sprintf("Expected event with sched-time %v, got %v", globalStartTime, res.Scheduled))
				return
			}
			require.Error(t, err, "Expected error, but got nil")
		})
	}
}
