// Package model describes data-structures for the app
package model

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CustomTime struct {
	time.Time
}

type Event struct {
	EID       uuid.UUID   `json:"event_id,omitempty"` // id задачи/события
	Created   time.Time   `json:"-"`                  // дата создания задачи/события в UTC, для внутреннего использования
	Updated   time.Time   `json:"-"`                  // дата обновления события/задачи пользователем в UTC, для внутреннего использования
	Scheduled *CustomTime `json:"date,omitempty"`     // дата выполнения/наступления задачи/события
	Task      string      `json:"event,omitempty"`    // сам текст события/задачи
}

type UserID uint // ID пользователя, создавшего событие

type Combined struct {
	Event
	UserID `json:"user_id"`
}

type CalendArray struct {
	EventMap map[UserID][]*Event
	sync.RWMutex
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
	ct.Time = t
	return nil
}

func (ct *CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + ct.Time.Format("2006-01-02") + `"`), nil
}
