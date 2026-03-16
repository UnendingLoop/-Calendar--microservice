package notifier

import (
	"testing"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
)

type mockRep struct{}

func (mr mockRep) GetNextEventTime() (time.Time, error) {
	return time.Time{}, nil
}
func (mr mockRep) MarkEventDone(uid uint, eid string) bool {}
func (mr mockRep) PopNearestEvent() (*model.Event, error)  {}

func TestRunNotifier(t *testing.T) {
}
