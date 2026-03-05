// Package engine returns configured Server(for main) and Engine(for tests)
package engine

import (
	"context"
	"net/http"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/logger"
	"github.com/UnendingLoop/-Calendar--microservice/internal/transport"
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/ginext"
)

func NewServer(ctx context.Context, c *config.Config, h *transport.EventHandler, eventLogger logger.Logger) *http.Server {
	engine := ginext.New(c.GetString("GIN_MODE"))

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
		Addr:         ":" + c.GetString("APP_PORT"),
		Handler:      engine,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
