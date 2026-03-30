package leftright

import (
	"runtime"
	"time"
)

const (
	// Number of Gosched() yields before switching to sleep backoff.
	yieldSpinLimit = 64
	// Initial sleep duration once yield spinning is exhausted.
	minBackoff = 1 * time.Microsecond
	// Maximum sleep duration during backoff.
	maxBackoff = 100 * time.Microsecond
)

type WriteHandle[T any, O any] struct {
	inner  *inner[T]
	oplog  []O
	absorb AbsorbFunc[T, O]
	// writerIdx is the index of the copy the writer can mutate.
	// (always the opposite of inner.which)
	writerIdx int
	// snapshots and slotsCopy are reused across Publish calls to avoid
	// per-publish allocation. Grown only when the reader count exceeds
	// current capacity.
	snapshots []uint64
	slotsCopy []*readerSlot
}

// Append adds an operation to the oplog.
// The operation is NOT visible to readers until Publish() is called.
func (w *WriteHandle[T, O]) Append(op O) {
	w.oplog = append(w.oplog, op)
}

// Publish makes all pending operations visible to readers.
//
// 1. Apply oplog to the inactive copy
// 2. Swap the active copy pointer
// 3. Wait for all readers on the old copy to finish
// 4. Replay oplog onto the now-stale copy
// 5. Clear oplog
func (w *WriteHandle[T, O]) Publish() {
	if len(w.oplog) == 0 {
		return
	}

	// Step 1: Apply ops to writer's copy (inactive)
	writerCopy := w.inner.data[w.writerIdx]
	for _, op := range w.oplog {
		w.absorb(writerCopy, op)
	}

	// Step 2: Swap - readers now see the updated copy
	w.inner.which.Store(uint32(w.writerIdx))

	// Step 3: Wait for all readers to depart from the old copy
	w.waitForReaders()

	// Step 4: Replay oplog onto the now-stale copy
	w.writerIdx = 1 - w.writerIdx // switch to the other copy
	staleCopy := w.inner.data[w.writerIdx]
	for _, op := range w.oplog {
		w.absorb(staleCopy, op)
	}

	// Step 5: Clear oplog
	w.oplog = w.oplog[:0]
}

func (w *WriteHandle[T, O]) waitForReaders() {
	w.inner.readerMu.Lock()

	n := len(w.inner.readers)
	if cap(w.snapshots) < n {
		w.snapshots = make([]uint64, n)
	} else {
		w.snapshots = w.snapshots[:n]
	}

	// Copy both the epoch snapshots and the slot pointers into
	// writer-owned buffers. A bare slice header copy would share
	// the backing array with inner.readers, racing with
	// ReadHandle.Close which shifts elements via append.
	if cap(w.slotsCopy) < n {
		w.slotsCopy = make([]*readerSlot, n)
	} else {
		w.slotsCopy = w.slotsCopy[:n]
	}
	copy(w.slotsCopy, w.inner.readers)

	for i, slot := range w.slotsCopy {
		w.snapshots[i] = slot.epoch.Load()
	}
	w.inner.readerMu.Unlock()

	// Spin until every reader has advanced past their snapshot.
	// A reader has departed if:
	//   - its epoch was even at snapshot time (was idle - already safe)
	//   - its epoch has changed from the snapshot (re-entered on new copy)
	spins := 0
	sleep := minBackoff
	for {
		allClear := true
		for i, slot := range w.slotsCopy {
			snap := w.snapshots[i]
			if snap%2 == 0 {
				// Was idle at snapshot time - already safe
				continue
			}
			current := slot.epoch.Load()
			if current == snap {
				// Still in the same read, not departed yet
				allClear = false
				break
			}
			// Epoch advanced, reader has departed and possibly re-entered on the new copy - safe
		}
		if allClear {
			return
		}

		spins++
		if spins <= yieldSpinLimit {
			runtime.Gosched()
		} else {
			time.Sleep(sleep)
			sleep *= 2
			if sleep > maxBackoff {
				sleep = maxBackoff
			}
		}
	}
}

// Close signals that no more write will occur.
// After Close() is called, ReadHandle.Read() will return false.
func (w *WriteHandle[T, O]) Close() {
	// Flush any pending ops before closing
	w.Publish()
	w.inner.which.Store(closedSentinel)
}
