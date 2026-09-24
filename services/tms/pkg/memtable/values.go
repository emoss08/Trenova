package memtable

func OptionalNumber[N ~int | ~int32 | ~int64 | ~float32 | ~float64](value *N) (float64, bool) {
	if value == nil {
		return 0, false
	}

	return float64(*value), true
}
