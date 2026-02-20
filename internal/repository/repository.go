// Package repository provides CRUD-methods to eventsmap
package repository

import (
	"time"

	"calendar/model"
)

type EventRepository interface {
	CreateEvent(uid model.UserID, event model.Event)
	UpdateEvent(uid model.UserID, event model.Event) *model.Event
	DeleteEvent(uid model.UserID, eid string) bool
	GetPeriodEvents(uid model.UserID, start, end time.Time) []model.Event
	SafeLockMap()
	SafeUnlockMap()
}

type eventRepository struct {
	Calendar model.CalendArray
}

func NewEventRepository(emap map[model.UserID][]*model.Event) EventRepository {
	return &eventRepository{
		Calendar: model.CalendArray{EventMap: emap},
	}
}

func (er *eventRepository) CreateEvent(uid model.UserID, event model.Event) {
	er.Calendar.Lock()
	defer er.Calendar.Unlock()

	e := event
	er.Calendar.EventMap[uid] = append(er.Calendar.EventMap[uid], &e)
}

func (er *eventRepository) UpdateEvent(uid model.UserID, event model.Event) *model.Event {
	er.Calendar.Lock()
	defer er.Calendar.Unlock()

	userEvents := er.Calendar.EventMap[uid]

	for _, v := range userEvents {
		if v.EID == event.EID {
			if event.Task != "" {
				v.Task = event.Task
			}
			if event.Scheduled != nil {
				v.Scheduled = event.Scheduled
			}
			v.Updated = time.Now().UTC()

			updatedEvent := *v // чтобы не обращаться к элементу, привязанному к карте вне мьютекса
			return &updatedEvent
		}
	}
	return nil
}

func (er *eventRepository) DeleteEvent(uid model.UserID, eid string) bool {
	er.Calendar.Lock()
	defer er.Calendar.Unlock()

	userEvents := er.Calendar.EventMap[uid]

	for i, v := range userEvents {
		if v.EID.String() == eid {
			er.Calendar.EventMap[uid] = append(userEvents[:i], userEvents[(i+1):]...)
			return true
		}
	}
	return false
}

func (er *eventRepository) GetPeriodEvents(uid model.UserID, start, end time.Time) []model.Event {
	er.Calendar.RLock()
	defer er.Calendar.RUnlock()

	userEvents, ok := er.Calendar.EventMap[uid]
	if !ok {
		return nil
	}

	result := []model.Event{}

	for _, v := range userEvents {
		if !v.Scheduled.Before(start) && !v.Scheduled.After(end) {
			result = append(result, *v)
		}
	}

	return result
}

func (er *eventRepository) SafeLockMap() {
	er.Calendar.RLock()
}

func (er *eventRepository) SafeUnlockMap() {
	er.Calendar.RUnlock()
}
