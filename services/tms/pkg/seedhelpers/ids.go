package seedhelpers

import (
	"crypto/md5" //nolint:gosec // identifier derivation, not security
	"encoding/hex"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

// DeterministicID derives a stable PULID-shaped identifier from a key, using
// the same `prefix || upper(substr(md5(key), 1, 26))` recipe the SQL
// migrations use for backfilled rows. Seeds that mirror a migration backfill
// build the same key so their INSERT ... ON CONFLICT DO NOTHING lands on the
// existing row instead of creating a twin.
func DeterministicID(prefix, key string) pulid.ID {
	sum := md5.Sum([]byte(key)) //nolint:gosec // identifier derivation, not security
	return pulid.ID(prefix + strings.ToUpper(hex.EncodeToString(sum[:]))[:26])
}
