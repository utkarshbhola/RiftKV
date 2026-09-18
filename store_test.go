package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreSetAndGet(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "test.wal"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if err := store.Set("alpha", "beta", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	value, ok := store.Get("alpha")
	if !ok {
		t.Fatal("Get() expected key to exist")
	}
	if value != "beta" {
		t.Fatalf("Get() = %q, want %q", value, "beta")
	}
}

func TestStoreReplayOnRestart(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "restart.wal")

	store1, err := NewStore(walPath)
	if err != nil {
		t.Fatalf("NewStore() first instance error = %v", err)
	}

	if err := store1.Set("city", "mumbai", 0); err != nil {
		t.Fatalf("Set() first instance error = %v", err)
	}
	if err := store1.Close(); err != nil {
		t.Fatalf("Close() first instance error = %v", err)
	}

	store2, err := NewStore(walPath)
	if err != nil {
		t.Fatalf("NewStore() second instance error = %v", err)
	}
	defer store2.Close()

	value, ok := store2.Get("city")
	if !ok {
		t.Fatal("expected persisted key to be replayed after restart")
	}
	if value != "mumbai" {
		t.Fatalf("replayed value = %q, want %q", value, "mumbai")
	}
}

func TestStoreTTLExpiration(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "ttl.wal"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if err := store.Set("temp", "value", 50*time.Millisecond); err != nil {
		t.Fatalf("Set() with TTL error = %v", err)
	}

	time.Sleep(120 * time.Millisecond)

	if _, ok := store.Get("temp"); ok {
		t.Fatal("expected expired TTL key to be removed")
	}
}
