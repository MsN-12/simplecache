package simplecache

// Identity returns v unchanged.
//
// Use this only for immutable values or values that are safe to share by copy,
// such as numbers, booleans, strings, and structs containing only immutable
// fields.
func Identity[V any](v V) V {
	return v
}

// CloneSlice returns a shallow copy of s.
//
// If the slice elements are mutable, such as pointers, maps, slices, or structs
// containing mutable fields, provide a custom CloneFunc instead.
func CloneSlice[E any](s []E) []E {
	if s == nil {
		return nil
	}

	out := make([]E, len(s))
	copy(out, s)
	return out
}

// CloneBytes returns a copy of b.
func CloneBytes(b []byte) []byte {
	return CloneSlice(b)
}

// CloneMap returns a shallow copy of m.
//
// If the map values are mutable, such as pointers, maps, slices, or structs
// containing mutable fields, provide a custom CloneFunc instead.
func CloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}

	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}

	return out
}
