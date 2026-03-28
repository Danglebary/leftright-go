package leftright

// AbsorbFunc applies a single operation to the data structure.
// Called once per copy per publish cycle, so it must be deterministic -
// applying the same operation to two identical T values must produce
// identical results.
type AbsorbFunc[T any, O any] func(data *T, op O)
