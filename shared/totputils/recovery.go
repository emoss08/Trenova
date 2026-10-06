package totputils

import (
	"crypto/rand"
	"fmt"
	"strings"
)

const (
	RecoveryCodeCount  = 10
	recoveryCodeGroups = 2
	recoveryGroupSize  = 5
	recoveryAlphabet   = "abcdefghjkmnpqrstuvwxyz23456789"
)

func GenerateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("recovery code count must be positive, got %d", count)
	}

	codes := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for len(codes) < count {
		code, err := recoveryCode()
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[code]; duplicate {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}

	return codes, nil
}

func NormalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(code)))
}

func recoveryCode() (string, error) {
	raw := make([]byte, recoveryCodeGroups*recoveryGroupSize)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate recovery code: %w", err)
	}

	var b strings.Builder
	b.Grow(len(raw) + recoveryCodeGroups - 1)
	for i, value := range raw {
		if i > 0 && i%recoveryGroupSize == 0 {
			b.WriteByte('-')
		}
		b.WriteByte(recoveryAlphabet[int(value)%len(recoveryAlphabet)])
	}

	return b.String(), nil
}
