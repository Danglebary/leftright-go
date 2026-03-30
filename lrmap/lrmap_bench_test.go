package lrmap_test

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Danglebary/leftright-go/lrmap"
)

const benchMapSize = 1000

// ---------------------------------------------------------------------------
// lrmap benchmarks
// ---------------------------------------------------------------------------

func BenchmarkLRMap(b *testing.B) {
	b.Run("Read", func(b *testing.B) {
		w, f := lrmap.New[int, int]()
		defer w.Close()
		for i := range benchMapSize {
			w.Set(i, i)
		}
		r := f.Handle()
		defer r.Close()

		b.ResetTimer()
		for range b.N {
			r.Get(42)
		}
	})

	b.Run("ReadParallel", func(b *testing.B) {
		w, f := lrmap.New[int, int]()
		defer w.Close()
		for i := range benchMapSize {
			w.Set(i, i)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := f.Handle()
			defer r.Close()
			for pb.Next() {
				r.Get(42)
			}
		})
	})

	b.Run("ReadUnderWrite", func(b *testing.B) {
		w, f := lrmap.New[int, int]()
		for i := range benchMapSize {
			w.Set(i, i)
		}

		var stop atomic.Bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := 0
			for !stop.Load() {
				w.Set(i%benchMapSize, i)
				i++
			}
		}()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := f.Handle()
			defer r.Close()
			for pb.Next() {
				r.Get(42)
			}
		})
		b.StopTimer()

		stop.Store(true)
		wg.Wait()
		w.Close()
	})

	for _, size := range []int{1, 10, 100, 1000} {
		b.Run("Publish/"+strconv.Itoa(size), func(b *testing.B) {
			w, _ := lrmap.New[int, int]()
			defer w.Close()

			b.ResetTimer()
			for range b.N {
				for j := range size {
					w.BufferSet(j, j)
				}
				w.Publish()
			}
		})
	}
}

// ---------------------------------------------------------------------------
// sync.RWMutex baseline
// ---------------------------------------------------------------------------

type rwMutexMap struct {
	mu sync.RWMutex
	m  map[int]int
}

func newRWMutexMap(size int) *rwMutexMap {
	m := make(map[int]int, size)
	for i := range size {
		m[i] = i
	}
	return &rwMutexMap{m: m}
}

func (r *rwMutexMap) Get(key int) (int, bool) {
	r.mu.RLock()
	v, ok := r.m[key]
	r.mu.RUnlock()
	return v, ok
}

func (r *rwMutexMap) Set(key, val int) {
	r.mu.Lock()
	r.m[key] = val
	r.mu.Unlock()
}

func BenchmarkRWMutex(b *testing.B) {
	b.Run("Read", func(b *testing.B) {
		m := newRWMutexMap(benchMapSize)
		b.ResetTimer()
		for range b.N {
			m.Get(42)
		}
	})

	b.Run("ReadParallel", func(b *testing.B) {
		m := newRWMutexMap(benchMapSize)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Get(42)
			}
		})
	})

	b.Run("ReadUnderWrite", func(b *testing.B) {
		m := newRWMutexMap(benchMapSize)

		var stop atomic.Bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := 0
			for !stop.Load() {
				m.Set(i%benchMapSize, i)
				i++
			}
		}()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Get(42)
			}
		})
		b.StopTimer()

		stop.Store(true)
		wg.Wait()
	})
}

// ---------------------------------------------------------------------------
// sync.Map baseline
// ---------------------------------------------------------------------------

func newSyncMap(size int) *sync.Map {
	var m sync.Map
	for i := range size {
		m.Store(i, i)
	}
	return &m
}

func BenchmarkSyncMap(b *testing.B) {
	b.Run("Read", func(b *testing.B) {
		m := newSyncMap(benchMapSize)
		b.ResetTimer()
		for range b.N {
			m.Load(42)
		}
	})

	b.Run("ReadParallel", func(b *testing.B) {
		m := newSyncMap(benchMapSize)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Load(42)
			}
		})
	})

	b.Run("ReadUnderWrite", func(b *testing.B) {
		m := newSyncMap(benchMapSize)

		var stop atomic.Bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := 0
			for !stop.Load() {
				m.Store(i%benchMapSize, i)
				i++
			}
		}()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Load(42)
			}
		})
		b.StopTimer()

		stop.Store(true)
		wg.Wait()
	})
}

