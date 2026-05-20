package provider

import "fmt"

type namedLookupCandidate[T any] struct {
	name  string
	value T
}

func selectUniqueByName[T any](kind string, name string, scope string, candidates []namedLookupCandidate[T]) (T, error) {
	var zero T
	var matches []T

	for _, candidate := range candidates {
		if candidate.name == name {
			matches = append(matches, candidate.value)
		}
	}

	switch len(matches) {
	case 0:
		return zero, fmt.Errorf("no %s named %q found in %s", kind, name, scope)
	case 1:
		return matches[0], nil
	default:
		return zero, fmt.Errorf("found %d %s entries named %q in %s; expected exactly one", len(matches), kind, name, scope)
	}
}
