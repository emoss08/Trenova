package cmd

import (
	"encoding/base64"
	"log"

	"github.com/gorilla/securecookie"
)

const keyLength = 32

func main() {
	// Generate a 32-byte random key
	key := securecookie.GenerateRandomKey(keyLength)
	if key == nil {
		log.Fatal("Failed to generate key")
		return
	}

	// Encode the key in base64
	encodedKey := base64.StdEncoding.EncodeToString(key)

	log.Printf("Generated key: %v\n", encodedKey)
}
