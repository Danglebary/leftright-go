package lrmap

import "github.com/Danglebary/leftright-go"

// Reader provides wait-free read access to the concurrent map.
// Each goroutine should have its own Reader instance; a Reader
// must not be used concurrently from multiple goroutines.
type Reader[K comparable, V any] struct {
	rh *leftright.ReadHandle[map[K]V]
}

// Get returns the value associated with key and true, or the zero
// value of V and false if the key is not present or the writer
// has been closed.
func (r *Reader[K, V]) Get(key K) (V, bool) {
	var val V
	var found bool
	r.rh.Read(func(data *map[K]V) {
		val, found = (*data)[key]
	})
	return val, found
}

// Len returns the number of entries in the map, or 0 if the writer
// has been closed.
func (r *Reader[K, V]) Len() int {
	var n int
	r.rh.Read(func(data *map[K]V) {
		n = len(*data)
	})
	return n
}

// Contains reports whether the map contains the given key.
// Returns false if the writer has been closed.
func (r *Reader[K, V]) Contains(key K) bool {
	var found bool
	r.rh.Read(func(data *map[K]V) {
		_, found = (*data)[key]
	})
	return found
}

// ForEach calls fn for each key-value pair in the map.
// If fn returns false, iteration stops early.
// ForEach is a no-op if the writer has been closed.
//
// Note: long-running callbacks will block Publish, because the
// reader's epoch remains active for the duration of the iteration.
// Keep callbacks fast.
func (r *Reader[K, V]) ForEach(fn func(key K, value V) bool) {
	r.rh.Read(func(data *map[K]V) {
		for k, v := range *data {
			if !fn(k, v) {
				return
			}
		}
	})
}

// Close deregisters this reader. After Close, the Reader must not be used.
func (r *Reader[K, V]) Close() {
	r.rh.Close()
}
