package picoshare

import "fmt"

type Settings struct {
	DefaultFileLifetime FileLifetime
}

func (s Settings) String() string {
	return fmt.Sprintf("{lifetime=%s}", s.DefaultFileLifetime.FriendlyName())
}
