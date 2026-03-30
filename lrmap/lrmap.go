// Package lrmap provides a concurrent map with wait-free reads,
// built on the leftright concurrency primitive.
//
// lrmap maintains two copies of a Go map[K]V and uses an atomic
// pointer swap to provide readers with zero-lock, zero-allocation
// access. A single writer buffers operations and publishes them
// to readers either individually or in batches.
//
// Writer methods are NOT safe for concurrent use. Reader methods
// are NOT safe for concurrent use across goroutines; each goroutine
// should obtain its own Reader from the ReaderFactory.
//
// Values are shallow-copied. If V is or contains a pointer type,
// mutations through the pointer bypass the leftright protocol and
// will cause data races. All mutations must go through the Writer.
package lrmap

import "github.com/Danglebary/leftright-go"

type opKind uint8

const (
	opSet opKind = iota
	opDelete
)

type mapOp[K comparable, V any] struct {
	kind  opKind
	key   K
	value V
}

// New creates a new concurrent map and returns a Writer and a ReaderFactory.
//
// The Writer is the sole owner of write operations and must not be shared
// across goroutines. The ReaderFactory is safe for concurrent use and
// creates per-goroutine Reader handles.
func New[K comparable, V any]() (*Writer[K, V], *ReaderFactory[K, V]) {
	init := make(map[K]V)

	clone := func(src *map[K]V) *map[K]V {
		m := make(map[K]V, len(*src))
		for k, v := range *src {
			m[k] = v
		}
		return &m
	}

	absorb := func(data *map[K]V, op mapOp[K, V]) {
		switch op.kind {
		case opSet:
			(*data)[op.key] = op.value
		case opDelete:
			delete(*data, op.key)
		}
	}

	wh, rf := leftright.New(&init, clone, absorb)

	return &Writer[K, V]{wh: wh}, &ReaderFactory[K, V]{rf: rf}
}
