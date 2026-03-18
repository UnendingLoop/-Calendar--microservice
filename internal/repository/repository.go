// Package repository provides CRUD-methods to eventsmap
package repository

import (
	"container/heap"
	"sync"
	"time"

	"github.com/UnendingLoop/-Calendar--microservice/internal/model"
)

type SecureEventsMap struct {
	eventMap map[uint]model.EventHeap // мапа для будущих ивентов
	archive  map[uint]model.EventHeap // архив для прошедших ивентов
	eh       model.EventHeap          // куча для упорядочивания ивентов по времени
	updateCh chan<- struct{}          // канал для оповещения о необходимости пересчета времени до ближайшего события
	mu       sync.RWMutex             // общий мьютекс для всех операций
}

func NewEventRepository(uc chan<- struct{}, emap, arch map[uint]model.EventHeap) *SecureEventsMap {
	cleanHeap := convertMapToSliceNormalize(emap, arch)
	heap.Init(&cleanHeap)

	return &SecureEventsMap{
		eventMap: emap,
		archive:  arch,
		eh:       cleanHeap,
		updateCh: uc,
		mu:       sync.RWMutex{},
	}
}

func (sem *SecureEventsMap) CreateEvent(event model.Event) {
	sem.mu.Lock()
	defer sem.mu.Unlock()
	// упаковываем ивент в элемент кучи
	newHeapEntity := &model.HeapEntity{
		Index: 0,
		Event: &event,
	}
	// кладем новый ивент в кучу
	sem.eh.Push(newHeapEntity)
	// кладем в мапу
	sem.eventMap[event.UID] = append(sem.eventMap[event.UID], newHeapEntity)
	// проверяем, надо ли будить горутину-слушатель для пересчета времени
	if sem.eh[0].Event.Scheduled.After(event.Scheduled.Time) || event.Scheduled.Before(time.Now().UTC()) {
		select {
		case sem.updateCh <- struct{}{}:
		default:
		}
	}
}

func (sem *SecureEventsMap) UpdateEvent(uid uint, event model.Event) *model.Event {
	sem.mu.Lock()
	defer sem.mu.Unlock()

	userEvents, ok := sem.eventMap[uid]
	if !ok {
		return nil
	}

	for _, v := range userEvents {
		if v.Event.EID == event.EID {
			// обновляем описание ивента
			if event.Description != "" {
				v.Event.Description = event.Description
			}

			// разбираемся с новым временем - если указано новое
			if event.Scheduled != nil && !event.Scheduled.Equal(v.Event.Scheduled.Time) {
				v.Event.Scheduled = event.Scheduled
				e, ok := sem.eh.PeekNext()

				heap.Fix(&sem.eh, v.Index)

				switch {
				case ok && e != nil:
					if e.Event.Scheduled.After(event.Scheduled.Time) {
						select {
						case sem.updateCh <- struct{}{}:
						default:
						}
					}
				case !ok:
					select {
					case sem.updateCh <- struct{}{}:
					default:
					}
				}

			}

			v.Event.Updated = time.Now().UTC()

			updatedEvent := *v.Event // чтобы не обращаться к элементу, привязанному к карте вне мьютекса
			return &updatedEvent
		}
	}
	return nil
}

func (sem *SecureEventsMap) DeleteEvent(uid uint, eid string) bool {
	sem.mu.Lock()
	defer sem.mu.Unlock()

	userEvents := sem.eventMap[uid]
	for i, v := range userEvents {
		if v.Event.EID == eid {
			sem.eventMap[uid] = append(userEvents[:i], userEvents[(i+1):]...)
			heap.Remove(&sem.eh, v.Index)
			if v.Index == 0 {
				select {
				case sem.updateCh <- struct{}{}:
				default:
				}
			}
			return true
		}
	}

	userArchive := sem.archive[uid]
	for i, v := range userArchive {
		if v.Event.EID == eid {
			sem.archive[uid] = append(userArchive[:i], userArchive[(i+1):]...)
			return true
		}
	}

	return false
}

func (sem *SecureEventsMap) GetPeriodEvents(uid uint, start, end *time.Time) []model.Event {
	sem.mu.RLock()
	defer sem.mu.RUnlock()

	// достаем актуальные ивенты
	userEvents, ok := sem.eventMap[uid]
	if !ok {
		return nil
	}

	// если start в прошлом, проверяем архивные записи и добавляем их в выборку
	if start.Before(time.Now().UTC()) {
		userArchive, ok := sem.archive[uid]
		if ok {
			userArchive = append(userArchive, userArchive...)
			sem.archive[uid] = userArchive
		}
	}

	result := []model.Event{}

	for _, v := range userEvents {
		if !v.Event.Scheduled.Before(*start) && !v.Event.Scheduled.After(*end) {
			result = append(result, *v.Event)
		}
	}

	return result
}

func (sem *SecureEventsMap) ArchiveExpired() int {
	sem.mu.Lock()
	defer sem.mu.Unlock()

	nArchived := 0

	for uid, events := range sem.eventMap {
		archE := model.EventHeap{}
		actualE := model.EventHeap{}

		for _, e := range events {
			switch e.Event.IsDone { // архивируем только выполненные ивенты
			case true:
				e.Index = -1
				archE = append(archE, e)
			case false:
				actualE = append(actualE, e)
			}
		}

		sem.eventMap[uid] = actualE
		sem.archive[uid] = archE

		nArchived += len(archE)
	}

	return nArchived
}

func (sem *SecureEventsMap) MarkEventDone(uid uint, eid string) bool {
	sem.mu.Lock()
	defer sem.mu.Unlock()

	userEvents := sem.eventMap[uid]

	for _, v := range userEvents {
		if v.Event.EID == eid {
			v.Event.IsDone = true
			return true
		}
	}

	return false
}

func (sem *SecureEventsMap) GetNextEventTime() (time.Time, error) {
	sem.mu.RLock()
	defer sem.mu.RUnlock()
	e, ok := sem.eh.PeekNext()
	if !ok {
		return time.Time{}, model.ErrNoEventInQueue
	}
	return e.Event.Scheduled.Time, nil
}

func (sem *SecureEventsMap) PopNearestEvent() (*model.Event, error) {
	event := heap.Pop(&sem.eh).(*model.HeapEntity)
	if event != nil {
		return event.Event, nil
	}

	return nil, model.ErrNoEventInQueue
}

func (sem *SecureEventsMap) SafeLockMap() {
	sem.mu.RLock()
}

func (sem *SecureEventsMap) SafeUnlockMap() {
	sem.mu.RUnlock()
}

func convertMapToSliceNormalize(emap, arch map[uint]model.EventHeap) model.EventHeap {
	resHeap := model.EventHeap{}

	for k, v := range emap {
		res := model.EventHeap{}
		for _, e := range v {
			if e.Event.IsDone {
				arch[k] = append(arch[k], e)
				continue
			}
			res = append(res, e)
		}
		emap[k] = res
		resHeap = append(resHeap, res...)
	}

	return resHeap
}
