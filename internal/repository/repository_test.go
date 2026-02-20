package repository_test

import (
	"testing"
	"time"

	"calendar/internal/repository"
	"calendar/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateEvent(t *testing.T) {
	emap := make(map[model.UserID][]*model.Event)
	repo := repository.NewEventRepository(emap)

	uid := model.UserID(1)
	event := model.Event{
		EID:       uuid.New(),
		Task:      "Test event",
		Scheduled: &model.CustomTime{Time: (time.Now())},
	}

	repo.CreateEvent(uid, event)

	assert.Len(t, emap[uid], 1)
	assert.Equal(t, "Test event", emap[uid][0].Task)
}

func TestUpdateEvent(t *testing.T) {
	uid := model.UserID(1)
	eid := uuid.New()

	oldTime := &model.CustomTime{Time: (time.Now())}
	newTime := &model.CustomTime{Time: (time.Now().Add(24 * time.Hour))}

	existing := &model.Event{
		EID:       eid,
		Task:      "Old task",
		Scheduled: oldTime,
	}

	emap := map[model.UserID][]*model.Event{
		uid: {existing},
	}

	repo := repository.NewEventRepository(emap)

	update := model.Event{
		EID:       eid,
		Task:      "New task",
		Scheduled: newTime,
	}

	result := repo.UpdateEvent(uid, update)

	assert.NotNil(t, result)
	assert.Equal(t, "New task", result.Task)
	assert.Equal(t, newTime, result.Scheduled)
}

func TestUpdateEvent_NotFound(t *testing.T) {
	uid := model.UserID(1)

	emap := map[model.UserID][]*model.Event{
		uid: {},
	}

	repo := repository.NewEventRepository(emap)

	update := model.Event{
		EID:  uuid.New(),
		Task: "Updated",
	}

	result := repo.UpdateEvent(uid, update)

	assert.Nil(t, result)
}

func TestDeleteEvent(t *testing.T) {
	uid := model.UserID(1)
	eid := uuid.New()

	event := &model.Event{
		EID:       eid,
		Scheduled: &model.CustomTime{Time: (time.Now())},
		Task:      "to delete",
	}

	emap := map[model.UserID][]*model.Event{
		uid: {event},
	}

	repo := repository.NewEventRepository(emap)

	ok := repo.DeleteEvent(uid, eid.String())

	assert.True(t, ok)
	assert.Len(t, emap[uid], 0)
}

func TestDeleteEvent_NotFound(t *testing.T) {
	uid := model.UserID(1)
	eid := uuid.New()

	emap := map[model.UserID][]*model.Event{
		uid: {},
	}

	repo := repository.NewEventRepository(emap)

	ok := repo.DeleteEvent(uid, eid.String())

	assert.False(t, ok)
}

func TestGetPeriodEvents(t *testing.T) {
	uid := model.UserID(1)

	d1 := &model.CustomTime{Time: time.Date(2023, 12, 30, 0, 0, 0, 0, time.UTC)}
	d2 := &model.CustomTime{Time: time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)}
	d3 := &model.CustomTime{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}

	emap := map[model.UserID][]*model.Event{
		uid: {
			{EID: uuid.New(), Scheduled: d1, Task: "e1"},
			{EID: uuid.New(), Scheduled: d2, Task: "e2"},
			{EID: uuid.New(), Scheduled: d3, Task: "e3"},
		},
	}

	repo := repository.NewEventRepository(emap)

	start := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	res := repo.GetPeriodEvents(uid, start, end)

	assert.Len(t, res, 2)
	assert.Equal(t, "e2", res[0].Task)
	assert.Equal(t, "e3", res[1].Task)
}

func TestGetPeriodEvents_UserNotFound(t *testing.T) {
	emap := map[model.UserID][]*model.Event{}
	repo := repository.NewEventRepository(emap)

	res := repo.GetPeriodEvents(1, time.Now(), time.Now().Add(24*time.Hour))

	assert.Nil(t, res)
}

func TestSafeLockUnlock(t *testing.T) {
	emap := make(map[model.UserID][]*model.Event)
	repo := repository.NewEventRepository(emap)

	repo.SafeLockMap()
	repo.SafeUnlockMap()

	assert.True(t, true)
}
