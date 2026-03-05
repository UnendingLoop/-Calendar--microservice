// Package model describes data-structures for the app
package model

import (
	"strings"
	"time"
)

var LoggerCtxName = "logger"

type CustomTime struct {
	time.Time
}

type Event struct {
	EID         string      `json:"event_id"`                             // id задачи/события
	UID         uint        `json:"user_id,omitempty" binding:"required"` // id создателя задачи
	Created     time.Time   `json:"created"`                              // дата создания задачи/события в UTC, для внутреннего использования
	Updated     time.Time   `json:"updated"`                              // дата обновления события/задачи пользователем в UTC, для внутреннего использования
	Scheduled   *CustomTime `json:"date"`                                 // дата выполнения/наступления задачи/события
	Description string      `json:"event"`                                // сам текст события/задачи
	IsDone      bool        `json:"is_done"`                              // флаг "отправленности"/выполненности задачи/события
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		ct.Time = time.Time{}
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	ct.Time = t.UTC()
	return nil
}

func (ct *CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + ct.Time.Format("2006-01-02") + `"`), nil
}

type EventRecord struct {
	Event *Event
}
