package intutils

// RoundSignificant rounds value to the given number of significant figures,
// half away from zero: 1,234 at two figures is 1,200 and 1,250 is 1,300. A
// count said to two figures stays the same while the work it counts barely
// moves, so text built from it changes only when the work does.
func RoundSignificant(value int64, digits int) int64 {
	if digits <= 0 || value == 0 {
		return value
	}

	magnitude := Abs(value)
	var scale int64 = 1
	for limit := pow10(digits); magnitude >= limit; magnitude /= 10 {
		scale *= 10
	}
	if scale == 1 {
		return value
	}

	rounded := (Abs(value) + scale/2) / scale * scale
	if value < 0 {
		return -rounded
	}

	return rounded
}

// RoundedShare is part as a whole-number percentage of whole, rounded half up
// to the nearest step: 33 of 100 at a step of 5 is 35. A whole of zero or less
// has no share and gives zero; a step of zero or less is taken as one.
func RoundedShare(part, whole int64, step int) int {
	if whole <= 0 || part <= 0 {
		return 0
	}
	if step <= 0 {
		step = 1
	}
	if part >= whole {
		return 100
	}

	tenths := part * 1000 / whole
	s := int64(step) * 10
	rounded := int((tenths + s/2) / s * int64(step))

	return Clamp(rounded, 0, 100)
}

// RoundToStep rounds value to the nearest multiple of step, half away from
// zero: 63 to a step of 5 is 65 and 62 is 60. A step of one or less returns
// value unchanged.
func RoundToStep(value, step int64) int64 {
	if step <= 1 {
		return value
	}

	rounded := (Abs(value) + step/2) / step * step
	if value < 0 {
		return -rounded
	}

	return rounded
}

func pow10(n int) int64 {
	result := int64(1)
	for range n {
		result *= 10
	}

	return result
}
