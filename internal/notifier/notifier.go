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
	GetNextEventTime() (time.Time, error)    // возврат времени ближайшего события без удаления его из кучи, error - если куча пустая
	MarkEventDone(uid uint, eid string) bool // возвращаемый bool - ивент найден/не найден
	PopNearestEvent() (*model.Event, error)  // достает ближайший ивент и удаляет его из кучи
}

func RunNotifier(ctx context.Context, wg *sync.WaitGroup, repo eventRepository, updCh <-chan struct{}) {
	wg.Add(1)
	log.Println("Launching event-notifier...")

	go func() {
		defer wg.Done()

		var nexTime time.Time
		var err error

		// инициализация работы
		for {
			nexTime, err = repo.GetNextEventTime()
			if err == nil {
				break
			}

			// ждём сигнал о новом событии
			select {
			case <-ctx.Done():
				log.Println("Notifier's ctx is cancelled before completing init. Exiting notifier...")
				return
			case _, ok := <-updCh:
				if !ok {
					log.Println("Notifier's update channel is closed. Exiting notifier...")
					return
				}
			case <-time.After(1 * time.Minute):
				continue // или ждем таймаут
			}
		}

		ticker := time.NewTimer(nexTime.Sub(time.Now().UTC()))

		log.Println("Event-notifier is successfully launched.")

		// основной цикл
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
