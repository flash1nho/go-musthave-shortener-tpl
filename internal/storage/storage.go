package storage

import (
		"sync"
)

type Storage struct {
		mu sync.RWMutex
		data map[string]string
}

func NewStorage() *Storage {
		return &Storage{
				data: make(map[string]string),
		}
}

func (s *Storage) Set(key, value string) {
		s.mu.Lock()
	  defer s.mu.Unlock()
	  s.data[key] = value
}

func (s *Storage) Get(key string) (string, bool) {
	  s.mu.RLock()
	  defer s.mu.RUnlock()
	  value, found := s.data[key]

	  return value, found
}
