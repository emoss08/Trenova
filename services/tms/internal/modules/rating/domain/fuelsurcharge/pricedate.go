package fuelsurcharge

import "time"

func NormalizePriceDate(value string) string {
	if len(value) < len(PriceDateLayout) {
		return value
	}

	day := value[:len(PriceDateLayout)]
	if _, err := time.Parse(PriceDateLayout, day); err != nil {
		return value
	}

	return day
}
