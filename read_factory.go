package leftright

import "runtime"

// ReadHandleFactory creates ReadHandle instances. It is safe for
// concurrent use from multiple goroutines.
type ReadHandleFactory[T any] struct {
	inner *inner[T]
}

// Handle creates a new ReadHandle. The caller is responsible for calling
// Close on the returned handle when it is no longer needed. A runtime
// finalizer provides a safety net for leaked handles, but explicit Close
// is strongly preferred.
func (f *ReadHandleFactory[T]) Handle() *ReadHandle[T] {
	slot := &readerSlot{}

	f.inner.readerMu.Lock()
	f.inner.readers = append(f.inner.readers, slot)
	f.inner.readerMu.Unlock()

	rh := &ReadHandle[T]{inner: f.inner, slot: slot}
	runtime.SetFinalizer(rh, (*ReadHandle[T]).Close)
	return rh
}
