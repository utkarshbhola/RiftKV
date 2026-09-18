package main

import (
	"sync"
	"time"
)

type Entry struct {
	value      string
	expiration time.Time
}

type Store struct {
	mu   sync.RWMutex
	data map[string]Entry
	wal  *WAL
}

func NewStore(filename string) (*Store, error) {
	if filename == "" {
		filename = "riftkv.wal"
	}

	wal, err := OpenWAL(filename)
	if err != nil {
		return nil, err
	}

	store := &Store{
		data: make(map[string]Entry),
		wal:  wal,
	}

	if err := wal.Replay(store); err != nil {
		_ = wal.Close()
		return nil, err
	}

	go store.ExpireLoop()
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.wal == nil {
		return nil
	}
	return s.wal.Close()
}

func (s *Store) Set(key, value string, ttl time.Duration) error {
	expiration := time.Time{}
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	record := WALRecord{
		Operation:  "SET",
		Key:        key,
		Value:      value,
		Expiration: expiration.UnixMilli(),
	}

	if err := s.wal.Append(record); err != nil {
		return err
	}

	s.mu.Lock()
	s.data[key] = Entry{value: value, expiration: expiration}
	s.mu.Unlock()

	return nil
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	entry, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return "", false
	}

	if !entry.expiration.IsZero() && time.Now().After(entry.expiration) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return "", false
	}

	return entry.value, true
}

func (s *Store) Delete(key string) error {
	record := WALRecord{Operation: "DEL", Key: key}
	if err := s.wal.Append(record); err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()

	return nil
}

func (s *Store) Exists(key string) bool {
	_, ok := s.Get(key)
	return ok
}

func (s *Store) applySet(key, value string, expiration int64) {
	expiresAt := time.Time{}
	if expiration > 0 {
		expiresAt = time.UnixMilli(expiration)
		if time.Now().After(expiresAt) {
			return
		}
	}

	s.mu.Lock()
	s.data[key] = Entry{value: value, expiration: expiresAt}
	s.mu.Unlock()
}

func (s *Store) applyDelete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
}

func (s *Store) ExpireLoop() {
	for {
		time.Sleep(100 * time.Millisecond)

		s.mu.Lock()
		for key, entry := range s.data {
			if !entry.expiration.IsZero() && time.Now().After(entry.expiration) {
				delete(s.data, key)
			}
		}
		s.mu.Unlock()
	}
}

