package clock_test

import (
	"testing"
	clock "time"
)

func TestAliasedCurrentTime(t *testing.T) {
	_ = clock.Now()
}
