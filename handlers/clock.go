package handlers

import "time"

type defaultClock struct{}

func (c defaultClock) Now() time.Time {
	return time.Now()
}

func NewClock() defaultClock {
	return defaultClock{}
}
