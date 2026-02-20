package service_test

import (
	"testing"
	"time"

	"calendar/internal/service"
	"calendar/model"

	"github.com/google/uuid"
)

// мокаем слой репозитория
type mockRepo struct {
	createCalled bool

	updateFn func(uid model.UserID, e model.Event) *model.Event
	deleteFn func(uid model.UserID, eid string) bool
	getFn    func(uid model.UserID, start, end time.Time) []model.Event
}

func (m *mockRepo) CreateEvent(uid model.UserID, e model.Event) {
	m.createCalled = true
}

func (m *mockRepo) UpdateEvent(uid model.UserID, e model.Event) *model.Event {
	if m.updateFn != nil {
		return m.updateFn(uid, e)
	}
	return nil
}

func (m *mockRepo) DeleteEvent(uid model.UserID, eid string) bool {
	if m.deleteFn != nil {
		return m.deleteFn(uid, eid)
	}
	return false
}

func (m *mockRepo) GetPeriodEvents(uid model.UserID, start, end time.Time) []model.Event {
	if m.getFn != nil {
		return m.getFn(uid, start, end)
	}
	return nil
}

func (m *mockRepo) SafeLockMap() {
}

func (m *mockRepo) SafeUnlockMap() {
}

// Сами тесты
func TestCreateEvent_EmptyInput(t *testing.T) {
	repo := &mockRepo{}
	srv := service.NewEventService(repo)

	e, err := srv.CreateEvent(0, model.Event{})
	if err != service.ErrNothingToCreate {
		t.Fatalf("expected ErrNothingToCreate, got: %v", err)
	}
	if e != nil {
		t.Fatalf("expected nil event, got: %+v", e)
	}
}

func TestCreateEvent_NoUserID(t *testing.T) {
	repo := &mockRepo{}
	srv := service.NewEventService(repo)

	evt := model.Event{
		Scheduled: &model.CustomTime{Time: time.Now()},
		Task:      "test",
	}

	_, err := srv.CreateEvent(0, evt)
	if err != service.ErrUserIDNotSpecified {
		t.Fatalf("expected ErrUserIDNotSpecified, got: %v", err)
	}
}

func TestCreateEvent_Success(t *testing.T) {
	repo := &mockRepo{}
	srv := service.NewEventService(repo)

	evt := model.Event{
		Scheduled: &model.CustomTime{Time: time.Now()},
		Task:      "meeting",
	}

	result, err := srv.CreateEvent(1, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.EID == uuid.Nil {
		t.Fatalf("expected generated UUID, got nil")
	}
	if result.Created.IsZero() {
		t.Fatalf("expected Created timestamp to be set")
	}
	if !repo.createCalled {
		t.Fatalf("expected CreateEvent to call repo.CreateEvent")
	}
}

func TestUpdateEvent_NotFound(t *testing.T) {
	repo := &mockRepo{
		updateFn: func(uid model.UserID, e model.Event) *model.Event {
			return nil
		},
	}
	srv := service.NewEventService(repo)

	evt := model.Event{
		EID:  uuid.New(),
		Task: "abc",
	}

	_, err := srv.UpdateEvent(1, evt)
	if err != service.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got: %v", err)
	}
}

func TestUpdateEvent_Success(t *testing.T) {
	updated := &model.Event{Task: "updated"}

	repo := &mockRepo{
		updateFn: func(uid model.UserID, e model.Event) *model.Event {
			return updated
		},
	}
	srv := service.NewEventService(repo)

	evt := model.Event{
		EID:  uuid.New(),
		Task: "old",
	}

	result, err := srv.UpdateEvent(1, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Task != "updated" {
		t.Fatalf("expected updated event, got: %+v", result)
	}
}

func TestDeleteEvent_NotFound(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(uid model.UserID, eid string) bool {
			return false
		},
	}
	srv := service.NewEventService(repo)

	_, err := srv.DeleteEvent(1, uuid.New())
	if err != service.ErrEventNotFound {
		t.Fatalf("expected ErrEventNotFound, got: %v", err)
	}
}

func TestDeleteEvent_Success(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(uid model.UserID, eid string) bool {
			return true
		},
	}
	srv := service.NewEventService(repo)

	ok, err := srv.DeleteEvent(1, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected delete success")
	}
}

func TestGetDayEvents_UserNotFound(t *testing.T) {
	repo := &mockRepo{
		getFn: func(uid model.UserID, start, end time.Time) []model.Event {
			return nil // сигнал: юзер отсутствует
		},
	}
	srv := service.NewEventService(repo)

	_, err := srv.GetDayEvents(1, `"2023-10-11"`)
	if err != service.ErrUserIDNotFound {
		t.Fatalf("expected ErrUserIDNotFound, got: %v", err)
	}
}

func TestGetDayEvents_Success(t *testing.T) {
	events := []model.Event{{Task: "A"}}

	repo := &mockRepo{
		getFn: func(uid model.UserID, start, end time.Time) []model.Event {
			return events
		},
	}
	srv := service.NewEventService(repo)

	result, err := srv.GetDayEvents(1, `"2023-10-11"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result))
	}
}
