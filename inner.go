package leftright

import (
	"sync"
	"sync/atomic"
)

// closedSentinel is stored in which to signal that the writer has closed.
// Readers check for this value after loading which, eliminating a separate
// atomic.Bool load on the read path.
const closedSentinel = ^uint32(0)

// inner holds the shared state between readers and the writer.
type inner[T any] struct {
	// data holds pointers to the two copies.
	data [2]*T

	// which stores the index (0 or 1) of the current reader copy.
	// The sentinel value closedSentinel signals that the writer has closed.
	// Readers atomically load this to know which copy to read.
	which atomic.Uint32

	// readers is the registry of all active reader epoch counters.
	// Protected by readerMu for registration/deregistration only.
	// The epoch counters themselves are accessed lock-free.
	readerMu sync.Mutex
	readers  []*readerSlot
}
