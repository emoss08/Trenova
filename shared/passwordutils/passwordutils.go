package passwordutils

import (
	"bufio"
	_ "embed"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

//go:embed common_passwords.txt
var commonPasswordsSource string

var commonPasswords = sync.OnceValue(func() map[string]struct{} {
	passwords := make(map[string]struct{}, strings.Count(commonPasswordsSource, "\n")+1)
	scanner := bufio.NewScanner(strings.NewReader(commonPasswordsSource))
	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if line == "" {
			continue
		}
		passwords[line] = struct{}{}
	}

	return passwords
})

func IsCommon(password string) bool {
	normalized := strings.ToLower(strings.TrimSpace(password))
	if normalized == "" {
		return false
	}
	if _, ok := commonPasswords()[normalized]; ok {
		return true
	}

	return isSingleRune(normalized) || isSequential(normalized)
}

func ContainsIgnoringCase(password, fragment string) bool {
	fragment = strings.ToLower(strings.TrimSpace(fragment))
	if utf8.RuneCountInString(fragment) < 3 {
		return false
	}

	return strings.Contains(strings.ToLower(password), fragment)
}

func isSingleRune(value string) bool {
	first, _ := utf8.DecodeRuneInString(value)
	for _, r := range value {
		if r != first {
			return false
		}
	}

	return true
}

func isSequential(value string) bool {
	runes := []rune(value)
	if len(runes) < 4 {
		return false
	}

	step := runes[1] - runes[0]
	if step != 1 && step != -1 {
		return false
	}
	for i := 2; i < len(runes); i++ {
		if runes[i]-runes[i-1] != step {
			return false
		}
		if !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) {
			return false
		}
	}

	return true
}
