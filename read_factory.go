package leftright

type ReadHandleFactory[T any] struct {
	inner *inner[T]
}

func (f *ReadHandleFactory[T]) Handle() *ReadHandle[T] {
	slot := &readerSlot{}

	f.inner.readerMu.Lock()
	f.inner.readers = append(f.inner.readers, slot)
	f.inner.readerMu.Unlock()

	return &ReadHandle[T]{inner: f.inner, slot: slot}
}
