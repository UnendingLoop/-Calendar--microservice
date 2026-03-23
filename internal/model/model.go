// Package model describes data-structures for the app
package model

import (
	"strings"
	"time"
)

type logCtxName string

var LoggerCtxName = logCtxName("logger")

type CustomTime struct {
	time.Time
}

type Event struct {
	EID         string      `json:"event_id"`                   // id задачи/события
	UID         uint        `json:"user_id" binding:"required"` // id создателя задачи
	Created     time.Time   `json:"created"`                    // дата создания задачи/события в UTC, для внутреннего использования
	Updated     time.Time   `json:"-"`                          // дата обновления события/задачи пользователем в UTC, для внутреннего использования
	Scheduled   *CustomTime `json:"date" binding:"required"`    // дата выполнения/наступления задачи/события
	Description string      `json:"event" binding:"required"`   // сам текст события/задачи
	IsDone      bool        `json:"is_done"`                    // флаг "отправленности"/выполненности задачи/события
}

type DeleteDTO struct {
	EID string `json:"event_id" binding:"required"` // id задачи/события
	UID uint   `json:"user_id" binding:"required"`  // id создателя задачи
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		ct.Time = time.Time{}
		return nil
	}

	loc := time.Local

	t, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
	if err != nil {
		return err
	}

	ct.Time = t.UTC()
	return nil
}

func (ct *CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + ct.Time.Local().Format("2006-01-02 15:04") + `"`), nil
}

type EventRecord struct {
	Event *Event
}
