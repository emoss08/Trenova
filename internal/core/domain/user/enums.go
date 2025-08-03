/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package user

type TimeFormat string

const (
	// TimeFormat12Hour is the 12-hour time format
	TimeFormat12Hour = TimeFormat("12-hour")

	// TimeFormat24Hour is the 24-hour time format (commonly known as military time)
	TimeFormat24Hour = TimeFormat("24-hour")
)
