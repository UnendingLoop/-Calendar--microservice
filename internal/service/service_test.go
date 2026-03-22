package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	returnUpdateEvent func(uid uint, event model.Event) *model.Event
	returnDeleteEvent func(uid uint, eid string) bool

	returnGetPeriodEvents func(uid uint, start, end time.Time) []model.Event
	calledEnd             time.Time
}

func (mr *mockRepository) CreateEvent(event model.Event) {
}
func (mr *mockRepository) UpdateEvent(uid uint, event model.Event) *model.Event {
	return mr.returnUpdateEvent(uid, event)
}
func (mr *mockRepository) DeleteEvent(uid uint, eid string) bool {
	return mr.returnDeleteEvent(uid, eid)
}
func (mr *mockRepository) GetPeriodEvents(uid uint, start, end time.Time) []model.Event {
	mr.calledEnd = end
	return mr.returnGetPeriodEvents(uid, start, end)
}

func TestCreateEvent(t *testing.T) {
	cases := []struct {
		name      string
		candEvent model.Event
		wantRes   bool
		wantErr   error
	}{
		{
			name: "Pos - no err",
			candEvent: func() model.Event {
				var candidate model.Event
				candidate.UID = 300
				candidate.Description = "Some description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, 1)
				candidate.Scheduled = &ct
				return candidate
			}(),
			wantRes: true,
			wantErr: nil,
		},
		{
			name: "Neg - all fields empty",
			candEvent: func() model.Event {
				var candidate model.Event
				candidate.UID = 0
				candidate.Description = ""
				candidate.Scheduled = nil
				return candidate
			}(),
			wantRes: false,
			wantErr: model.ErrNothingToCreate,
		},
		{
			name: "Neg - UID empty",
			candEvent: func() model.Event {
				var candidate model.Event
				candidate.UID = 0
				candidate.Description = "Some description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, 1)
				candidate.Scheduled = &ct
				return candidate
			}(),
			wantRes: false,
			wantErr: model.ErrUserIDNotSpecified,
		},
		{
			name: "Neg - empty description",
			candEvent: func() model.Event {
				var candidate model.Event
				candidate.UID = 300
				candidate.Description = ""
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, 1)
				candidate.Scheduled = &ct
				return candidate
			}(),
			wantRes: false,
			wantErr: model.ErrEventDescrEmpty,
		},
		{
			name: "Neg - empty scheduled time",
			candEvent: func() model.Event {
				var candidate model.Event
				candidate.UID = 300
				candidate.Description = "Some description"
				candidate.Scheduled = nil
				return candidate
			}(),
			wantRes: false,
			wantErr: model.ErrIncorrectDate,
		},
		{
			name: "Neg - scheduled time in the past",
			candEvent: func() model.Event {
				var candidate model.Event
				candidate.UID = 300
				candidate.Description = "Some description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, -1)
				candidate.Scheduled = &ct
				return candidate
			}(),
			wantRes: false,
			wantErr: model.ErrIncorrectDate,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewEventService(&mockRepository{})
			res, err := svc.CreateEvent(tt.candEvent)

			switch {
			case tt.wantErr != nil:
				require.Error(t, err, "Expected error, but got nil")
				if err != nil {
					require.ErrorIs(t, err, tt.wantErr, fmt.Sprintf("Expected error %q, but got %q", tt.wantErr.Error(), err.Error()))
				}
			case tt.wantRes:
				require.NotEqual(t, tt.candEvent.EID, res.EID)
				require.NoError(t, uuid.Validate(res.EID), fmt.Sprintf("EID is invalid uuid: %v", uuid.Validate(res.EID)))
				require.Equal(t, false, res.Created.IsZero(), "'Created' cannot be zero")
				require.Equal(t, true, res.Updated.IsZero(), fmt.Sprintf("'Updated' must be zero, but got: %v", res.Updated))
				require.Equal(t, tt.candEvent.Scheduled.Time, res.Scheduled.Time, fmt.Sprintf("'Scheduled' must be %v, but got %v", tt.candEvent.Scheduled, res.Scheduled.Time))
			}
		})
	}
}

