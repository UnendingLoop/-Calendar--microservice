package model

type HeapEntity struct {
	Index int    `json:"heap_id"`
	Event *Event `json:"event"`
}

type EventHeap []*HeapEntity

func (h EventHeap) Len() int {
	return len(h)
}

func (h EventHeap) Less(i, j int) bool {
	return h[i].Event.Scheduled.Before(h[j].Event.Scheduled.Time)
}

func (h EventHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *EventHeap) Push(x any) {
	*h = append(*h, x.(*HeapEntity))
}

func (h *EventHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]

	old[n-1] = nil

	*h = old[:n-1]
	return item
}

func (h EventHeap) PeekNext() (*HeapEntity, bool) {
	if h.Len() == 0 {
		return nil, false
	}
	return h[0], true
}
