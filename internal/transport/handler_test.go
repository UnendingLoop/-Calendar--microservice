package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/engine"
	"github.com/UnendingLoop/-Calendar--microservice/internal/logger"
	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/ginext"
)

type mockSrv struct {
	returnCreateEvent    func(newEvent model.Event) (*model.Event, error)
	returnUpdateEvent    func(updatedEvent model.Event) (*model.Event, error)
	returnDeleteEvent    func(eid string, uid uint) error
	returnGetDayEvents   func(uid uint, start *time.Time) ([]model.Event, error)
	returnGetWeekEvents  func(uid uint, start *time.Time) ([]model.Event, error)
	returnGetMonthEvents func(uid uint, start *time.Time) ([]model.Event, error)
}

func (ms *mockSrv) CreateEvent(newEvent model.Event) (*model.Event, error) {
	return ms.returnCreateEvent(newEvent)
}
func (ms *mockSrv) UpdateEvent(updatedEvent model.Event) (*model.Event, error) {
	return ms.returnUpdateEvent(updatedEvent)
}
func (ms *mockSrv) DeleteEvent(eid string, uid uint) error {
	return ms.returnDeleteEvent(eid, uid)
}
func (ms *mockSrv) GetDayEvents(uid uint, start *time.Time) ([]model.Event, error) {
	return ms.returnGetDayEvents(uid, start)
}
func (ms *mockSrv) GetWeekEvents(uid uint, start *time.Time) ([]model.Event, error) {
	return ms.returnGetWeekEvents(uid, start)
}
func (ms *mockSrv) GetMonthEvents(uid uint, start *time.Time) ([]model.Event, error) {
	return ms.returnGetMonthEvents(uid, start)
}

func newMockServerWithCancel(mSrv *mockSrv) (*http.Server, func()) {
	hndlr := NewEventHandler(mSrv)
	mockCtx, cancel := context.WithCancel(context.Background())
	mockCh := make(chan *logger.EventEntry, 1)
	mockLogger := logger.NewAsyncLogger(mockCtx, mockCh)

	go func() {
		for range mockCh {
		}
	}()

	cancelFunc := func() {
		cancel()
		close(mockCh)
	}
	mockedSrv := engine.NewServer(mockCtx, "debug", "", hndlr, mockLogger)
	return mockedSrv, cancelFunc
}