func TestUpdateEvent(t *testing.T) {
	cases := []struct {
		name        string
		updateEvent model.Event
		mockRepo    *mockRepository
		wantErr     error
	}{
		{
			name: "Pos - all possible fields updated",
			mockRepo: &mockRepository{
				returnUpdateEvent: func(uid uint, event model.Event) *model.Event {
					ee := model.Event{
						UID:         event.UID,
						EID:         event.EID,
						Description: event.Description,
						Created:     time.Now().UTC().AddDate(0, 0, -1),
						Updated:     time.Now().UTC(),
						Scheduled:   event.Scheduled,
					}
					return &ee
				},
			},
			updateEvent: func() model.Event {
				ue := model.Event{}
				ue.UID = 300
				ue.EID = "some_mock_UUID"
				ue.Description = "New description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, 2)
				ue.Scheduled = &ct
				return ue
			}(),
			wantErr: nil,
		},
		{
			name:     "Neg - 'Scheduled' in the past",
			mockRepo: nil,
			updateEvent: func() model.Event {
				ue := model.Event{}
				ue.UID = 300
				ue.EID = "some_mock_UUID"
				ue.Description = "New description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, -2)
				ue.Scheduled = &ct
				return ue
			}(),
			wantErr: model.ErrEventTimePast,
		},
		{
			name:     "Neg - UID empty",
			mockRepo: nil,
			updateEvent: func() model.Event {
				ue := model.Event{}
				ue.UID = 0
				ue.EID = "some_mock_UUID"
				ue.Description = "New description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, -2)
				ue.Scheduled = &ct
				return ue
			}(),
			wantErr: model.ErrUserIDNotSpecified,
		},
		{
			name:     "Neg - EID empty",
			mockRepo: nil,
			updateEvent: func() model.Event {
				ue := model.Event{}
				ue.UID = 300
				ue.EID = ""
				ue.Description = "New description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, -2)
				ue.Scheduled = &ct
				return ue
			}(),
			wantErr: model.ErrEventIDNotSpecified,
		},
		{
			name:     "Neg - Description and Scheduled are empty",
			mockRepo: nil,
			updateEvent: func() model.Event {
				ue := model.Event{}
				ue.UID = 300
				ue.EID = "some_mock_UUID"
				ue.Description = ""
				ue.Scheduled = nil
				return ue
			}(),
			wantErr: model.ErrNothingToUpdate,
		},
		{
			name: "Neg - event not found",
			mockRepo: &mockRepository{
				returnUpdateEvent: func(uid uint, event model.Event) *model.Event {
					return nil
				},
			},
			updateEvent: func() model.Event {
				ue := model.Event{}
				ue.UID = 300
				ue.EID = "some_mock_UUID"
				ue.Description = "New description"
				ct := model.CustomTime{}
				ct.Time = time.Now().UTC().AddDate(0, 0, 2)
				ue.Scheduled = &ct
				return ue
			}(),
			wantErr: model.ErrEventNotFound,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewEventService(tt.mockRepo)
			_, err := svc.UpdateEvent(tt.updateEvent)

			if tt.wantErr != nil {
				require.Error(t, err, "Expected error, but got nil")
				require.ErrorIs(t, err, tt.wantErr, fmt.Sprintf("Expected error %q, but got %q", tt.wantErr.Error(), err.Error()))
			}
		})
	}
}

