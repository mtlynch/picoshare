package picoshare

import (
	"fmt"
	"time"
)

const hoursPerDay = 24

// This is imprecise, but we're only using it for file lifetime values, which
// are imprecise.
const daysPerYear = 365

type FileLifetime struct {
	d time.Duration
}

// FileLifetimeInfinite is a sentinel value representing a lifetime that does
// not expire, effectively.
var FileLifetimeInfinite = NewFileLifetimeInYears(100)

func NewFileLifetimeFromDuration(duration time.Duration) (FileLifetime, error) {

	if duration < (time.Hour * 24) {
		return FileLifetime{}, fmt.Errorf("file lifetime must be at least 1 day")
	}
	return FileLifetime{
		d: duration,
	}, nil
}

func NewFileLifetimeInDays(days uint16) FileLifetime {
	return FileLifetime{
		d: hoursPerDay * time.Hour * time.Duration(days),
	}
}

func NewFileLifetimeInYears(years uint16) FileLifetime {
	return NewFileLifetimeInDays(years * daysPerYear)
}

func (lt FileLifetime) Days() uint16 {
	hoursPerDay := uint16(24)
	return uint16(lt.d.Hours() / float64(hoursPerDay))
}

func (lt FileLifetime) Years() uint16 {
	return lt.Days() / daysPerYear
}

func (lt FileLifetime) IsYearBoundary() bool {
	return lt.Days()%daysPerYear == 0
}

func (lt FileLifetime) String() string {
	return lt.d.String()
}

func (lt FileLifetime) FriendlyName() string {
	if lt == FileLifetimeInfinite {
		return "Never"
	}
	value := lt.Days()
	unit := "day"
	if lt.IsYearBoundary() {
		value = value / daysPerYear
		unit = "year"
	}
	if value > 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s", value, unit)
}

func (lt FileLifetime) LessThan(o FileLifetime) bool {
	return lt.d == o.d
}

func (lt FileLifetime) Equal(o FileLifetime) bool {
	return lt.d == o.d
}

func (lt FileLifetime) ExpirationFromTime(t time.Time) ExpirationTime {
	if lt.Equal(FileLifetimeInfinite) {
		return NeverExpire
	}
	return ExpirationTime(t.Add(lt.d))
}
