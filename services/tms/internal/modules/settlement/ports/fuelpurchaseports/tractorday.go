package fuelpurchaseports

import "context"

const (
	TractorDaysRecent   = 1
	TractorDaysTrailing = 62
)

type TractorDayRefresher interface {
	RefreshTractorDays(ctx context.Context, days int) (int64, error)
}
