// Package notifier launches as a goroutine and processes events taken from the heap
package notifier

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
)

type eventRepository interface {
	GetNextEventTime() (time.Time, error)
	MarkEventDone(uid uint, eid string) bool
	PopNearestEvent() (*model.Event, error)
}

func RunNotifier(ctx context.Context, wg *sync.WaitGroup, repo eventRepository, updCh <-chan struct{}) {
	wg.Add(1)
	log.Println("Launching event-notifier...")

	go func() {
		defer wg.Done()

		var nexTime time.Time
		var err error

		for {
			nexTime, err = repo.GetNextEventTime()
			if err == nil {
				break
			}

			// ждём сигнал о новом событии
			select {
			case <-updCh:
				continue // получили сигнал, пробуем снова
			case <-time.After(1 * time.Minute):
				continue // или таймаут
			}
		}

		ticker := time.NewTimer(nexTime.Sub(time.Now().UTC()))

		log.Println("Event-notifier is successfully launched.")

		for {
			select {
			case <-ctx.Done():
				log.Println("Notifier's ctx is cancelled. Exiting notifier...")
				return
			case <-updCh:
				nexTime, err = repo.GetNextEventTime()
				now := time.Now().UTC()
				if err == nil {
					ticker.Stop()
					ticker.Reset(nexTime.Sub(now))
				}
			case <-ticker.C:
				// обрабатываем наступивший ивент
				poppedEvent, popErr := repo.PopNearestEvent()
				if popErr != nil || poppedEvent == nil {
					continue
				}

				if ok := repo.MarkEventDone(poppedEvent.UID, poppedEvent.EID); !ok {
					log.Printf("Event %q is not found in repo.", poppedEvent.EID)
				} else {
					log.Printf("Event %q with descr. %q is marked as done.", poppedEvent.EID, poppedEvent.Description)
				}

				// пересчитываем таймер сна на следующий ивент
				now := time.Now().UTC()
				if nexTime, err = repo.GetNextEventTime(); err == nil {
					ticker.Stop()
					ticker.Reset(nexTime.Sub(now))
				}
			}
		}
	}()
}
