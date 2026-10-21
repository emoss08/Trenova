package totputils

import (
	"crypto/rand"
	"fmt"
	"io"
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
	return recoveryCodeFrom(rand.Reader)
}

func recoveryCodeFrom(source io.Reader) (string, error) {
	const length = recoveryCodeGroups * recoveryGroupSize
	limit := byte(256 - 256%len(recoveryAlphabet))

	var b strings.Builder
	b.Grow(length + recoveryCodeGroups - 1)
	buffer := make([]byte, length)
	written := 0
	for written < length {
		if _, err := io.ReadFull(source, buffer); err != nil {
			return "", fmt.Errorf("generate recovery code: %w", err)
		}
		for _, value := range buffer {
			if value >= limit {
				continue
			}
			if written > 0 && written%recoveryGroupSize == 0 {
				b.WriteByte('-')
			}
			b.WriteByte(recoveryAlphabet[int(value)%len(recoveryAlphabet)])
			written++
			if written == length {
				break
			}
		}
	}

	return b.String(), nil
}
