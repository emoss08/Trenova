package user

type TimeFormat string

const (
	// TimeFormat12Hour is the 12-hour time format
	TimeFormat12Hour = TimeFormat("12-hour")

	// TimeFormat24Hour is the 24-hour time format (commonly known as military time)
	TimeFormat24Hour = TimeFormat("24-hour")
)
