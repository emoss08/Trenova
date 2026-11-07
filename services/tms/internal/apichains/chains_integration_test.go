//go:build integration

package apichains_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/apichain/chainrun"
)

func TestChains(t *testing.T) {
	chainrun.RunDir(t, "testdata/chains")
}
