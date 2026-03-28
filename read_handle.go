package leftright

// ReadHandle provides wait-free read access to the data structure.
// Each goroutine that reads should have its own ReadHandle.
// ReadHandle is NOT safe for concurrent use by multiple goroutines,
// as sharing one would cause contention on the epoch counter,
// defeating the purpose of this design.
type ReadHandle[T any] struct {
	inner *inner[T]
	slot  *readerSlot
}

// Read executes `fn` with a pointer to the current reader copy.
// The pointer is only valid for the duration of `fn`, do not retain it.
// Returns false if the WriteHandle has been dropped.
func (r *ReadHandle[T]) Read(fn func(data *T)) bool {
	if r.inner.closed.Load() {
		return false
	}

	// Enter: bump epoch to odd
	r.slot.epoch.Add(1)
	// Read which copy is active
	idx := r.inner.which.Load()
	// Execute user function
	fn(r.inner.data[idx])
	// Leave: bump epoch to even
	r.slot.epoch.Add(1)
	return true
}

func (r *ReadHandle[T]) Close() {
	r.inner.readerMu.Lock()
	// find and remove the slot from the readers slice
	for i, slot := range r.inner.readers {
		if slot == r.slot {
			// remove the slot from the slice
			r.inner.readers = append(r.inner.readers[:i], r.inner.readers[i+1:]...)
			break
		}
	}

	// Clear the slot's epoch to indicate it's no longer active
	r.slot = nil
	// Unlock the readerMu after modifying the readers slice
	r.inner.readerMu.Unlock()
	// Clear the inner reference to allow garbage collection
	r.inner = nil
}
