package engine

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type SSEEvent struct {
	Event string
	Data  any
}

type Hub struct {
	mu   sync.RWMutex
	subs map[uuid.UUID]map[chan SSEEvent]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[uuid.UUID]map[chan SSEEvent]struct{}{}}
}

func (h *Hub) Subscribe(taskID uuid.UUID) chan SSEEvent {
	ch := make(chan SSEEvent, 128)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[taskID] == nil {
		h.subs[taskID] = map[chan SSEEvent]struct{}{}
	}
	h.subs[taskID][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(taskID uuid.UUID, ch chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.subs[taskID]; ok {
		delete(m, ch)
		close(ch)
		if len(m) == 0 {
			delete(h.subs, taskID)
		}
	}
}

func (h *Hub) Publish(taskID uuid.UUID, ev SSEEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if m, ok := h.subs[taskID]; ok {
		for ch := range m {
			select {
			case ch <- ev:
			default:
				// drop if slow consumer
			}
		}
	}
}

func (h *Hub) PublishJSON(taskID uuid.UUID, event string, payload any) {
	b, _ := json.Marshal(payload)
	var obj any
	_ = json.Unmarshal(b, &obj)
	h.Publish(taskID, SSEEvent{Event: event, Data: obj})
}
