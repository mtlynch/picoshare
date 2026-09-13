package handlers

import "time"

// NowFunc returns the current time.
type NowFunc func() time.Time