func TestDeleteEvent(t *testing.T) {
	cases := []struct {
		name     string
		uid      uint
		eid      string
		mockRepo *mockRepository
		wantErr  error
	}{
		{
			name: "Pos - successfull delete",
			uid:  300,
			eid:  "some_mock_UUID",
			mockRepo: &mockRepository{
				returnDeleteEvent: func(uid uint, eid string) bool {
					return true
				},
			},
			wantErr: nil,
		},
		{
			name: "Pos - event not found",
			uid:  300,
			eid:  "some_mock_UUID",
			mockRepo: &mockRepository{
				returnDeleteEvent: func(uid uint, eid string) bool {
					return false
				},
			},
			wantErr: model.ErrEventNotFound,
		},
		{
			name:     "Neg - invalid uid and eid",
			uid:      0,
			eid:      "",
			mockRepo: nil,
			wantErr:  model.ErrNothingToDelete,
		},
		{
			name:     "Neg - invalid uid",
			uid:      0,
			eid:      "some_mock_UUID",
			mockRepo: nil,
			wantErr:  model.ErrUserIDNotSpecified,
		},
		{
			name:     "Neg - invalid eid",
			uid:      300,
			eid:      "",
			mockRepo: nil,
			wantErr:  model.ErrEventIDNotSpecified,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewEventService(tt.mockRepo)
			err := svc.DeleteEvent(tt.eid, tt.uid)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, fmt.Sprintf("Expected error %v, but got %v", tt.wantErr, err))
			} else {
				require.NoError(t, err, fmt.Sprintf("Expected no error, but got %v", err))
			}
		})
	}
}

func TestGetPeriodEvents(t *testing.T) {
	startTime := time.Now().UTC()
	cases := []struct {
		name     string
		mockRepo *mockRepository
		uid      uint
		start    *time.Time
		wantErr  error
		wantRes  bool
	}{
		{
			name: "Pos - user found",
			mockRepo: &mockRepository{
				returnGetPeriodEvents: func(uid uint, start time.Time, end time.Time) []model.Event {
					return []model.Event{}
				},
			},
			uid:     300,
			start:   &startTime,
			wantErr: nil,
			wantRes: true,
		},
		{
			name: "Pos - user not found",
			mockRepo: &mockRepository{
				returnGetPeriodEvents: func(uid uint, start time.Time, end time.Time) []model.Event {
					return nil
				},
			},
			uid:     300,
			start:   &startTime,
			wantErr: nil,
			wantRes: false,
		},
		{
			name:     "Neg - invalid uid",
			mockRepo: nil,
			uid:      0,
			start:    &startTime,
			wantErr:  model.ErrUserIDNotSpecified,
			wantRes:  false,
		},
		{
			name:     "Neg - invalid start",
			mockRepo: nil,
			uid:      300,
			start:    nil,
			wantErr:  model.ErrIncorrectDate,
			wantRes:  false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewEventService(tt.mockRepo)

			resultChecker := func(period string, res []model.Event, err error) {
				if tt.wantErr != nil {
					require.ErrorIs(t, err, tt.wantErr, fmt.Sprintf("Expected error %v, got %v", tt.wantErr, err))
					require.Nil(t, res, fmt.Sprintf("Expected nil-result, but got %v", res))
				} else {
					var expEnd time.Time
					switch period {
					case "day":
						expEnd = startTime.AddDate(0, 0, 1)
					case "week":
						expEnd = startTime.AddDate(0, 0, 7)
					case "month":
						expEnd = startTime.AddDate(0, 1, 0)
					}

					if tt.wantRes {
						require.NotNil(t, res, "Fetched result is nil instead of slice")
					}
					require.Equal(t, expEnd, tt.mockRepo.calledEnd, fmt.Sprintf("Expected called end-time %v, but got %v", expEnd, tt.mockRepo.calledEnd))
				}
			}

			res, err := svc.GetDayEvents(tt.uid, tt.start)
			resultChecker("day", res, err)
			res, err = svc.GetWeekEvents(tt.uid, tt.start)
			resultChecker("week", res, err)
			res, err = svc.GetMonthEvents(tt.uid, tt.start)
			resultChecker("month", res, err)
		})
	}
}
