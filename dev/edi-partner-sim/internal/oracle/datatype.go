package oracle

import (
	"strconv"
	"strings"
	"time"
)

type dataType struct {
	name    string
	numeric bool
}

func parseDataType(name string) (dataType, bool) {
	switch name {
	case "AN", "ID", "DT", "TM", "R":
		return dataType{name: name, numeric: name == "R"}, true
	case "N":
		return dataType{name: "N0", numeric: true}, true
	}
	if len(name) == 2 && name[0] == 'N' && name[1] >= '0' && name[1] <= '9' {
		return dataType{name: name, numeric: true}, true
	}
	return dataType{}, false
}

func (t dataType) length(value string) int {
	if !t.numeric {
		return len(value)
	}
	n := 0
	for i := range len(value) {
		if value[i] >= '0' && value[i] <= '9' {
			n++
		}
	}
	return n
}

func (t dataType) check(value string) (string, string) {
	switch {
	case t.name == "DT":
		if !allDigits(value) || !validDate(value) {
			return codeInvalidDate, "is not a CCYYMMDD or YYMMDD date"
		}
	case t.name == "TM":
		if !allDigits(value) || !validTime(value) {
			return codeInvalidTime, "is not an HHMM, HHMMSS or HHMMSSDD time"
		}
	case t.name == "R":
		if !validDecimal(value) {
			return codeInvalidCharacter, "is not a decimal number"
		}
	case t.numeric:
		if strings.Contains(value, ".") {
			return codeInvalidCharacter, "has an explicit decimal point in an implied-decimal " + t.name
		}
		if !allDigits(strings.TrimPrefix(value, "-")) || value == "-" {
			return codeInvalidCharacter, "is not an integer"
		}
	default:
		for i := range len(value) {
			if value[i] < 0x20 || value[i] > 0x7E {
				return codeInvalidCharacter, "holds a character outside the X12 character set"
			}
		}
	}
	return "", ""
}

func allDigits(v string) bool {
	if v == "" {
		return false
	}
	for i := range len(v) {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
	}
	return true
}

func digitsWithin(v string, minLen, maxLen int) bool {
	return len(v) >= minLen && len(v) <= maxLen && allDigits(v)
}

func validDecimal(v string) bool {
	v = strings.TrimPrefix(v, "-")
	digits, points := 0, 0
	for i := range len(v) {
		switch {
		case v[i] >= '0' && v[i] <= '9':
			digits++
		case v[i] == '.':
			points++
		default:
			return false
		}
	}
	return digits > 0 && points <= 1
}

func validDate(v string) bool {
	if !allDigits(v) {
		return false
	}
	var layout string
	switch len(v) {
	case 6:
		layout = "060102"
	case 8:
		layout = "20060102"
	default:
		return false
	}
	_, err := time.Parse(layout, v)
	return err == nil
}

func validTime(v string) bool {
	if !allDigits(v) {
		return false
	}
	switch len(v) {
	case 4, 6, 7, 8:
	default:
		return false
	}
	hh, _ := strconv.Atoi(v[0:2])
	mm, _ := strconv.Atoi(v[2:4])
	if hh > 23 || mm > 59 {
		return false
	}
	if len(v) >= 6 {
		ss, _ := strconv.Atoi(v[4:6])
		return ss <= 59
	}
	return true
}
