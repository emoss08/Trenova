package commodity

import (
	"strconv"
	"strings"
)

const freightClassPrefix = "Class"

var freightClassByNumber = func() map[float64]FreightClass {
	byNumber := make(map[float64]FreightClass, len(FreightClasses))
	for _, class := range FreightClasses {
		number, err := strconv.ParseFloat(class.Number(), 64)
		if err == nil {
			byNumber[number] = class
		}
	}
	return byNumber
}()

func (fc FreightClass) Number() string {
	if !fc.IsValid() {
		return ""
	}
	return strings.ReplaceAll(strings.TrimPrefix(string(fc), freightClassPrefix), "_", ".")
}

func ParseFreightClass(value string) (FreightClass, bool) {
	text := strings.TrimSpace(value)
	if len(text) >= len(freightClassPrefix) &&
		strings.EqualFold(text[:len(freightClassPrefix)], freightClassPrefix) {
		text = strings.TrimSpace(text[len(freightClassPrefix):])
	}

	number, err := strconv.ParseFloat(strings.ReplaceAll(text, "_", "."), 64)
	if err != nil {
		return "", false
	}

	class, ok := freightClassByNumber[number]
	return class, ok
}
