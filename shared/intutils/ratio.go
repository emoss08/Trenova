package intutils

func RatioLeadExceedsPoints(num, den, otherNum, otherDen, points int) bool {
	if den <= 0 || otherDen <= 0 {
		return false
	}

	lead := 100 * (int64(num)*int64(otherDen) - int64(otherNum)*int64(den))
	return lead > int64(points)*int64(den)*int64(otherDen)
}
