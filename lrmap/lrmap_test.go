package lrmap_test

import (
	"sync"
	"testing"

	"github.com/Danglebary/leftright-go/lrmap"
)

func TestNewReturnsNonNil(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	if w == nil {
		t.Fatal("Writer is nil")
	}
	if f == nil {
		t.Fatal("ReaderFactory is nil")
	}
}

func TestSetAndGet(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Set("a", 1)

	val, ok := r.Get("a")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != 1 {
		t.Fatalf("expected 1, got %d", val)
	}
}

func TestGetMissing(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	val, ok := r.Get("missing")
	if ok {
		t.Fatal("expected key to not exist")
	}
	if val != 0 {
		t.Fatalf("expected zero value, got %d", val)
	}
}

func TestOverwrite(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Set("a", 1)
	w.Set("a", 2)

	val, ok := r.Get("a")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != 2 {
		t.Fatalf("expected 2, got %d", val)
	}
}

func TestDelete(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Set("a", 1)
	w.Delete("a")

	_, ok := r.Get("a")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestDeleteMissing(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	// Should not panic.
	w.Delete("nonexistent")
}

func TestLen(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	if r.Len() != 0 {
		t.Fatalf("expected 0, got %d", r.Len())
	}

	w.Set("a", 1)
	w.Set("b", 2)
	w.Set("c", 3)

	if r.Len() != 3 {
		t.Fatalf("expected 3, got %d", r.Len())
	}

	w.Delete("b")

	if r.Len() != 2 {
		t.Fatalf("expected 2, got %d", r.Len())
	}
}

func TestContains(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	if r.Contains("a") {
		t.Fatal("expected false for missing key")
	}

	w.Set("a", 1)

	if !r.Contains("a") {
		t.Fatal("expected true for present key")
	}
}

func TestForEach(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Set("a", 1)
	w.Set("b", 2)
	w.Set("c", 3)

	seen := make(map[string]int)
	r.ForEach(func(k string, v int) bool {
		seen[k] = v
		return true
	})

	if len(seen) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(seen))
	}
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := seen[k]; !ok {
			t.Fatalf("missing key %q", k)
		}
	}
}

func TestForEachEarlyStop(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Set("a", 1)
	w.Set("b", 2)
	w.Set("c", 3)

	count := 0
	r.ForEach(func(_ string, _ int) bool {
		count++
		return false // stop after first
	})

	if count != 1 {
		t.Fatalf("expected 1 iteration, got %d", count)
	}
}

func TestBufferSetPublish(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.BufferSet("a", 1)
	w.BufferSet("b", 2)

	// Not yet visible.
	if r.Len() != 0 {
		t.Fatal("expected buffered ops to not be visible before Publish")
	}

	w.Publish()

	if r.Len() != 2 {
		t.Fatalf("expected 2 after Publish, got %d", r.Len())
	}
	val, ok := r.Get("a")
	if !ok || val != 1 {
		t.Fatalf("expected a=1, got %d, %v", val, ok)
	}
}

func TestBufferDeletePublish(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Set("a", 1)
	w.BufferDelete("a")

	// Still visible before publish.
	if !r.Contains("a") {
		t.Fatal("expected key to still be visible before Publish")
	}

	w.Publish()

	if r.Contains("a") {
		t.Fatal("expected key to be deleted after Publish")
	}
}

func TestMixedBatch(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.BufferSet("a", 1)
	w.BufferSet("b", 2)
	w.BufferSet("c", 3)
	w.BufferDelete("b")
	w.Publish()

	if r.Len() != 2 {
		t.Fatalf("expected 2, got %d", r.Len())
	}
	if r.Contains("b") {
		t.Fatal("expected b to be deleted")
	}
}

func TestPublishNoOps(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	// Should not panic or change state.
	w.Publish()

	if r.Len() != 0 {
		t.Fatal("expected empty map")
	}
}

func TestWriterClose(t *testing.T) {
	w, f := lrmap.New[string, int]()
	r := f.Handle()
	defer r.Close()

	w.Set("a", 1)
	w.Close()

	val, ok := r.Get("a")
	if ok {
		t.Fatalf("expected false after writer close, got val=%d", val)
	}
}

func TestReaderClose(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()
	r := f.Handle()

	w.Set("a", 1)
	r.Close()

	// Writer should still work after reader closes.
	w.Set("b", 2)

	r2 := f.Handle()
	defer r2.Close()

	val, ok := r2.Get("b")
	if !ok || val != 2 {
		t.Fatalf("expected b=2, got %d, %v", val, ok)
	}
}

func TestMultipleReaders(t *testing.T) {
	w, f := lrmap.New[string, int]()
	defer w.Close()

	r1 := f.Handle()
	defer r1.Close()
	r2 := f.Handle()
	defer r2.Close()

	w.Set("a", 42)

	v1, ok1 := r1.Get("a")
	v2, ok2 := r2.Get("a")
	if !ok1 || !ok2 {
		t.Fatal("expected both readers to find the key")
	}
	if v1 != 42 || v2 != 42 {
		t.Fatalf("expected 42 from both, got %d and %d", v1, v2)
	}
}

func TestConcurrentReads(t *testing.T) {
	w, f := lrmap.New[int, int]()
	defer w.Close()

	for i := range 100 {
		w.Set(i, i*10)
	}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := f.Handle()
			defer r.Close()
			for i := range 100 {
				val, ok := r.Get(i)
				if !ok {
					t.Errorf("expected key %d to exist", i)
					return
				}
				if val != i*10 {
					t.Errorf("expected %d, got %d", i*10, val)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentReadWrite(t *testing.T) {
	w, f := lrmap.New[int, int]()

	var wg sync.WaitGroup

	// Readers
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := f.Handle()
			defer r.Close()
			for range 1000 {
				r.Get(0)
				r.Len()
				r.Contains(0)
			}
		}()
	}

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer w.Close()
		for i := range 1000 {
			w.Set(i%100, i)
		}
	}()

	wg.Wait()
}

func TestLargeMap(t *testing.T) {
	w, f := lrmap.New[int, int]()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	const n = 10_000
	for i := range n {
		w.BufferSet(i, i)
	}
	w.Publish()

	if r.Len() != n {
		t.Fatalf("expected %d, got %d", n, r.Len())
	}

	for i := range n {
		val, ok := r.Get(i)
		if !ok || val != i {
			t.Fatalf("key %d: expected %d, got %d (ok=%v)", i, i, val, ok)
		}
	}
}
