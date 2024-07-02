// COPYRIGHT(c) 2024 Trenova

package internal

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func EnvVar(key string) string {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}
