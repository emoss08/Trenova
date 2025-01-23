package dbtests_test

import (
	"os"
	"testing"

	"github.com/emoss08/trenova/test/testutils"
)

func TestMain(m *testing.M) {
	code := m.Run()

	testutils.CleanupTestDB()

	os.Exit(code)
}
