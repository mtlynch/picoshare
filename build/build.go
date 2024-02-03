package build

import (
	"log"
	"strconv"
	"time"
)

// These values are set by ldflags at build time.
// https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
var unixTime string
var Version string

func Time() time.Time {
	t, err := strconv.ParseInt(unixTime, 10, 64)
	if err != nil {
		log.Printf("no build time specified through ldflags")
		return time.Time{}
	}
	return time.Unix(t, 0)
}
