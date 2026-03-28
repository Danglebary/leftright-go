package leftright

import (
	"sync"
	"sync/atomic"
)

// inner holdes the shared state between readers and the writer.
type inner[T any] struct {
	// data holds pointers to the two copies.
	data [2]*T

	// which stores the index (0 or 1) of the current reader copy.
	// Readers atomically load this to know which copy to read.
	which atomic.Uint32

	// readers is the registry of all active reader epoch counters.
	// Protected by readerMu for registration/deregistration only.
	// The epoch counters themselves are accessed lock-free.
	readerMu sync.Mutex
	readers  []*readerSlot

	// closed is set to true when the WriteHandle is dropped.
	// Causes ReadHandle.Read() to return false.
	closed atomic.Bool
}
