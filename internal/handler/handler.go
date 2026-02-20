// Package handler implements HTTP-methods and redirects requests to service-layer
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"calendar/internal/service"
	"calendar/model"
)

type EventHandler interface {
	CreateEvent(w http.ResponseWriter, r *http.Request)
	UpdateEvent(w http.ResponseWriter, r *http.Request)
	DeleteEvent(w http.ResponseWriter, r *http.Request)
	GetDayEvents(w http.ResponseWriter, r *http.Request)
	GetWeekEvents(w http.ResponseWriter, r *http.Request)
	GetMonthEvents(w http.ResponseWriter, r *http.Request)
}

type eventHandler struct {
	Srv service.EventService
}

func NewEventHandler(srv service.EventService) EventHandler {
	return &eventHandler{Srv: srv}
}

func (eh *eventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "Отсутствует тело запроса")
		return
	}

	defer r.Body.Close()

	userEvent := model.Combined{}
	if err := json.NewDecoder(r.Body).Decode(&userEvent); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Некорректный формат данных: %v", err))
		return
	}

	result, err := eh.Srv.CreateEvent(userEvent.UserID, userEvent.Event)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // вообще надо бы отдавать 201 Created, но в задаче написано использовать 200 для успешных запросов.
	_ = json.NewEncoder(w).Encode(result)
}

func (eh *eventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "Отсутствует тело запроса")
		return
	}

	defer r.Body.Close()

	userEvent := model.Combined{}
	if err := json.NewDecoder(r.Body).Decode(&userEvent); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Некорректный формат данных: %v", err))
		return
	}

	result, err := eh.Srv.UpdateEvent(userEvent.UserID, userEvent.Event)
	if err != nil {
		if errors.Is(err, service.ErrEventNotFound) {
			writeError(w, http.StatusServiceUnavailable, err.Error()) // должен быть статус 404, но в задаче написано использовать 503
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func (eh *eventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "Отсутствует тело запроса")
		return
	}
	defer r.Body.Close()

	var userEvent model.Combined
	if err := json.NewDecoder(r.Body).Decode(&userEvent); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Ошибка при декодировании JSON: %v", err))
		return
	}

	ok, err := eh.Srv.DeleteEvent(userEvent.UserID, userEvent.EID)
	if ok {
		w.WriteHeader(http.StatusOK) // в задании указано 200 OK
		return
	}

	switch {
	case errors.Is(err, service.ErrEventNotFound):
		writeError(w, http.StatusServiceUnavailable, err.Error()) // по заданию бизнес-ошибка = 503
	case errors.Is(err, service.ErrUserIDNotSpecified),
		errors.Is(err, service.ErrEventIDNotSpecified),
		errors.Is(err, service.ErrNothingToDelete):
		writeError(w, http.StatusBadRequest, err.Error()) // ошибки ввода = 400
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Внутренняя ошибка: %v", err))
	}
}

func (eh *eventHandler) GetDayEvents(w http.ResponseWriter, r *http.Request) {
	id, date := prepareToGetEvents(w, r)
	if id == 0 && date == "" {
		return
	}

	events, err := eh.Srv.GetDayEvents(model.UserID(id), date)
	if err != nil {
		checkError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(events)
}

func (eh *eventHandler) GetWeekEvents(w http.ResponseWriter, r *http.Request) {
	id, date := prepareToGetEvents(w, r)
	if id == 0 && date == "" {
		return
	}

	events, err := eh.Srv.GetWeekEvents(model.UserID(id), date)
	if err != nil {
		checkError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(events)
}

func (eh *eventHandler) GetMonthEvents(w http.ResponseWriter, r *http.Request) {
	id, date := prepareToGetEvents(w, r)
	if id == 0 && date == "" {
		return
	}

	events, err := eh.Srv.GetMonthEvents(model.UserID(id), date)
	if err != nil {
		checkError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(events)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func prepareToGetEvents(w http.ResponseWriter, r *http.Request) (int, string) {
	q := r.URL.Query()
	id, err := strconv.Atoi(q.Get("user_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return 0, ""
	}

	date := q.Get("date")

	if id == 0 && date == "" {
		writeError(w, http.StatusBadRequest, "missing user_id and date")
		return 0, ""
	}

	if id == 0 {
		writeError(w, http.StatusBadRequest, "missing user_id")
		return 0, ""
	}

	if date == "" {
		writeError(w, http.StatusBadRequest, "missing date")
		return 0, ""
	}

	return id, date
}

func checkError(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, service.ErrUserIDNotSpecified), errors.Is(err, service.ErrDateNotSpecified):
		writeError(w, http.StatusBadRequest, err.Error()) // ошибки ввода = 400
	case errors.Is(err, service.ErrUserIDNotFound):
		writeError(w, http.StatusServiceUnavailable, err.Error()) // по заданию бизнес-ошибка = 503, но должно быть 404
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Внутренняя ошибка: %v", err))
	}
}