// ---------------------------------------------------------------------------
// String-key benchmarks
// ---------------------------------------------------------------------------

func BenchmarkLRMapString(b *testing.B) {
	b.Run("Read", func(b *testing.B) {
		w, f := lrmap.New[string, string]()
		defer w.Close()
		for i := range benchMapSize {
			s := strconv.Itoa(i)
			w.Set(s, s)
		}
		r := f.Handle()
		defer r.Close()

		b.ResetTimer()
		for range b.N {
			r.Get("42")
		}
	})

	b.Run("ReadParallel", func(b *testing.B) {
		w, f := lrmap.New[string, string]()
		defer w.Close()
		for i := range benchMapSize {
			s := strconv.Itoa(i)
			w.Set(s, s)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := f.Handle()
			defer r.Close()
			for pb.Next() {
				r.Get("42")
			}
		})
	})

	b.Run("ReadUnderWrite", func(b *testing.B) {
		w, f := lrmap.New[string, string]()
		for i := range benchMapSize {
			s := strconv.Itoa(i)
			w.Set(s, s)
		}

		var stop atomic.Bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := 0
			for !stop.Load() {
				s := strconv.Itoa(i % benchMapSize)
				w.Set(s, s)
				i++
			}
		}()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			r := f.Handle()
			defer r.Close()
			for pb.Next() {
				r.Get("42")
			}
		})
		b.StopTimer()

		stop.Store(true)
		wg.Wait()
		w.Close()
	})
}

type rwMutexMapString struct {
	mu sync.RWMutex
	m  map[string]string
}

func newRWMutexMapString(size int) *rwMutexMapString {
	m := make(map[string]string, size)
	for i := range size {
		s := strconv.Itoa(i)
		m[s] = s
	}
	return &rwMutexMapString{m: m}
}

func (r *rwMutexMapString) Get(key string) (string, bool) {
	r.mu.RLock()
	v, ok := r.m[key]
	r.mu.RUnlock()
	return v, ok
}

func (r *rwMutexMapString) Set(key, val string) {
	r.mu.Lock()
	r.m[key] = val
	r.mu.Unlock()
}

func BenchmarkRWMutexString(b *testing.B) {
	b.Run("Read", func(b *testing.B) {
		m := newRWMutexMapString(benchMapSize)
		b.ResetTimer()
		for range b.N {
			m.Get("42")
		}
	})

	b.Run("ReadParallel", func(b *testing.B) {
		m := newRWMutexMapString(benchMapSize)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Get("42")
			}
		})
	})

	b.Run("ReadUnderWrite", func(b *testing.B) {
		m := newRWMutexMapString(benchMapSize)

		var stop atomic.Bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := 0
			for !stop.Load() {
				s := strconv.Itoa(i % benchMapSize)
				m.Set(s, s)
				i++
			}
		}()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Get("42")
			}
		})
		b.StopTimer()

		stop.Store(true)
		wg.Wait()
	})
}

func newSyncMapString(size int) *sync.Map {
	var m sync.Map
	for i := range size {
		s := strconv.Itoa(i)
		m.Store(s, s)
	}
	return &m
}

func BenchmarkSyncMapString(b *testing.B) {
	b.Run("Read", func(b *testing.B) {
		m := newSyncMapString(benchMapSize)
		b.ResetTimer()
		for range b.N {
			m.Load("42")
		}
	})

	b.Run("ReadParallel", func(b *testing.B) {
		m := newSyncMapString(benchMapSize)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Load("42")
			}
		})
	})

	b.Run("ReadUnderWrite", func(b *testing.B) {
		m := newSyncMapString(benchMapSize)

		var stop atomic.Bool
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			i := 0
			for !stop.Load() {
				s := strconv.Itoa(i % benchMapSize)
				m.Store(s, s)
				i++
			}
		}()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				m.Load("42")
			}
		})
		b.StopTimer()

		stop.Store(true)
		wg.Wait()
	})
}
