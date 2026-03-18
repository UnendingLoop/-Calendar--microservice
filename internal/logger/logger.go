// Package logger receives events from consumers and sends them to channel
package logger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/ginext"
)

type Logger interface {
	With(fields ...Field) Logger
	Info(msg string)
	Error(msg string, err error)
	RequestLogger() ginext.HandlerFunc
	Shutdown()
}

type Field struct {
	Key   string
	Value any
}

type EventEntry struct {
	Level     string
	Msg       string
	Err       error
	TimeStamp time.Time
	Fields    []Field
}

type AsyncLogger struct {
	ch        chan<- *EventEntry
	event     *EventEntry
	ctx       context.Context
	ctxCancel context.CancelFunc
	wg        *sync.WaitGroup
}

func NewAsyncLogger(ctx context.Context, ch chan<- *EventEntry) Logger {
	lctx, cancel := context.WithCancel(ctx)
	return &AsyncLogger{
		ch:        ch,
		ctx:       lctx,
		ctxCancel: cancel,
		wg:        &sync.WaitGroup{},
		event: &EventEntry{
			Fields: []Field{},
		},
	}
}

func (el *AsyncLogger) WithNewLogger() Logger {
	return &AsyncLogger{
		ch:        el.ch,
		ctx:       el.ctx,
		ctxCancel: el.ctxCancel,
		wg:        el.wg,
		event: &EventEntry{
			Fields: []Field{},
		},
	}
}

func (el *AsyncLogger) With(fields ...Field) Logger {
	newFields := append(el.event.Fields, fields...)
	return &AsyncLogger{
		ch:  el.ch,
		ctx: el.ctx,
		wg:  el.wg,
		event: &EventEntry{
			Fields: newFields,
		},
	}
}

func (el *AsyncLogger) Error(msg string, err error) {
	el.wg.Add(1)
	defer el.wg.Done()

	fieldsCopy := append([]Field(nil), el.event.Fields...)

	entry := &EventEntry{
		Level:     "ERROR",
		Msg:       msg,
		Err:       err,
		TimeStamp: time.Now(),
		Fields:    fieldsCopy,
	}
	select {
	case <-el.ctx.Done():
		return
	case el.ch <- entry:
	}
}

func (el *AsyncLogger) Info(msg string) {
	el.wg.Add(1)
	defer el.wg.Done()

	fieldsCopy := append([]Field(nil), el.event.Fields...)

	entry := &EventEntry{
		Level:     "INFO",
		Msg:       msg,
		Err:       fmt.Errorf("no-error"),
		TimeStamp: time.Now(),
		Fields:    fieldsCopy,
	}

	select {
	case <-el.ctx.Done():
		return
	case el.ch <- entry:
	}
}

func (el *AsyncLogger) Shutdown() {
	el.ctxCancel()
	el.wg.Wait()
	close(el.ch)
}

func (el *AsyncLogger) RequestLogger() ginext.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now().UTC()
		rid := uuid.New().String()
		c.Header("X-Request-ID", rid)

		newLogger := el.WithNewLogger()

		// добавляем в логгер базовую информацию о запросе
		rlog := newLogger.With(
			Field{
				Key:   "Request ID",
				Value: rid,
			},
			Field{
				Key:   "Method",
				Value: c.Request.Method,
			},
			Field{
				Key:   "URL path",
				Value: c.Request.URL.Path,
			},
		)

		// если есть - добавляем userID и eventID в логгер
		uid, eid, err := getUserIDandEventID(c)
		if err != nil {
			rlog.Error("JSON decode error", err)
		}
		if uid != "" {
			rlog.With(Field{Key: "userID", Value: uid})
		}
		if eid != "" {
			rlog.With(Field{Key: "eventID", Value: eid})
		}

		// логируем время начала запроса
		rlog.Info(fmt.Sprintln("Request start time:", start))

		// кладем логгер в контекст запроса
		ctx := context.WithValue(c.Request.Context(), model.LoggerCtxName, rlog)
		c.Request = c.Request.Clone(ctx)

		// передаем вызов дальше
		c.Next()

		// логируем время окончания обработки запроса
		rlog.Info(fmt.Sprintln("Request end time:", time.Now().UTC()))
	}
}

func getUserIDandEventID(c *ginext.Context) (string, string, error) {
	var res struct {
		UserID  string `json:"user_id"`
		EventID string `json:"event_id"`
	}

	// сначала пробуем прочитать JSON из тела
	err := c.ShouldBind(&res)

	// пробуем читать параметры запроса
	if res.UserID == "" {
		uid, uok := c.Params.Get("user_id")
		if uok && uid != "" {
			res.UserID = uid
		}
	}
	if res.EventID == "" {
		eid, eok := c.Params.Get("event_id")
		if eok && eid != "" {
			res.UserID = eid
		}
	}

	return res.UserID, res.EventID, err
}
