package stringutils

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func Title(s string) string {
	caser := cases.Title(language.English)

	return caser.String(s)
}

func GenerateRandomString(length int) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		b[i] = letters[n.Int64()]
	}

	return string(b)
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
