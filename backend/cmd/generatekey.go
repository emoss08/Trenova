package main

import (
	"encoding/base64"
	"fmt"

	"github.com/gorilla/securecookie"
)

func main() {
	// Generate a 32-byte random key
	key := securecookie.GenerateRandomKey(32)
	if key == nil {
		fmt.Println("Failed to generate a key")
		return
	}

	// Encode the key in base64
	encodedKey := base64.StdEncoding.EncodeToString(key)

	fmt.Println("Copy the following key to your .env file:", encodedKey)
}
