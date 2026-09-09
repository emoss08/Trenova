package ifta

import (
	"errors"
	"fmt"
	"time"
)

const (
	MinPeriodYear = 2000
	MaxPeriodYear = 2100

	monthsPerQuarter = 3
)

var (
	ErrPeriodYearOutOfRange = errors.New("period year must be between 2000 and 2100")
	ErrPeriodQuarterInvalid = errors.New("period quarter must be between 1 and 4")
)

type Period struct {
	Year    int `json:"year"`
	Quarter int `json:"quarter"`
}

func NewPeriod(year, quarter int) Period {
	return Period{Year: year, Quarter: quarter}
}

func PeriodOf(ts int64, loc *time.Location) Period {
	if loc == nil {
		loc = time.UTC
	}
	local := time.Unix(ts, 0).In(loc)

	return Period{
		Year:    local.Year(),
		Quarter: (int(local.Month())-1)/monthsPerQuarter + 1,
	}
}

func (p Period) Validate() error {
	if p.Year < MinPeriodYear || p.Year > MaxPeriodYear {
		return ErrPeriodYearOutOfRange
	}
	if p.Quarter < 1 || p.Quarter > 4 {
		return ErrPeriodQuarterInvalid
	}
	return nil
}

func (p Period) IsZero() bool { return p.Year == 0 && p.Quarter == 0 }

func (p Period) Key() string { return fmt.Sprintf("%dQ%d", p.Year, p.Quarter) }

func (p Period) Label() string { return fmt.Sprintf("Q%d %d", p.Quarter, p.Year) }

func (p Period) StartMonth() time.Month {
	return time.Month((p.Quarter-1)*monthsPerQuarter + 1)
}

func (p Period) Bounds(loc *time.Location) (start, end int64) {
	if loc == nil {
		loc = time.UTC
	}
	startMonth := p.StartMonth()
	startTime := time.Date(p.Year, startMonth, 1, 0, 0, 0, 0, loc)
	endTime := time.Date(p.Year, startMonth+monthsPerQuarter, 1, 0, 0, 0, 0, loc)

	return startTime.Unix(), endTime.Unix()
}

func (p Period) DueDate(loc *time.Location) int64 {
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(p.Year, p.StartMonth()+monthsPerQuarter+1, 0, 0, 0, 0, 0, loc).Unix()
}

func (p Period) Previous() Period {
	if p.Quarter <= 1 {
		return Period{Year: p.Year - 1, Quarter: 4}
	}
	return Period{Year: p.Year, Quarter: p.Quarter - 1}
}

func (p Period) Next() Period {
	if p.Quarter >= 4 {
		return Period{Year: p.Year + 1, Quarter: 1}
	}
	return Period{Year: p.Year, Quarter: p.Quarter + 1}
}

func (p Period) Before(other Period) bool {
	if p.Year != other.Year {
		return p.Year < other.Year
	}
	return p.Quarter < other.Quarter
}

func (p Period) Equal(other Period) bool {
	return p.Year == other.Year && p.Quarter == other.Quarter
}

func (p Period) Contains(ts int64, loc *time.Location) bool {
	start, end := p.Bounds(loc)
	return ts >= start && ts < end
}
