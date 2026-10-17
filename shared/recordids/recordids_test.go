package recordids

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The cases are the web client's (client/apps/web/src/lib/__tests__/
// record-ids.test.ts), so the two strip the same text the same way.
func TestStrip_DropsAnIDInAnAsideWithItsLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"Approval completed: **SEED-PAY-009** created invoice **INV2610000011**.",
		Strip("Approval completed: **SEED-PAY-009** created invoice **INV2610000011** "+
			"(ID **inv_01M42BNHACKS99T13QKY88TVXV**)."))
	assert.Equal(t,
		"It is **INV2610000011**. Its status is Draft.",
		Strip("It is **INV2610000011**, `inv_01M42BNHACKS99T13QKY88TVXV`. Its status is Draft."))
}

func TestStrip_DropsAnIDNamedInASentence(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Approve proposal to proceed.",
		Strip("Approve proposal **ap_01M42BMTM3FGMGES7JN5715W3K** to proceed."))
	assert.Equal(t,
		"- **SEED-PAY-009** — shipment; Peak Distributing; **InReview**",
		Strip("- **SEED-PAY-009** — shipment `shp_01M3Q2Y4SRFE0YW60JY6F5NRW7`; "+
			"Peak Distributing; **InReview**"))
	assert.Equal(t,
		"The uncovered move was on **SEED-SHP-006**.",
		Strip("The uncovered move was on **SEED-SHP-006** "+
			"(shipment ID **shp_01M3Q2Y3JYTGFNHAJX8H1DB6JH**)."))
}

func TestStrip_LeavesArtifactLinksWebAddressesAndOrdinaryWordsAlone(t *testing.T) {
	t.Parallel()

	text := "See [Billing queue items](artifact:art_01M42AYFPF05FCCQ4R5R1Y7EQJ) and " +
		"https://x.test/r/inv_01M42BNHACKS99T13QKY88TVXV for SEED_PAY_009."
	assert.Equal(t, text, Strip(text))
	assert.Equal(t, "No ids here, just the BOL-2026-0109.",
		Strip("No ids here, just the BOL-2026-0109."))
	assert.False(t, Contains(text))
	assert.True(t, Contains("Shipment shp_01M3Q2Y4SRFE0YW60JY6F5NRW7 is late."))
}

// Closing up the words is for where an id came out; text that had none to
// take out keeps its spacing.
func TestStrip_LeavesSpacingAloneWhenNothingWasTakenOut(t *testing.T) {
	t.Parallel()

	text := "| Pro  | Status |\n| --- | --- |\n| A1  | Late |\n" +
		"[Table](artifact:art_01M42AYFPF05FCCQ4R5R1Y7EQJ)"
	assert.Equal(t, text, Strip(text))
}

func TestStrip_DropsASentenceThatOnlyNamesAnID(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"The partner was saved, as recorded in "+
			"[EDI partner Acme Retail Group](artifact:art_01M504P2QDKXPBM3QCTD7C0HWR). "+
			"Nothing is live yet.",
		Strip("The partner was saved, as recorded in "+
			"[EDI partner Acme Retail Group](artifact:art_01M504P2QDKXPBM3QCTD7C0HWR). "+
			"Its ID is edip_01M504P2M5BKZJ2EK36C1T4XKA. Nothing is live yet."))
	assert.Equal(t, "Nothing is live yet.",
		Strip("Acme Retail Group's ID: `edip_01M504P2M5BKZJ2EK36C1T4XKA`.\nNothing is live yet."))
	assert.Equal(t, "The partner is saved.",
		Strip("The partner is saved, and its ID is **edip_01M504P2M5BKZJ2EK36C1T4XKA**."))
	assert.Equal(t, "Saved.", Strip("Saved. The partner ID was edip_01M504P2M5BKZJ2EK36C1T4XKA"))
}

func TestStrip_KeepsASentenceThatSaysMoreThanTheID(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Its ID is the one Acme sent.",
		Strip("Its ID is edip_01M504P2M5BKZJ2EK36C1T4XKA the one Acme sent."))
}

func TestFind_NamesEachIDOnceInOrder(t *testing.T) {
	t.Parallel()

	text := `{"customer":{"id":"cus_01M4PHSWHNYFKQ9GSBZ7CB2PJX"},"locations":[` +
		`{"id":"loc_01M4SAMFDCV8AGW5XZ28YHJYC6"},{"id":"cus_01M4PHSWHNYFKQ9GSBZ7CB2PJX"},` +
		`{"id":"loc_01M4SAMFS0M58Y5T8XVG9X7Q9P"}],"code":"ANY-JAC-FL-065"}`

	assert.Equal(t, []string{
		"cus_01M4PHSWHNYFKQ9GSBZ7CB2PJX",
		"loc_01M4SAMFDCV8AGW5XZ28YHJYC6",
		"loc_01M4SAMFS0M58Y5T8XVG9X7Q9P",
	}, Find(text, 10))
	assert.Equal(t, []string{"cus_01M4PHSWHNYFKQ9GSBZ7CB2PJX"}, Find(text, 1))
	assert.Empty(t, Find(text, 0))
	assert.Empty(t, Find("no ids here", 10))
}
