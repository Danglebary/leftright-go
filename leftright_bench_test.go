package leftright

import "testing"

// BenchmarkRead measures the cost of the full Read() path (with defer)
// versus a manual epoch bump (without defer) to isolate the defer overhead.
func BenchmarkRead(b *testing.B) {
	init := make(map[string]int)
	for i := range 1000 {
		init[string(rune('A'+i%26))] = i
	}

	clone := func(src *map[string]int) *map[string]int {
		m := make(map[string]int, len(*src))
		for k, v := range *src {
			m[k] = v
		}
		return &m
	}

	type op struct {
		key string
		val int
	}
	absorb := func(data *map[string]int, o op) {
		(*data)[o.key] = o.val
	}

	w, rf := New(&init, clone, absorb)
	defer w.Close()
	rh := rf.Handle()
	defer rh.Close()

	b.Run("WithDefer", func(b *testing.B) {
		for range b.N {
			rh.Read(func(data *map[string]int) {
				_ = (*data)["A"]
			})
		}
	})

	b.Run("ManualEpoch", func(b *testing.B) {
		// Same atomic ops as Read() but without defer.
		// This is test-only code to measure the defer overhead.
		slot := rh.slot
		inner := rh.inner
		for range b.N {
			slot.epoch.Add(1)
			idx := inner.which.Load()
			if idx == closedSentinel {
				slot.epoch.Add(1)
				continue
			}
			_ = (*inner.data[idx])["A"]
			slot.epoch.Add(1)
		}
	})
}
