package leftright

import "sync/atomic"

// readerSlot holds a single reader's epoch counter,
// padded to a full cache line to prevent false sharing.
type readerSlot struct {
	epoch atomic.Uint64
	_     [cacheLineBytes - 8]byte // pad to cacheLineBytes
}
