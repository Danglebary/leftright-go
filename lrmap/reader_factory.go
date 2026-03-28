package lrmap

import "github.com/halfblown/leftright"

// ReaderFactory creates Reader instances. It is safe for concurrent
// use from multiple goroutines. Each goroutine that needs to read
// the map should call Handle to obtain its own Reader.
type ReaderFactory[K comparable, V any] struct {
	rf *leftright.ReadHandleFactory[map[K]V]
}

// Handle creates a new Reader. The caller is responsible for calling
// Close on the returned Reader when it is no longer needed.
func (f *ReaderFactory[K, V]) Handle() *Reader[K, V] {
	return &Reader[K, V]{rh: f.rf.Handle()}
}
