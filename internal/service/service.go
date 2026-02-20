// Package service implements business logics and calls repository methods for updating eventsmap
package service

import (
	"errors"
	"fmt"
	"time"

	"calendar/internal/repository"
	"calendar/model"

	"github.com/google/uuid"
)

type EventService interface {
	CreateEvent(uid model.UserID, newEvent model.Event) (*model.Event, error)
	UpdateEvent(uid model.UserID, newEvent model.Event) (*model.Event, error)
	DeleteEvent(uid model.UserID, eid uuid.UUID) (bool, error)
	GetDayEvents(uid model.UserID, start string) ([]model.Event, error)
	GetWeekEvents(uid model.UserID, start string) ([]model.Event, error)
	GetMonthEvents(uid model.UserID, start string) ([]model.Event, error)
}

type eventService struct {
	Repo repository.EventRepository
}

var (
	ErrUserIDNotSpecified  = errors.New("отсутствует ID пользователя")
	ErrUserIDNotFound      = errors.New("указанный ID пользователя не найден")
	ErrEventIDNotSpecified = errors.New("отсутствует ID события")
	ErrNothingToDelete     = errors.New("отсутствуют ID пользователя и события для удаления")
	ErrEventNotFound       = errors.New("указанное по ID событие для указанного пользователя не найдено")
	ErrDateNotSpecified    = errors.New("отсутствует дата для события")
	ErrEventNotSpecified   = errors.New("отсутствует описание события")
	ErrNothingToUpdate     = errors.New("отсутствуют новые дата и описание для обновления события")
	ErrNothingToCreate     = errors.New("отсутствуют все данные для создания события")
)

func NewEventService(repo repository.EventRepository) EventService {
	return &eventService{Repo: repo}
}

func (es *eventService) CreateEvent(uid model.UserID, newEvent model.Event) (*model.Event, error) {
	switch {
	case uid == 0 && newEvent.Scheduled == nil && newEvent.Task == "":
		return nil, ErrNothingToCreate
	case uid == 0:
		return nil, ErrUserIDNotSpecified
	case newEvent.Scheduled == nil:
		return nil, ErrDateNotSpecified
	case newEvent.Task == "":
		return nil, ErrEventNotSpecified
	}

	newEvent.EID = uuid.New()
	newEvent.Created = time.Now().UTC()
	es.Repo.CreateEvent(uid, newEvent)

	return &newEvent, nil
}

func (es *eventService) UpdateEvent(uid model.UserID, newEvent model.Event) (*model.Event, error) {
	switch {
	case uid == 0:
		return nil, ErrUserIDNotSpecified
	case newEvent.EID.String() == "":
		return nil, ErrEventIDNotSpecified
	case newEvent.Scheduled == nil && newEvent.Task == "":
		return nil, ErrNothingToUpdate
	default:
		newEvent.Updated = time.Now().UTC()
		updatedEvent := es.Repo.UpdateEvent(uid, newEvent)
		if updatedEvent == nil {
			return nil, ErrEventNotFound
		}
		return updatedEvent, nil
	}
}

func (es *eventService) DeleteEvent(uid model.UserID, eid uuid.UUID) (bool, error) {
	eventID := eid.String()

	switch {
	case uid == 0 && eventID == "":
		return false, ErrNothingToDelete
	case eventID == "":
		return false, ErrEventIDNotSpecified
	case uid == 0:
		return false, ErrUserIDNotSpecified
	default:
		if es.Repo.DeleteEvent(uid, eventID) {
			return true, nil
		}
		return false, ErrEventNotFound
	}
}

func (es *eventService) GetDayEvents(uid model.UserID, start string) ([]model.Event, error) {
	return es.getEvents(uid, start, 1, 0)
}

func (es *eventService) GetWeekEvents(uid model.UserID, start string) ([]model.Event, error) {
	return es.getEvents(uid, start, 7, 0)
}

func (es *eventService) GetMonthEvents(uid model.UserID, start string) ([]model.Event, error) {
	return es.getEvents(uid, start, 0, 1)
}

func (es *eventService) getEvents(uid model.UserID, start string, addDays, addMonths int) ([]model.Event, error) {
	switch {
	case uid == 0:
		return nil, ErrUserIDNotSpecified
	case start == "" || start == "nil" || start == "null":
		return nil, ErrDateNotSpecified
	}

	startDate := model.CustomTime{}
	if err := startDate.UnmarshalJSON([]byte(start)); err != nil {
		return nil, fmt.Errorf("ошибка при парсинге указанной даты: %v", err)
	}

	endDate := startDate.AddDate(0, addMonths, addDays).UTC()

	result := es.Repo.GetPeriodEvents(uid, startDate.Time, endDate)

	if result == nil {
		return nil, ErrUserIDNotFound
	}

	return result, nil
}
