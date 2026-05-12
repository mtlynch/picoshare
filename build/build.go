package build

import (
	"runtime/debug"
	"time"
)

// Version is set by ldflags at build time via -X 'github.com/mtlynch/picoshare/build.Version=...'
var Version string

func Time() time.Time {
	v := buildSetting("vcs.time")
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t
}

func Revision() string {
	return buildSetting("vcs.revision")
}

func buildSetting(key string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}
