# leftright

A Go implementation of the [left-right](https://docs.rs/left-right/latest/left_right/) concurrency primitive: **wait-free reads** over a single-writer data structure by maintaining two copies and swapping an atomic pointer between them.

Ideal for read-heavy workloads where reader throughput matters more than write latency.

## How It Works

Two identical copies of data structure `T` are maintained ("left" and "right"). Readers always access whichever copy the atomic pointer currently references -- no locks, no allocations, just two atomic increments and one atomic load per read.

The writer appends operations to an oplog. When `Publish()` is called:

1. Apply all pending operations to the **inactive** copy
2. Atomically swap the pointer (readers now see the updated copy)
3. Wait for all readers still on the **old** copy to finish
4. Replay the oplog onto the now-stale copy so both copies converge
5. Clear the oplog

Each reader has its own **cache-line-padded epoch counter** to prevent false sharing. The epoch protocol (odd = reading, even = idle) lets the writer determine when all readers have departed the old copy without any locks on the read path.

## Installation

```
go get github.com/Danglebary/leftright-go
```

Requires **Go 1.26+** (generics).

## Usage

```go
package main

import (
    "fmt"
    "github.com/Danglebary/leftright-go"
)

// Define your operation type
type MapOp struct {
    Key   string
    Value string
}

func main() {
    // Initial state
    init := &map[string]string{"hello": "world"}

    // Clone function to create the second copy
    clone := func(src *map[string]string) *map[string]string {
        m := make(map[string]string, len(*src))
        for k, v := range *src {
            m[k] = v
        }
        return &m
    }

    // Absorb function: how to apply an operation
    absorb := func(data *map[string]string, op MapOp) {
        (*data)[op.Key] = op.Value
    }

    // Create the left-right instance
    writeHandle, factory := leftright.New(init, clone, absorb)
    defer writeHandle.Close()

    // Obtain a read handle from the factory
    readHandle := factory.Handle()
    defer readHandle.Close()

    // Write: append operations and publish
    writeHandle.Append(MapOp{Key: "foo", Value: "bar"})
    writeHandle.Publish() // Now visible to readers

    // Read: wait-free, zero-allocation
    readHandle.Read(func(data *map[string]string) {
        fmt.Println((*data)["foo"]) // "bar"
    })
}
```

### Multiple Readers

Each goroutine should have its own `ReadHandle` to avoid contention on the epoch counter. `New` returns a `ReadHandleFactory` for this purpose:

```go
writeHandle, factory := leftright.New(init, clone, absorb)

// Each goroutine gets its own handle from the factory
for i := 0; i < numWorkers; i++ {
    rh := factory.Handle()
    go func() {
        defer rh.Close()
        for {
            rh.Read(func(data *MyType) {
                // ... wait-free read
            })
        }
    }()
}
```

## API

### Construction

```go
func New[T any, O any](
    init   *T,                   // initial state (writer takes ownership)
    clone  func(src *T) *T,      // deep-clone to create the second copy
    absorb AbsorbFunc[T, O],     // how to apply operations
) (*WriteHandle[T, O], *ReadHandleFactory[T])
```

### Types

| Type                   | Description                                                                                                                                                                                      |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `AbsorbFunc[T, O]`     | `func(data *T, op O)` -- applies a single operation to the data structure. Must be **deterministic**: applying the same operation to two identical values of `T` must produce identical results. |
| `ReadHandle[T]`        | Wait-free reader. One per goroutine. Not safe for concurrent use.                                                                                                                                |
| `ReadHandleFactory[T]` | Creates new `ReadHandle` instances (thread-safe).                                                                                                                                                |
| `WriteHandle[T, O]`    | Single writer with oplog and publish semantics.                                                                                                                                                  |

### ReadHandle

| Method                        | Description                                                                                                                                             |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Read(fn func(data *T)) bool` | Execute `fn` with a pointer to the current snapshot. The pointer is valid only for the duration of `fn`. Returns `false` if the writer has been closed. |
| `Close()`                     | Deregisters this reader from the epoch registry.                                                                                                        |

### WriteHandle

| Method         | Description                                                                                             |
| -------------- | ------------------------------------------------------------------------------------------------------- |
| `Append(op O)` | Add an operation to the oplog. Not visible to readers until `Publish()`.                                |
| `Publish()`    | Make all pending operations visible to readers. Blocks until all readers on the old copy have departed. |
| `Close()`      | Flush pending ops and signal readers that no more writes will occur.                                    |

## Performance Characteristics

| Path                         | Locks                                 | Allocations                       | Atomics                |
| ---------------------------- | ------------------------------------- | --------------------------------- | ---------------------- |
| `Read()`                     | None                                  | Zero                              | 2 increments + 1 load  |
| `Append()`                   | None                                  | Amortized zero (slice append)     | None                   |
| `Publish()`                  | Brief mutex (snapshot epoch counters) | None (reuses snapshot buffer)     | Store + epoch polling  |
| `ReadHandleFactory.Handle()` | Brief mutex (register slot)           | 1 (epoch slot, cache-line padded) | None                   |

The writer's `waitForReaders()` uses adaptive backoff: it yields via `runtime.Gosched()` for 64 iterations, then switches to exponential sleep backoff (1us to 100us).

### Benchmarks: `lrmap` vs `sync.RWMutex` vs `sync.Map`

Read throughput with 1,000 entries on Apple M1 (8 cores):

**`map[int]int`**

| Benchmark          | `lrmap`     | `sync.RWMutex` | `sync.Map`  |
| ------------------ | ----------- | -------------- | ----------- |
| Read (single)      | 16.06 ns/op | 14.19 ns/op    | 10.61 ns/op |
| Read (parallel)    | 5.25 ns/op  | 75.91 ns/op    | 2.36 ns/op  |
| Read (under write) | 3.64 ns/op  | 92.62 ns/op    | 2.80 ns/op  |

**`map[string]string`**

| Benchmark          | `lrmap`     | `sync.RWMutex` | `sync.Map`  |
| ------------------ | ----------- | -------------- | ----------- |
| Read (single)      | 19.02 ns/op | 14.38 ns/op    | 14.39 ns/op |
| Read (parallel)    | 5.23 ns/op  | 89.03 ns/op    | 3.17 ns/op  |
| Read (under write) | 4.33 ns/op  | 89.63 ns/op    | 3.91 ns/op  |

All benchmarks achieve **zero allocations** on the read path.

**Takeaway:** Under contention (parallel reads, or reads concurrent with writes), `lrmap` is **~14-25x faster** than `sync.RWMutex`. `sync.Map` is faster for simple key types due to its internal optimizations, but does not generalize to arbitrary data structures the way leftright does. With string keys, the single-threaded gap between `lrmap` and `sync.Map` narrows significantly (19 vs 14 ns).

Publish latency (single writer, no readers):

| Oplog size | ns/op     |
| ---------- | --------- |
| 1          | 23 ns     |
| 10         | 171 ns    |
| 100        | 1,688 ns  |
| 1,000      | 16,955 ns |

Run benchmarks yourself with `just bench` or save a comparison baseline with `just bench-save`.

## Performance Tips

### Batch reads to amortize atomic overhead

Each `Read()` call costs 3 atomic operations (2 increments + 1 load) regardless of how much work the callback does. If you need multiple lookups, batch them into a single `Read()` call:

```go
// 3 separate Read() calls = 9 atomic ops
readHandle.Read(func(data *map[string]string) { v1 = (*data)["a"] })
readHandle.Read(func(data *map[string]string) { v2 = (*data)["b"] })
readHandle.Read(func(data *map[string]string) { v3 = (*data)["c"] })

// 1 batched Read() call = 3 atomic ops total
readHandle.Read(func(data *map[string]string) {
    v1 = (*data)["a"]
    v2 = (*data)["b"]
    v3 = (*data)["c"]
})
```

Note: `lrmap.Reader` methods like `Get()` and `Contains()` each perform one `Read()` call internally. For bulk lookups, use `ForEach` or drop to the core `leftright` API directly.

## Design Decisions

- **Closure-based reads** instead of guard objects -- Go has no borrow checker, so a closure scopes the read lifetime safely.
- **Cache-line padding** on epoch counters (64 bytes on amd64, 128 bytes on arm64) eliminates false sharing between reader goroutines.
- **Function-based absorb** instead of an interface to avoid self-referential type constraints in Go generics.
- **Clone at construction** instead of `Default` -- avoids the Rust footgun where two `Default` instances may differ (e.g., `HashMap` with random hasher seeds).
- **Sequential consistency** via Go's `sync/atomic` -- no attempt to use relaxed ordering like the Rust version

## Caveats

- **No writer-side reads.** `WriteHandle` has no `Read` method. The writer cannot inspect the current state of the data structure. Patterns like "set if absent" or "read-modify-write" cannot be expressed through the leftright API alone; maintain a secondary index or shadow state externally if needed.

- **Shallow-copy semantics.** If `T` is or contains pointer types, the `clone` function and `AbsorbFunc` must deep-copy any referenced data. Otherwise both copies share the same underlying pointer, and mutations through one copy will corrupt the other, bypassing the leftright protocol entirely. This applies equally to the `lrmap` package: if `V` is a pointer or contains pointers, mutations through the pointer are data races.

- **Reader handle lifecycle.** Each `ReadHandle` must be closed when no longer needed via `Close()`. Leaked handles accumulate slots in the reader registry, increasing `Publish()` latency and consuming memory. A runtime finalizer provides a safety net for leaked handles, but explicit `Close()` (typically via `defer`) is strongly preferred. `Read()` returns `false` after a handle is closed.
