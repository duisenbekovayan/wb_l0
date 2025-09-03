package cache

import (
	"sync"
	"time"

	model "github.com/duisenbekovayan/wb_l0/internal/models"
)

type Store struct {
	mu sync.RWMutex
	m  map[string]model.Order
}

func New() *Store { return &Store{m: make(map[string]model.Order)} }

func (s *Store) Get(id string) (model.Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.m[id]
	return o, ok
}

func (s *Store) Set(o model.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[o.OrderUID] = o
}

// optional: ttl (если надо)
func (s *Store) Sweep(_ time.Duration) {}
