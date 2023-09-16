package sqlite

import (
	"time"

	"github.com/mtlynch/picoshare/v2/picoshare"
)

func (d DB) GetEntryDownloads(id picoshare.EntryID) ([]picoshare.DownloadRecord, error) {

	return []picoshare.DownloadRecord{
		{
			Time:     time.Now().Add(-25 * time.Minute),
			ClientIP: "192.168.1.1",
			Browser:  "Firefox",
			Platform: "Windows",
		},
		{
			Time:     time.Now().Add(-25 * time.Hour),
			ClientIP: "10.0.0.2",
			Browser:  "Chrome",
			Platform: "OS X",
		},
	}, nil
}
