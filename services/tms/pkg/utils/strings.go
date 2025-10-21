package utils

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func JoinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func GenerateRandomDigits(count int) string {
	if count <= 0 {
		return ""
	}

	var result strings.Builder
	result.Grow(count)

	for range count {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return GenerateFallbackRandomDigits(count)
		}
		result.WriteString(n.String())
	}

	return result.String()
}

func GenerateFallbackRandomDigits(count int) string {
	var result strings.Builder
	result.Grow(count)

	nano := time.Now().UnixNano()
	for range count {
		digit := nano % 10
		result.WriteString(strconv.FormatInt(digit, 10))
		nano /= 10
		if nano == 0 {
			nano = time.Now().UnixNano()
		}
	}

	return result.String()
}

func ConvertCamelToSnake(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(s) + 10)

	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prevIsLower := unicode.IsLower(runes[i-1])
				nextIsLower := i < len(runes)-1 && unicode.IsLower(runes[i+1])

				if prevIsLower || nextIsLower {
					result.WriteByte('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func ConvertSnakeToCamel(s string) string {
	if s == "" {
		return ""
	}

	words := strings.Split(s, "_")
	if len(words) == 0 {
		return ""
	}

	var result strings.Builder
	for i, word := range words {
		if i == 0 {
			result.WriteString(strings.ToLower(word))
		} else if word != "" {
			result.WriteRune(unicode.ToUpper(rune(word[0])))
			if len(word) > 1 {
				result.WriteString(strings.ToLower(word[1:]))
			}
		}
	}

	return result.String()
}

func ToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

func ToPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func GenerateSecurePassword(length int) string {
	if length < 8 {
		length = 8
	}

	const (
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		numbers   = "0123456789"
		special   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	allChars := uppercase + lowercase + numbers + special

	password := make([]byte, length)

	password[0] = uppercase[secureRandomInt(len(uppercase))]
	password[1] = lowercase[secureRandomInt(len(lowercase))]
	password[2] = numbers[secureRandomInt(len(numbers))]
	password[3] = special[secureRandomInt(len(special))]

	for i := 4; i < length; i++ {
		password[i] = allChars[secureRandomInt(len(allChars))]
	}

	for i := length - 1; i > 0; i-- {
		j := secureRandomInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

func secureRandomInt(maxValue int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxValue)))
	if err != nil {
		panic(err)
	}
	return int(n.Int64())
}

func ToCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}

	result := parts[0]

	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return result
}

func ToConstName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

func RemoveStringSlice(slice, itemsToRemove []string) []string {
	removeMap := make(map[string]bool)
	for _, item := range itemsToRemove {
		removeMap[item] = true
	}

	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !removeMap[s] {
			result = append(result, s)
		}
	}
	return result
}

func MergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(a)+len(b))

	for _, s := range a {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}

	for _, s := range b {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}

	return result
}

func RemoveString(slice []string, item string) []string {
	result := make([]string, 0, len(slice))

	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}

	return result
}

func JoinStringsSep(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	n := len(sep) * (len(strs) - 1)
	for _, s := range strs {
		n += len(s)
	}

	b := make([]byte, 0, n)
	for i, s := range strs {
		if i > 0 {
			b = append(b, sep...)
		}
		b = append(b, s...)
	}
	return string(b)
}

func IsLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
