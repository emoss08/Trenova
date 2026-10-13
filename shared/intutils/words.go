package intutils

import "strconv"

var smallNumberWords = [...]string{
	"zero", "one", "two", "three", "four", "five", "six",
	"seven", "eight", "nine", "ten", "eleven", "twelve",
}

func SmallNumberWord(n int) string {
	if n >= 0 && n < len(smallNumberWords) {
		return smallNumberWords[n]
	}

	return strconv.Itoa(n)
}
