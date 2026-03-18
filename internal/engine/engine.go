// Package engine returns configured Server(for main) and Engine(for tests)
package engine

import (
	"context"
	"net/http"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/logger"
	"github.com/wb-go/wbf/ginext"
)

type handler interface {
	CreateEvent(c *ginext.Context)
	DeleteEvent(c *ginext.Context)
	GetDayEvents(c *ginext.Context)
	GetMonthEvents(c *ginext.Context)
	GetWeekEvents(c *ginext.Context)
	SimplePinger(c *ginext.Context)
	UpdateEvent(c *ginext.Context)
}

func NewServer(ctx context.Context, mode, port string, h handler, eventLogger logger.Logger) *http.Server {
	engine := ginext.New(mode)

	engine.Use(eventLogger.RequestLogger()) // логирование запросов
	engine.GET("/ping", h.SimplePinger)

	events := engine.Group("/events")

	events.POST("", h.CreateEvent)             // создание нового события
	events.PATCH("", h.UpdateEvent)            // обновление существующего
	events.DELETE("", h.DeleteEvent)           // удаление существующего
	events.GET("/for_day", h.GetDayEvents)     // получить все события на день ?user_id=1&date=2023-12-31
	events.GET("/for_week", h.GetWeekEvents)   // события на неделю ?user_id=1&date=2023-12-31
	events.GET("/for_month", h.GetMonthEvents) // события на месяц ?user_id=1&date=2023-12-31

	return &http.Server{
		Addr:         ":" + port,
		Handler:      engine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
