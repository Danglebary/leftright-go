package leftright_test

import (
	"runtime"
	"sync"
	"testing"
	"time"

	leftright "github.com/Danglebary/leftright-go"
)

type testOp struct {
	key string
	val int
	del bool
}

func newTestLR() (*leftright.WriteHandle[map[string]int, testOp], *leftright.ReadHandleFactory[map[string]int]) {
	init := make(map[string]int)

	clone := func(src *map[string]int) *map[string]int {
		m := make(map[string]int, len(*src))
		for k, v := range *src {
			m[k] = v
		}
		return &m
	}

	absorb := func(data *map[string]int, op testOp) {
		if op.del {
			delete(*data, op.key)
		} else {
			(*data)[op.key] = op.val
		}
	}

	return leftright.New(&init, clone, absorb)
}

func TestNewReturnsNonNil(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()
	if w == nil {
		t.Fatal("WriteHandle is nil")
	}
	if f == nil {
		t.Fatal("ReadHandleFactory is nil")
	}
}

func TestAppendPublishRead(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Append(testOp{key: "a", val: 1})
	w.Publish()

	var got int
	var found bool
	r.Read(func(data *map[string]int) {
		got, found = (*data)["a"]
	})
	if !found || got != 1 {
		t.Fatalf("expected a=1, got %d (found=%v)", got, found)
	}
}

func TestPublishEmptyOplog(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	// Should not panic or change state.
	w.Publish()

	var n int
	r.Read(func(data *map[string]int) {
		n = len(*data)
	})
	if n != 0 {
		t.Fatalf("expected empty map, got %d entries", n)
	}
}

func TestPublishZeroReaders(t *testing.T) {
	w, _ := newTestLR()
	defer w.Close()

	w.Append(testOp{key: "a", val: 1})
	// Should not panic or block with no readers registered.
	w.Publish()
}

func TestMultipleReadersSeeState(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()
	r1 := f.Handle()
	defer r1.Close()
	r2 := f.Handle()
	defer r2.Close()

	w.Append(testOp{key: "x", val: 42})
	w.Publish()

	for i, r := range []*leftright.ReadHandle[map[string]int]{r1, r2} {
		var got int
		r.Read(func(data *map[string]int) {
			got = (*data)["x"]
		})
		if got != 42 {
			t.Fatalf("reader %d: expected 42, got %d", i, got)
		}
	}
}

func TestReaderCreatedAfterWrites(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()

	w.Append(testOp{key: "a", val: 1})
	w.Append(testOp{key: "b", val: 2})
	w.Publish()

	// Reader created after publish should see all prior writes.
	r := f.Handle()
	defer r.Close()

	var n int
	r.Read(func(data *map[string]int) {
		n = len(*data)
	})
	if n != 2 {
		t.Fatalf("expected 2 entries, got %d", n)
	}
}

func TestWriterCloseSignalsReaders(t *testing.T) {
	w, f := newTestLR()
	r := f.Handle()
	defer r.Close()

	w.Append(testOp{key: "a", val: 1})
	w.Publish()

	w.Close()

	ok := r.Read(func(data *map[string]int) {
		t.Fatal("callback should not be invoked after writer close")
	})
	if ok {
		t.Fatal("expected Read to return false after writer close")
	}
}

func TestReaderCloseDuringWriterOps(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()

	r := f.Handle()
	w.Append(testOp{key: "a", val: 1})
	w.Publish()

	r.Close()

	// Writer should continue to function after a reader closes.
	w.Append(testOp{key: "b", val: 2})
	w.Publish()

	r2 := f.Handle()
	defer r2.Close()

	var got int
	r2.Read(func(data *map[string]int) {
		got = (*data)["b"]
	})
	if got != 2 {
		t.Fatalf("expected b=2, got %d", got)
	}
}

func TestReadAfterReadHandleClose(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()
	r := f.Handle()

	r.Close()

	ok := r.Read(func(data *map[string]int) {
		t.Fatal("callback should not be invoked after ReadHandle close")
	})
	if ok {
		t.Fatal("expected Read to return false after ReadHandle close")
	}
}

func TestReadHandleDoubleClose(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()
	r := f.Handle()

	r.Close()
	r.Close() // should not panic
}

func TestConcurrentReadsUnderWrite(t *testing.T) {
	w, f := newTestLR()

	// Pre-populate
	for i := range 100 {
		w.Append(testOp{key: string(rune('A' + i%26)), val: i})
	}
	w.Publish()

	var wg sync.WaitGroup

	// Readers
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := f.Handle()
			defer r.Close()
			for range 10_000 {
				r.Read(func(data *map[string]int) {
					_ = len(*data)
				})
			}
		}()
	}

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer w.Close()
		for i := range 1_000 {
			w.Append(testOp{key: "w", val: i})
			w.Publish()
		}
	}()

	wg.Wait()
}

func TestPanicInReadCallback(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()
	r := f.Handle()
	defer r.Close()

	w.Append(testOp{key: "a", val: 1})
	w.Publish()

	// Trigger a panic inside Read.
	func() {
		defer func() { recover() }()
		r.Read(func(data *map[string]int) {
			panic("boom")
		})
	}()

	// The critical check: writer must be able to publish without deadlocking.
	done := make(chan struct{})
	go func() {
		w.Append(testOp{key: "b", val: 2})
		w.Publish()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Publish deadlocked after panic in Read callback")
	}

	// Verify the reader still works after the panic.
	var got int
	r.Read(func(data *map[string]int) {
		got = (*data)["b"]
	})
	if got != 2 {
		t.Fatalf("expected b=2 after recovery, got %d", got)
	}
}

func TestFinalizerCleansUpSlot(t *testing.T) {
	w, f := newTestLR()
	defer w.Close()

	// Create a reader and immediately drop the reference.
	f.Handle() //nolint:staticcheck // intentionally not closing

	// Encourage the GC to collect the unreachable ReadHandle.
	runtime.GC()
	runtime.GC()

	// Publish should complete without issues — the finalizer should
	// have deregistered the leaked slot. This is best-effort since
	// finalizer timing is nondeterministic.
	done := make(chan struct{})
	go func() {
		w.Append(testOp{key: "a", val: 1})
		w.Publish()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked — finalizer may not have cleaned up leaked slot")
	}
}
