package lrmap

import "github.com/Danglebary/leftright-go"

// Writer provides write access to the concurrent map.
// A Writer must not be used concurrently from multiple goroutines.
type Writer[K comparable, V any] struct {
	wh *leftright.WriteHandle[map[K]V, mapOp[K, V]]
}

// Set inserts or updates a key-value pair and immediately publishes
// the change to readers. For bulk mutations, use BufferSet and Publish.
func (w *Writer[K, V]) Set(key K, value V) {
	w.wh.Append(mapOp[K, V]{kind: opSet, key: key, value: value})
	w.wh.Publish()
}

// Delete removes a key and immediately publishes the change to readers.
// If the key does not exist, this is a no-op that still incurs a publish.
// For bulk mutations, use BufferDelete and Publish.
func (w *Writer[K, V]) Delete(key K) {
	w.wh.Append(mapOp[K, V]{kind: opDelete, key: key})
	w.wh.Publish()
}

// BufferSet stages a set operation without publishing.
// Call Publish to make buffered operations visible to readers.
func (w *Writer[K, V]) BufferSet(key K, value V) {
	w.wh.Append(mapOp[K, V]{kind: opSet, key: key, value: value})
}

// BufferDelete stages a delete operation without publishing.
// Call Publish to make buffered operations visible to readers.
func (w *Writer[K, V]) BufferDelete(key K) {
	w.wh.Append(mapOp[K, V]{kind: opDelete, key: key})
}

// Publish makes all buffered operations visible to readers.
// If there are no buffered operations, Publish is a no-op.
func (w *Writer[K, V]) Publish() {
	w.wh.Publish()
}

// Close flushes any pending buffered operations and signals readers
// that no more writes will occur. After Close, Reader.Get will return
// the zero value of V and false for all keys.
func (w *Writer[K, V]) Close() {
	w.wh.Close()
}
