package dbhelper

const NowEpoch = "extract(epoch from current_timestamp)::bigint"

func MonthStartEpoch(column string) string {
	return "extract(epoch from date_trunc('month', to_timestamp(" + column +
		") at time zone 'UTC'))::bigint"
}
