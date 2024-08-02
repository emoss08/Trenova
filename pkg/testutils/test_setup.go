package testutils

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	InitTestEnvironment()
	code := m.Run()
	CleanupTestDatabase()
	os.Exit(code)
}
