package leftright

import "runtime"

// ReadHandle provides wait-free read access to the data structure.
// Each goroutine that reads should have its own ReadHandle.
// ReadHandle is NOT safe for concurrent use by multiple goroutines,
// as sharing one would cause contention on the epoch counter,
// defeating the purpose of this design.
type ReadHandle[T any] struct {
	inner  *inner[T]
	slot   *readerSlot
	closed bool
}

// Read executes `fn` with a pointer to the current reader copy.
// The pointer is only valid for the duration of `fn`, do not retain it.
// Returns false if the ReadHandle has been closed or the WriteHandle
// has been dropped.
func (r *ReadHandle[T]) Read(fn func(data *T)) bool {
	if r.closed || r.inner.closed.Load() {
		return false
	}

	// Enter: bump epoch to odd
	r.slot.epoch.Add(1)
	// Leave: bump epoch to even (deferred to ensure it runs even if fn panics)
	defer r.slot.epoch.Add(1)
	// Read which copy is active
	idx := r.inner.which.Load()
	// Execute user function
	fn(r.inner.data[idx])
	return true
}

// Close deregisters this reader from the epoch registry.
// After Close, Read will return false. Close is idempotent.
func (r *ReadHandle[T]) Close() {
	if r.closed {
		return
	}
	r.closed = true
	runtime.SetFinalizer(r, nil)

	r.inner.readerMu.Lock()
	for i, slot := range r.inner.readers {
		if slot == r.slot {
			r.inner.readers = append(r.inner.readers[:i], r.inner.readers[i+1:]...)
			break
		}
	}
	r.inner.readerMu.Unlock()
}
