// Package service implements business logics and calls repository methods for updating eventsmap
package service

import (
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"

	"github.com/google/uuid"
)

type eventRepository interface {
	CreateEvent(event model.Event)
	UpdateEvent(uid uint, event model.Event) *model.Event
	DeleteEvent(uid uint, eid string) bool
	GetPeriodEvents(uid uint, start, end time.Time) []model.Event
}

type EventService struct {
	er eventRepository
}

func NewEventService(repo eventRepository) *EventService {
	return &EventService{er: repo}
}

func (es *EventService) CreateEvent(newEvent model.Event) (*model.Event, error) {
	switch {
	case newEvent.UID == 0 && newEvent.Scheduled == nil && newEvent.Description == "":
		return nil, model.ErrNothingToCreate
	case newEvent.UID == 0:
		return nil, model.ErrUserIDNotSpecified
	case newEvent.Scheduled == nil:
		return nil, model.ErrDateNotSpecified
	case newEvent.Description == "":
		return nil, model.ErrEventNotSpecified
	}

	newEvent.EID = uuid.New().String()
	newEvent.Created = time.Now().UTC()
	es.er.CreateEvent(newEvent)

	return &newEvent, nil
}

func (es *EventService) UpdateEvent(updatedEvent model.Event) (*model.Event, error) {
	switch {
	case updatedEvent.UID == 0:
		return nil, model.ErrUserIDNotSpecified
	case updatedEvent.EID == "":
		return nil, model.ErrEventIDNotSpecified
	case updatedEvent.Scheduled == nil && updatedEvent.Description == "":
		return nil, model.ErrNothingToUpdate
	case updatedEvent.Scheduled != nil && updatedEvent.Scheduled.Before(time.Now().UTC()):
		return nil, model.ErrEventTimePast
	default:
		updatedEvent.Updated = time.Now().UTC()
		updatedEvent := es.er.UpdateEvent(updatedEvent.UID, updatedEvent)
		if updatedEvent == nil {
			return nil, model.ErrEventNotFound
		}
		return updatedEvent, nil
	}
}

func (es *EventService) DeleteEvent(event model.Event) error {
	switch {
	case event.UID == 0 && event.EID == "":
		return model.ErrNothingToDelete
	case event.EID == "":
		return model.ErrEventIDNotSpecified
	case event.UID == 0:
		return model.ErrUserIDNotSpecified
	default:
		if es.er.DeleteEvent(event.UID, event.EID) {
			return nil
		}
		return model.ErrEventNotFound
	}
}

func (es *EventService) GetDayEvents(uid uint, start *model.CustomTime) ([]model.Event, error) {
	return es.getEvents(uid, start, 1, 0)
}

func (es *EventService) GetWeekEvents(uid uint, start *model.CustomTime) ([]model.Event, error) {
	return es.getEvents(uid, start, 7, 0)
}

func (es *EventService) GetMonthEvents(uid uint, start *model.CustomTime) ([]model.Event, error) {
	return es.getEvents(uid, start, 0, 1)
}

func (es *EventService) getEvents(uid uint, start *model.CustomTime, addDays, addMonths int) ([]model.Event, error) {
	switch {
	case uid == 0:
		return nil, model.ErrUserIDNotSpecified
	case start == nil:
		return nil, model.ErrDateNotSpecified
	}

	endDate := start.AddDate(0, addMonths, addDays).UTC()

	result := es.er.GetPeriodEvents(uid, start.Time, endDate)

	if result == nil {
		return nil, model.ErrUserIDNotFound
	}

	return result, nil
}
