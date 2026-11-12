package recordshare_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
A share's link is joined to the app's own address in the email it sends, so
it is only ever an address inside the app. Anything that could send the
reader somewhere else is refused before it is stored.
*/
func TestValidPath_IsOnlyAnAddressInsideTheApp(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]bool{
		"/billing/invoices?item=inv_1&tab=charges": true,
		"/desk/r/shipment/shp_1":                   true,
		"/desk/needs?lens=decisions&item=wt_1":     true,
		"":                                         false,
		"billing/invoices":                         false,
		"//evil.example/billing":                   false,
		"https://evil.example/billing":             false,
		"/desk\\r/shipment":                        false,
		"/desk/r/ship ment":                        false,
		"/desk\n/r":                                false,
		"/" + strings.Repeat("a", recordshare.MaxPathLength): false,
	} {
		assert.Equal(t, want, recordshare.ValidPath(path), path)
	}
}

func TestRevoke_KeepsTheShareAndSaysWhoTookItBackAndWhen(t *testing.T) {
	t.Parallel()

	share := &recordshare.Share{
		RecordID:     pulid.MustNew("inv_"),
		SharedWithID: pulid.MustNew("usr_"),
		SharedByID:   pulid.MustNew("usr_"),
	}
	assert.False(t, share.Revoked())

	by := pulid.MustNew("usr_")
	share.Revoke(by, 1_800_000_000)

	assert.True(t, share.Revoked())
	assert.Equal(t, int64(1_800_000_000), *share.RevokedAt)
	assert.Equal(t, by, share.RevokedByID)
}
