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

func TestSetWithTTL(t *testing.T) {
	cache := MustNew[string, int](time.Hour, Identity[int])

	err := cache.SetWithTTL("short", 1, time.Nanosecond)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	for cache.Has("short") {
		time.Sleep(time.Millisecond)
	}

	if _, ok := cache.Get("short"); ok {
		t.Fatal("expected value to expire using custom ttl")
	}
}

func TestSetWithTTLRejectsInvalidTTL(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	err := cache.SetWithTTL("answer", 42, 0)
	if !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("expected ErrInvalidTTL, got %v", err)
	}

	if cache.Has("answer") {
		t.Fatal("expected invalid ttl value not to be stored")
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

func TestHas(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	if cache.Has("answer") {
		t.Fatal("expected missing key not to exist")
	}

	cache.Set("answer", 42)
	if !cache.Has("answer") {
		t.Fatal("expected fresh key to exist")
	}
}

func TestHasRemovesExpiredEntry(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("answer", 42)
	expireKey(t, cache, "answer")

	if cache.Has("answer") {
		t.Fatal("expected expired key not to exist")
	}

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected expired key to be removed, got len %d", got)
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

func TestClear(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("one", 1)
	cache.Set("two", 2)
	cache.Clear()

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected empty cache, got len %d", got)
	}

	if cache.Has("one") || cache.Has("two") {
		t.Fatal("expected all keys to be removed")
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

func TestLenFresh(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("expired", 1)
	expireKey(t, cache, "expired")
	cache.Set("fresh", 2)

	if got := cache.Len(); got != 2 {
		t.Fatalf("expected Len to include expired entries before cleanup, got %d", got)
	}

	if got := cache.LenFresh(); got != 1 {
		t.Fatalf("expected LenFresh to count only fresh entries, got %d", got)
	}

	if cache.Has("expired") {
		t.Fatal("expected LenFresh to remove expired entries")
	}
}

func TestStartCleanupRejectsInvalidInterval(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	err := cache.StartCleanup(0)
	if !errors.Is(err, ErrInvalidCleanupInterval) {
		t.Fatalf("expected ErrInvalidCleanupInterval, got %v", err)
	}
}

func TestStartCleanupRejectsAlreadyRunningCleanup(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	if err := cache.StartCleanup(time.Millisecond); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer cache.StopCleanup()

	err := cache.StartCleanup(time.Millisecond)
	if !errors.Is(err, ErrCleanupAlreadyRunning) {
		t.Fatalf("expected ErrCleanupAlreadyRunning, got %v", err)
	}
}

func TestStartCleanupRemovesExpiredEntries(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.Set("expired", 1)
	expireKey(t, cache, "expired")

	if err := cache.StartCleanup(time.Millisecond); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer cache.StopCleanup()

	waitUntil(t, func() bool {
		return cache.Len() == 0
	})
}

func TestStopCleanupIsIdempotentAndCanRestart(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	cache.StopCleanup()
	if err := cache.StartCleanup(time.Millisecond); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	cache.StopCleanup()

	if err := cache.StartCleanup(time.Millisecond); err != nil {
		t.Fatalf("expected cleanup to restart, got %v", err)
	}
	cache.StopCleanup()
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

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatal("condition was not met before deadline")
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

func TestCloneHelpersHandleNil(t *testing.T) {
	if got := CloneSlice[int](nil); got != nil {
		t.Fatalf("expected nil slice, got %v", got)
	}

	if got := CloneBytes(nil); got != nil {
		t.Fatalf("expected nil bytes, got %v", got)
	}

	if got := CloneMap[string, int](nil); got != nil {
		t.Fatalf("expected nil map, got %v", got)
	}
}

func TestIdentity(t *testing.T) {
	if got := Identity(42); got != 42 {
		t.Fatalf("expected 42, got %d", got)
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

func TestGetOrSetReturnsCachedValue(t *testing.T) {
	cache := MustNew[string, []int](time.Minute, CloneSlice[int])
	cache.Set("numbers", []int{1, 2, 3})

	called := false
	got, cached, err := cache.GetOrSet("numbers", func() ([]int, error) {
		called = true
		return []int{4, 5, 6}, nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !cached {
		t.Fatal("expected cached result")
	}
	if called {
		t.Fatal("expected load function not to be called")
	}

	got[0] = 99
	gotAgain, _ := cache.Get("numbers")
	if gotAgain[0] != 1 {
		t.Fatalf("expected cached value to be cloned, got %v", gotAgain)
	}
}

func TestGetOrSetStoresComputedValue(t *testing.T) {
	cache := MustNew[string, []int](time.Minute, CloneSlice[int])

	computed := []int{1, 2, 3}
	got, cached, err := cache.GetOrSet("numbers", func() ([]int, error) {
		return computed, nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cached {
		t.Fatal("expected computed result")
	}

	computed[0] = 99
	got[1] = 88

	gotAgain, ok := cache.Get("numbers")
	if !ok {
		t.Fatal("expected computed value to be stored")
	}
	if gotAgain[0] != 1 || gotAgain[1] != 2 {
		t.Fatalf("expected stored computed value to be cloned, got %v", gotAgain)
	}
}

func TestGetOrSetDoesNotStoreOnError(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])
	expectedErr := errors.New("load failed")

	got, cached, err := cache.GetOrSet("answer", func() (int, error) {
		return 42, expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected load error, got %v", err)
	}
	if cached {
		t.Fatal("expected uncached result")
	}
	if got != 0 {
		t.Fatalf("expected zero value, got %d", got)
	}
	if cache.Has("answer") {
		t.Fatal("expected failed load not to be stored")
	}
}

func TestGetOrSetRejectsNilLoadFunc(t *testing.T) {
	cache := MustNew[string, int](time.Minute, Identity[int])

	_, _, err := cache.GetOrSet("answer", nil)
	if !errors.Is(err, ErrNilLoadFunc) {
		t.Fatalf("expected ErrNilLoadFunc, got %v", err)
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

func TestConcurrentNewMethods(t *testing.T) {
	cache := MustNew[int, int](time.Minute, Identity[int])

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			_ = cache.SetWithTTL(i, i, time.Minute)
			_, _, err := cache.GetOrSet(i, func() (int, error) {
				return i, nil
			})
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
			_ = cache.Has(i)
			_ = cache.LenFresh()
			if i%10 == 0 {
				cache.Clear()
			}
		}(i)
	}

	wg.Wait()
}
