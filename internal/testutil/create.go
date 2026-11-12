// Package testutil contains helpers for creating values in tests.
package testutil

import "testing"

// MustCreate creates a test value, failing the test if creation returns an error.
func MustCreate[RawInput, CreatedValue any](
	t testing.TB,
	create func(RawInput) (CreatedValue, error),
	rawInput RawInput,
) CreatedValue {
	t.Helper()

	value, err := create(rawInput)
	if err != nil {
		t.Fatalf("failed to create test value: %v", err)
	}
	return value
}

// MustCreate2 creates a test value from two inputs, failing the test if creation
// returns an error.
func MustCreate2[RawInput1, RawInput2, CreatedValue any](
	t testing.TB,
	create func(RawInput1, RawInput2) (CreatedValue, error),
	rawInput1 RawInput1,
	rawInput2 RawInput2,
) CreatedValue {
	t.Helper()

	value, err := create(rawInput1, rawInput2)
	if err != nil {
		t.Fatalf("failed to create test value: %v", err)
	}
	return value
}
