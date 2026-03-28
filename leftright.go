package leftright

func New[T any, O any](
	init *T, // the initial state (writer takes ownership)
	clone func(src *T) *T, // deep-clone to create the second copy
	absorb AbsorbFunc[T, O], // how to apply operations
) (*WriteHandle[T, O], *ReadHandleFactory[T]) {
	second := clone(init)

	shared := &inner[T]{
		data: [2]*T{init, second},
	}

	wh := &WriteHandle[T, O]{
		inner:     shared,
		absorb:    absorb,
		writerIdx: 1, // readers start on 0, writer owns 1
	}

	rf := &ReadHandleFactory[T]{
		inner: shared,
	}

	return wh, rf
}
