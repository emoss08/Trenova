package timeutils

// Clock tells the current time in Unix seconds. A service takes one through
// dependency injection rather than calling NowUnix, so every service in a
// process reads the same clock and a test can hold the time still.
type Clock func() int64

// SystemClock is the clock of the machine the process runs on.
func SystemClock() Clock { return NowUnix }