func TestCreateEvent(t *testing.T) {
	cases := []struct {
		name     string
		event    *model.Event
		target   string
		method   string
		mockSvc  *mockSrv
		wantCode int
	}{
		{
			name:     "Incorrect JSON",
			event:    &model.Event{},
			target:   "/events",
			method:   http.MethodPost,
			mockSvc:  &mockSrv{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "400 error",
			event: &model.Event{
				UID:         1,
				Scheduled:   &model.CustomTime{},
				Description: "description",
			},
			target: "/events",
			method: http.MethodPost,
			mockSvc: &mockSrv{
				returnCreateEvent: func(newEvent model.Event) (*model.Event, error) {
					return nil, model.ErrIncorrectDate
				},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "500 error",
			event: &model.Event{
				UID:         1,
				Scheduled:   &model.CustomTime{},
				Description: "description",
			},
			target: "/events",
			method: http.MethodPost,
			mockSvc: &mockSrv{
				returnCreateEvent: func(newEvent model.Event) (*model.Event, error) {
					return nil, errors.New("some error")
				},
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "201 - success",
			event: &model.Event{
				UID:         1,
				Scheduled:   &model.CustomTime{},
				Description: "description",
			},
			target: "/events",
			method: http.MethodPost,
			mockSvc: &mockSrv{
				returnCreateEvent: func(newEvent model.Event) (*model.Event, error) {
					return &model.Event{}, nil
				},
			},
			wantCode: http.StatusCreated,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.event)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, tt.target, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			r, cancel := newMockServerWithCancel(tt.mockSvc)
			defer cancel()
			r.Handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestUpdateEvent(t *testing.T) {
	cases := []struct {
		name     string
		event    *model.Event
		target   string
		method   string
		mockSvc  *mockSrv
		wantCode int
	}{
		{
			name:     "Incorrect JSON",
			event:    &model.Event{},
			target:   "/events",
			method:   http.MethodPatch,
			mockSvc:  &mockSrv{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "400 error",
			event: &model.Event{
				UID:         1,
				Scheduled:   &model.CustomTime{},
				Description: "description",
			},
			target: "/events",
			method: http.MethodPatch,
			mockSvc: &mockSrv{
				returnUpdateEvent: func(updatedEvent model.Event) (*model.Event, error) {
					return nil, model.ErrIncorrectDate
				},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "500 error",
			event: &model.Event{
				UID:         1,
				Scheduled:   &model.CustomTime{},
				Description: "description",
			},
			target: "/events",
			method: http.MethodPatch,
			mockSvc: &mockSrv{
				returnUpdateEvent: func(newEvent model.Event) (*model.Event, error) {
					return nil, errors.New("some error")
				},
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "200 - success",
			event: &model.Event{
				UID:         1,
				Scheduled:   &model.CustomTime{},
				Description: "description",
			},
			target: "/events",
			method: http.MethodPatch,
			mockSvc: &mockSrv{
				returnUpdateEvent: func(newEvent model.Event) (*model.Event, error) {
					return &model.Event{}, nil
				},
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.event)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, tt.target, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			r, cancel := newMockServerWithCancel(tt.mockSvc)
			defer cancel()
			r.Handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestDeleteEvent(t *testing.T) {
	cases := []struct {
		name      string
		candidate *model.DeleteDTO
		target    string
		method    string
		mockSvc   *mockSrv
		wantCode  int
	}{
		{
			name:      "Incorrect JSON",
			candidate: &model.DeleteDTO{},
			target:    "/events",
			method:    http.MethodDelete,
			mockSvc:   &mockSrv{},
			wantCode:  http.StatusBadRequest,
		},
		{
			name: "400 error",
			candidate: &model.DeleteDTO{
				EID: "",
				UID: 0,
			},
			target: "/events",
			method: http.MethodDelete,
			mockSvc: &mockSrv{
				returnDeleteEvent: func(eid string, uid uint) error {
					return model.ErrUserIDNotSpecified
				},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "500 error",
			candidate: &model.DeleteDTO{
				EID: "someUUID",
				UID: 300,
			},
			target: "/events",
			method: http.MethodDelete,
			mockSvc: &mockSrv{
				returnDeleteEvent: func(eid string, uid uint) error {
					return model.ErrCommon500
				},
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "204 - success",
			candidate: &model.DeleteDTO{
				EID: "someUUID",
				UID: 300,
			},
			target: "/events",
			method: http.MethodDelete,
			mockSvc: &mockSrv{
				returnDeleteEvent: func(eid string, uid uint) error {
					return nil
				},
			},
			wantCode: http.StatusNoContent,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.candidate)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, tt.target, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()

			r, cancel := newMockServerWithCancel(tt.mockSvc)
			defer cancel()
			r.Handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestGetEvents(t *testing.T) {
	cases := []struct {
		name     string
		target   [][]string
		method   string
		mockSvc  *mockSrv
		wantCode int
	}{
		{
			name: "Incorrect Method",
			target: [][]string{
				{"For Day", "/events/for_day?user_id=1&date=2026-12-31"},
				{"For Week", "/events/for_week?user_id=1&date=2026-12-31"},
				{"For Month", "/events/for_month?user_id=1&date=2026-12-31"},
			},
			method:   http.MethodPost,
			mockSvc:  &mockSrv{},
			wantCode: http.StatusNotFound,
		},
		{
			name: "Incorrect Params - date",
			target: [][]string{
				{"For Day", "/events/for_day?user_id=1&date=300300"},
				{"For Week", "/events/for_week?user_id=1&date=300300"},
				{"For Month", "/events/for_month?user_id=1&date=300300"},
			},
			method:   http.MethodGet,
			mockSvc:  &mockSrv{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "Incorrect Params - userID",
			target: [][]string{
				{"For Day", "/events/for_day?user=1&date=2026-12-31"},
				{"For Week", "/events/for_week?user=1&date=2026-12-31"},
				{"For Month", "/events/for_month?user=1&date=2026-12-31"},
			},
			method:   http.MethodGet,
			mockSvc:  &mockSrv{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "500 error",
			target: [][]string{
				{"For Day", "/events/for_day?user_id=1&date=2026-12-31"},
				{"For Week", "/events/for_week?user_id=1&date=2026-12-31"},
				{"For Month", "/events/for_month?user_id=1&date=2026-12-31"},
			},
			method: http.MethodGet,
			mockSvc: &mockSrv{
				returnGetDayEvents: func(uid uint, start *time.Time) ([]model.Event, error) {
					return nil, model.ErrCommon500
				},
				returnGetWeekEvents: func(uid uint, start *time.Time) ([]model.Event, error) {
					return nil, model.ErrCommon500
				},
				returnGetMonthEvents: func(uid uint, start *time.Time) ([]model.Event, error) {
					return nil, model.ErrCommon500
				},
			},
			wantCode: http.StatusInternalServerError,
		},
		{
			name: "200 - success",
			target: [][]string{
				{"For Day", "/events/for_day?user_id=1&date=2026-12-31"},
				{"For Week", "/events/for_week?user_id=1&date=2026-12-31"},
				{"For Month", "/events/for_month?user_id=1&date=2026-12-31"},
			},
			method: http.MethodGet,
			mockSvc: &mockSrv{
				returnGetDayEvents: func(uid uint, start *time.Time) ([]model.Event, error) {
					return []model.Event{}, nil
				},
				returnGetWeekEvents: func(uid uint, start *time.Time) ([]model.Event, error) {
					return []model.Event{}, nil
				},
				returnGetMonthEvents: func(uid uint, start *time.Time) ([]model.Event, error) {
					return []model.Event{}, nil
				},
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range cases {
		for _, target := range tt.target {
			t.Run(tt.name+" - endpoint "+target[0], func(t *testing.T) {

				req := httptest.NewRequest(tt.method, target[1], bytes.NewReader(nil))

				rec := httptest.NewRecorder()

				r, cancel := newMockServerWithCancel(tt.mockSvc)
				defer cancel()
				r.Handler.ServeHTTP(rec, req)

				require.Equal(t, tt.wantCode, rec.Code, fmt.Sprintf("Endpoint %q returned %d instead of %d", target, tt.wantCode, rec.Code))
			})
		}

	}
}

func TestHealthCheck(t *testing.T) {
	srv, cancel := newMockServerWithCancel(nil)
	defer cancel()
	require.NotEqual(t, nil, srv, "Received nil-server")

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	srv.Handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGetUserIDAndDate(t *testing.T) {
	testCases := []struct {
		name           string
		queryParams    string
		expectedUserID int
		expectedDate   *time.Time
		shouldError    bool
		errorContains  string
	}{
		{
			name:           "valid user_id and date",
			queryParams:    "user_id=123&date=2026-03-18",
			expectedUserID: 123,
			expectedDate:   ptrTime(time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)),
			shouldError:    false,
		},
		{
			name:          "missing user_id parameter",
			queryParams:   "date=2026-03-18",
			shouldError:   true,
			errorContains: "empty user ID",
		},
		{
			name:          "missing date parameter",
			queryParams:   "user_id=123",
			shouldError:   true,
			errorContains: "empty date",
		},
		{
			name:          "invalid user_id - not a number",
			queryParams:   "user_id=abc&date=2026-03-18",
			shouldError:   true,
			errorContains: "incorrect user ID",
		},
		{
			name:          "invalid user_id - zero",
			queryParams:   "user_id=0&date=2026-03-18",
			shouldError:   true,
			errorContains: "incorrect user ID",
		},
		{
			name:          "invalid user_id - negative",
			queryParams:   "user_id=-5&date=2026-03-18",
			shouldError:   true,
			errorContains: "incorrect user ID",
		},
		{
			name:          "invalid date format",
			queryParams:   "user_id=123&date=18-03-2026",
			shouldError:   true,
			errorContains: "date parse error",
		},
		{
			name:           "large valid user_id",
			queryParams:    "user_id=999999&date=2026-12-31",
			expectedUserID: 999999,
			expectedDate:   ptrTime(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError:    false,
		},
		{
			name:           "user_id with leading zeros",
			queryParams:    "user_id=00123&date=2026-01-01",
			expectedUserID: 123,
			expectedDate:   ptrTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			shouldError:    false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.queryParams, nil)
			c := &ginext.Context{Request: req}

			id, date, err := getUserIDandDate(c)

			if tt.shouldError {
				require.Error(t, err, "expected error but got none")
				require.Contains(t, err.Error(), tt.errorContains,
					"error should contain %q", tt.errorContains)
			} else {
				require.NoError(t, err, "expected no error but got: %v", err)
				require.Equal(t, tt.expectedUserID, id,
					"expected user_id %d, got %d", tt.expectedUserID, id)
				if tt.expectedDate != nil && date != nil {
					require.Equal(t, tt.expectedDate.Unix(), date.Unix(),
						"dates should match")
				}
			}
		})
	}
}

// Helper function to create a pointer to time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
