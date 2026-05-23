package realtime

import (
	"sync"

	"github.com/observa/observad/internal/model"
)

type subscriber struct {
	id     uint64
	ch     chan model.Log
	filter Filter
}

type Filter struct {
	Service string
	Level   model.Level
}

func (f Filter) allows(l model.Log) bool {
	if f.Service != "" && f.Service != l.Service {
		return false
	}
	if f.Level != "" && f.Level != l.Level {
		return false
	}
	return true
}

type Hub struct {
	mu     sync.RWMutex
	subs   map[uint64]*subscriber
	nextID uint64
}

func NewHub() *Hub {
	return &Hub{subs: make(map[uint64]*subscriber)}
}

func (h *Hub) Subscribe(f Filter) (<-chan model.Log, func()) {
	h.mu.Lock()
	id := h.nextID
	h.nextID++
	s := &subscriber{
		id:     id,
		ch:     make(chan model.Log, 256),
		filter: f,
	}
	h.subs[id] = s
	h.mu.Unlock()

	return s.ch, func() { h.unsubscribe(id) }
}

func (h *Hub) unsubscribe(id uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.subs[id]; ok {
		delete(h.subs, id)
		close(s.ch)
	}
}

// Broadcast drops messages for slow subscribers instead of blocking.
func (h *Hub) Broadcast(logs []model.Log) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, l := range logs {
		for _, s := range h.subs {
			if !s.filter.allows(l) {
				continue
			}
			select {
			case s.ch <- l:
			default:
			}
		}
	}
}

func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
