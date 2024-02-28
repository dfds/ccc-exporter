package util

import (
	"fmt"
	"time"
)

type YearMonthDayDate struct {
	Year  int
	Month int
	Day   int
}

func (d YearMonthDayDate) ToCSVString() string {
	return d.ToTimeUTC().Format("2006-01-02")
}
func (d YearMonthDayDate) ToTimeUTC() time.Time {
	return time.Date(d.Year, time.Month(d.Month), d.Day, 0, 0, 0, 0, time.UTC)
}

func (d YearMonthDayDate) ToFileNameFormat() string {
	return fmt.Sprintf("%d_%d_%d.csv", d.Year, d.Month, d.Day)
}

func (d YearMonthDayDate) String() string {
	return d.ToCSVString()
}

func ToYearMonthDayDate(t time.Time) YearMonthDayDate {
	return YearMonthDayDate{
		Year:  t.Year(),
		Month: int(t.Month()),
		Day:   t.Day(),
	}
}
