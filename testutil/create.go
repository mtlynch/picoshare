package testutil

import "testing"

// MustCreate constructs a test value or fails the test if construction fails.
func MustCreate[RawInput, CreatedValue any](
	t testing.TB,
	create func(RawInput) (CreatedValue, error),
	rawInput RawInput,
) CreatedValue {
	t.Helper()
	createdValue, err := create(rawInput)
	if err != nil {
		t.Fatalf("failed to create test value: %v", err)
	}
	return createdValue
}
