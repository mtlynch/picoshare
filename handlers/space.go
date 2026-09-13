package handlers

import "github.com/mtlynch/picoshare/space"

// SpaceCheckFunc measures PicoShare storage usage.
type SpaceCheckFunc func() (space.Usage, error)
