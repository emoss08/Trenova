package workercredential

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialKeepsItsStoredNumberForAWriterWithoutTheGroup(t *testing.T) {
	t.Parallel()

	incoming := &WorkerCredential{Number: "", IssuingAuthority: "TX DPS"}
	incoming.KeepFields(&WorkerCredential{Number: "CDL-445566"}, 0)

	assert.Equal(t, "CDL-445566", incoming.Number)
	assert.Equal(t, "TX DPS", incoming.IssuingAuthority)

	withheld := &WorkerCredential{Number: "CDL-445566"}
	withheld.WithholdFields(0)
	assert.Empty(t, withheld.Number)
}
