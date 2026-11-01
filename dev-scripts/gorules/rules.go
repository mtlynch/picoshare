//go:build ruleguard

// Use ruleguard, not ignore, so go mod tidy and vendoring retain the DSL
// dependency while normal builds exclude this file.
package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

func forbidRealTimeInTests(m dsl.Matcher) {
	m.Match(`time.Now()`).
		Where(m.File().Name.Matches(`_test\.go$`)).
		Report("use a fixed time instead of time.Now() in tests")
}
