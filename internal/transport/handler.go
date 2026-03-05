// Package transport implements HTTP-methods and redirects requests to service-layer, logs errors via async-logger
package transport

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/UnendingLoop/-Calendar--microservice/internal/logger"
	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/wb-go/wbf/ginext"
)

type eventService interface {
	CreateEvent(newEvent model.Event) (*model.Event, error)
	UpdateEvent(updatedEvent model.Event) (*model.Event, error)
	DeleteEvent(event model.Event) error
	GetDayEvents(uid uint, start *model.CustomTime) ([]model.Event, error)
	GetWeekEvents(uid uint, start *model.CustomTime) ([]model.Event, error)
	GetMonthEvents(uid uint, start *model.CustomTime) ([]model.Event, error)
}

type EventHandler struct {
	es eventService
}

func NewEventHandler(srv eventService) *EventHandler {
	return &EventHandler{es: srv}
}

func (eh *EventHandler) SimplePinger(c *ginext.Context) {
	c.JSON(200, "Pong")
}

func (eh *EventHandler) CreateEvent(c *ginext.Context) {
	eloger := (c.Value(model.LoggerCtxName)).(logger.Logger)
	newEvent := model.Event{}
	if err := c.ShouldBind(&newEvent); err != nil {
		eloger.Error("Failed to parse new event data from JSON", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := eh.es.CreateEvent(newEvent)
	if err != nil {
		eloger.Error("Failed to create new event", err)
		c.JSON(getErrorCode(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (eh *EventHandler) UpdateEvent(c *ginext.Context) {
	eloger := (c.Value(model.LoggerCtxName)).(logger.Logger)
	newEvent := model.Event{}
	if err := c.ShouldBind(&newEvent); err != nil {
		eloger.Error("Failed to parse event updated data from JSON", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := eh.es.UpdateEvent(newEvent)
	if err != nil {
		eloger.Error("Failed to update event", err)
		c.JSON(getErrorCode(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (eh *EventHandler) DeleteEvent(c *ginext.Context) {
	eloger := (c.Value(model.LoggerCtxName)).(logger.Logger)
	newEvent := model.Event{}
	if err := c.ShouldBind(&newEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := eh.es.DeleteEvent(newEvent); err != nil {
		eloger.Error("Failed to delete event", err)
		c.JSON(getErrorCode(err), gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (eh *EventHandler) GetDayEvents(c *ginext.Context) {
	eloger := (c.Value(model.LoggerCtxName)).(logger.Logger)
	id, date, err := getUserIDandDate(c)
	if err != nil {
		eloger.Error("Failed to parse userID and/or date to get day-events list", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, err := eh.es.GetDayEvents(uint(id), date)
	if err != nil {
		eloger.Error("Failed to get events list for specific day", err)
		c.JSON(getErrorCode(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

func (eh *EventHandler) GetWeekEvents(c *ginext.Context) {
	eloger := (c.Value(model.LoggerCtxName)).(logger.Logger)
	id, date, err := getUserIDandDate(c)
	if err != nil {
		eloger.Error("Failed to parse userID and/or date to get week-events list", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, err := eh.es.GetWeekEvents(uint(id), date)
	if err != nil {
		eloger.Error("Failed to get events list for specific week", err)
		c.JSON(getErrorCode(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

func (eh *EventHandler) GetMonthEvents(c *ginext.Context) {
	eloger := (c.Value(model.LoggerCtxName)).(logger.Logger)
	id, date, err := getUserIDandDate(c)
	if err != nil {
		eloger.Error("Failed to parse userID and/or date to get month-events list", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events, err := eh.es.GetMonthEvents(uint(id), date)
	if err != nil {
		eloger.Error("Failed to get events list for specific month", err)
		c.JSON(getErrorCode(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

func getUserIDandDate(c *ginext.Context) (int, *model.CustomTime, error) {
	rawID, uOK := c.Params.Get("user_id")
	if !uOK || rawID == "" {
		return 0, nil, errors.New("empty user ID")
	}

	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		return 0, nil, errors.New("incorrect user ID")
	}

	rawDate, dOK := c.Params.Get("date")
	if !dOK {
		return 0, nil, errors.New("empty date")
	}

	startDate := model.CustomTime{}
	if err := startDate.UnmarshalJSON([]byte(rawDate)); err != nil {
		return 0, nil, fmt.Errorf("date parse error: %v", err)
	}

	return id, &startDate, nil
}

func getErrorCode(e error) int {
	switch {
	case errors.Is(e, model.ErrUserIDNotSpecified) ||
		errors.Is(e, model.ErrEventIDNotSpecified) ||
		errors.Is(e, model.ErrNothingToDelete) ||
		errors.Is(e, model.ErrDateNotSpecified) ||
		errors.Is(e, model.ErrEventNotSpecified) ||
		errors.Is(e, model.ErrNothingToUpdate) ||
		errors.Is(e, model.ErrNothingToCreate) ||
		errors.Is(e, model.ErrEventTimePast) ||
		errors.Is(e, model.ErrEventDescrEmpty):
		return 400
	case errors.Is(e, model.ErrUserIDNotFound) ||
		errors.Is(e, model.ErrEventDescrEmpty):
		return 404
	}
	return 500
}
