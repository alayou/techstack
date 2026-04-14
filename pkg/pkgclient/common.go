package pkgclient

import (
	"iter"
)

func First[T any](seq iter.Seq[T]) (T, bool) {
	for v := range seq {
		return v, true
	}
	var zero T
	return zero, false
}
