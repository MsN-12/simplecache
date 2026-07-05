package simplecache

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewRejectsInvalidTTL(t *testing.T) {
	_, err := New[string, int](0, Identity[int])
	if !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("expected ErrInvalidTTL, got %v", err)
	}
}

func TestNewRejectsNilCloneFunc(t *testing.T) {
	_, err := New[string, int](time.Minute, nil)
	if !errors.Is(err, ErrNilCloneFunc) {
		t.Fatalf("expected ErrNilCloneFunc, got %v", err)
	}
}

func TestSetGet(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("answer", 42)

	got, ok := cache.Get("answer")
	if !ok {
		t.Fatal("expected value to exist")
	}

	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestDelete(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("answer", 42)
	cache.Delete("answer")

	_, ok := cache.Get("answer")
	if ok {
		t.Fatal("expected value to be deleted")
	}
}

func TestGetExpiresEntries(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("answer", 42)
	expireKey(t, cache, "answer")

	_, ok := cache.Get("answer")
	if ok {
		t.Fatal("expected value to expire")
	}

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected expired value to be removed, got len %d", got)
	}
}

func TestDeleteExpired(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("expired", 1)
	expireKey(t, cache, "expired")
	cache.Set("fresh", 2)

	removed := cache.DeleteExpired()
	if removed != 1 {
		t.Fatalf("expected 1 removed entry, got %d", removed)
	}

	if _, ok := cache.Get("expired"); ok {
		t.Fatal("expected expired entry to be removed")
	}

	got, ok := cache.Get("fresh")
	if !ok {
		t.Fatal("expected fresh entry to remain")
	}

	if got != 2 {
		t.Fatalf("expected fresh value 2, got %d", got)
	}
}

func expireKey[K comparable, V any](t *testing.T, cache *Cache[K, V], key K) {
	t.Helper()

	cache.mu.Lock()
	defer cache.mu.Unlock()

	item, ok := cache.items[key]
	if !ok {
		t.Fatalf("expected key %v to exist", key)
	}

	item.expiresAt = time.Now().Add(-time.Second)
	cache.items[key] = item
}

func TestSetClonesSlice(t *testing.T) {
	cache := MustNew[string, []int](time.Minute, CloneSlice[int])

	value := []int{1, 2, 3}
	cache.Set("numbers", value)
	value[0] = 99

	got, ok := cache.Get("numbers")
	if !ok {
		t.Fatal("expected value to exist")
	}

	if got[0] != 1 {
		t.Fatalf("expected cached slice to be protected from Set caller mutation, got %v", got)
	}
}

func TestGetClonesSlice(t *testing.T) {
	cache := MustNew[string, []int](time.Minute, CloneSlice[int])

	cache.Set("numbers", []int{1, 2, 3})

	got, ok := cache.Get("numbers")
	if !ok {
		t.Fatal("expected value to exist")
	}

	got[0] = 99

	gotAgain, ok := cache.Get("numbers")
	if !ok {
		t.Fatal("expected value to exist")
	}

	if gotAgain[0] != 1 {
		t.Fatalf("expected cached slice to be protected from Get caller mutation, got %v", gotAgain)
	}
}

func TestMapClone(t *testing.T) {
	cache := MustNew[string, map[string]int](time.Minute, CloneMap[string, int])

	value := map[string]int{"one": 1}
	cache.Set("numbers", value)
	value["one"] = 99

	got, ok := cache.Get("numbers")
	if !ok {
		t.Fatal("expected value to exist")
	}

	got["one"] = 100

	gotAgain, ok := cache.Get("numbers")
	if !ok {
		t.Fatal("expected value to exist")
	}

	if gotAgain["one"] != 1 {
		t.Fatalf("expected cached map to be protected from caller mutation, got %v", gotAgain)
	}
}

func TestCustomDeepClone(t *testing.T) {
	type user struct {
		Name string
		Tags []string
	}

	cloneUser := func(value user) user {
		value.Tags = CloneSlice(value.Tags)
		return value
	}

	cache := MustNew[string, user](time.Minute, cloneUser)

	original := user{Name: "mohsen", Tags: []string{"go", "cache"}}
	cache.Set("user", original)
	original.Tags[0] = "changed"

	got, ok := cache.Get("user")
	if !ok {
		t.Fatal("expected value to exist")
	}

	got.Tags[1] = "changed"

	gotAgain, ok := cache.Get("user")
	if !ok {
		t.Fatal("expected value to exist")
	}

	if gotAgain.Tags[0] != "go" || gotAgain.Tags[1] != "cache" {
		t.Fatalf("expected custom clone to protect nested mutable fields, got %+v", gotAgain)
	}
}

func TestConcurrentAccess(t *testing.T) {
	cache := MustNew[int, []int](time.Minute, CloneSlice[int])

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			cache.Set(i, []int{i})
			got, ok := cache.Get(i)
			if !ok {
				t.Errorf("expected key %d to exist", i)
				return
			}

			got[0] = -1
			cache.Delete(i)
		}(i)
	}

	wg.Wait()
}
