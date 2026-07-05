package simplecache

import (
	"testing"
	"time"
)

func BenchmarkSetInt(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(i, i)
	}
}

func BenchmarkGetInt(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])
	cache.Set(1, 1)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(1)
	}
}

func BenchmarkGetMiss(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(i)
	}
}

func BenchmarkSetWithTTLInt(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.SetWithTTL(i, i, time.Minute)
	}
}

func BenchmarkHasInt(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])
	cache.Set(1, 1)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.Has(1)
	}
}

func BenchmarkGetOrSetHit(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])
	cache.Set(1, 1)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = cache.GetOrSet(1, func() (int, error) {
			return 2, nil
		})
	}
}

func BenchmarkGetOrSetMiss(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = cache.GetOrSet(i, func() (int, error) {
			return i, nil
		})
	}
}

func BenchmarkSetBytes(b *testing.B) {
	cache := MustNew[int, []byte](time.Minute, CloneBytes)
	value := make([]byte, 1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(i, value)
	}
}

func BenchmarkGetBytes(b *testing.B) {
	cache := MustNew[int, []byte](time.Minute, CloneBytes)
	cache.Set(1, make([]byte, 1024))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(1)
	}
}

func BenchmarkDeleteExpired(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])
	expiresAt := time.Now().Add(-time.Second)
	const entries = 100

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < entries; j++ {
			cache.items[j] = entry[int]{id: uint64(j + 1), value: j, expiresAt: expiresAt}
		}

		_ = cache.DeleteExpired()
	}
}

func BenchmarkConcurrentGetSet(b *testing.B) {
	cache := MustNew[int, int](time.Minute, Identity[int])

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Set(1, 1)
			_, _ = cache.Get(1)
		}
	})
}

func BenchmarkDeepCloneNestedStruct(b *testing.B) {
	value := testUser{
		ID:    "u-123",
		Name:  "Alice",
		Email: "alice@example.com",
		Profile: testProfile{
			Bio:     "Software Engineer",
			Age:     30,
			Address: testAddress{Street: "123 Main St", City: "Berlin", Country: "Germany"},
			Tags:    []string{"go", "cache"},
			Counters: map[string][]int{
				"login": []int{1, 2},
			},
		},
		Notifier: &testEmailNotifier{Sent: []string{"created"}},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DeepClone(value)
	}
}

func BenchmarkSetAutoNestedStruct(b *testing.B) {
	cache := MustNewAuto[int, testUser](time.Minute)
	value := testUser{
		ID:    "u-123",
		Name:  "Alice",
		Email: "alice@example.com",
		Profile: testProfile{
			Bio:     "Software Engineer",
			Age:     30,
			Address: testAddress{Street: "123 Main St", City: "Berlin", Country: "Germany"},
			Tags:    []string{"go", "cache"},
			Counters: map[string][]int{
				"login": []int{1, 2},
			},
		},
		Notifier: &testEmailNotifier{Sent: []string{"created"}},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(i, value)
	}
}
