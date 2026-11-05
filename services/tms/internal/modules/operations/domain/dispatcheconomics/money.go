package dispatcheconomics

import "strconv"

const (
	centsPerDollar = 100
	halfDollar     = 50
)

func Dollars(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}

	return sign + "$" + strconv.FormatInt((cents+halfDollar)/centsPerDollar, 10)
}
