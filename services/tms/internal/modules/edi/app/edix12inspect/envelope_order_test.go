package edix12inspect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	orderISA = "ISA*00*          *00*          *ZZ*ACMEFOODS      *ZZ*TRENOVA        " +
		"*261013*0912*^*00401*000000317*0*P*>~"
	orderIEA = "IEA*2*000000317~"
)

func orderDiagnostics(raw string) []int {
	result := InspectX12(&InspectX12Request{RawX12: raw})
	indexes := make([]int, 0)
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "x12.envelope.order" {
			indexes = append(indexes, diagnostic.SegmentIndex)
		}
	}
	return indexes
}

func TestEnvelopeOrder_AllowsManyTransactionSetsInManyGroups(t *testing.T) {
	t.Parallel()

	raw := orderISA +
		"GS*SM*ACMEFOODS*TRENOVA*20261013*0912*317*X*004010~" +
		"ST*204*0001~B2A*00~SE*3*0001~" +
		"ST*204*0002~B2A*04~SE*3*0002~" +
		"GE*2*317~" +
		"GS*FA*ACMEFOODS*TRENOVA*20261013*0912*318*X*004010~" +
		"ST*997*0001~AK5*A~SE*3*0001~" +
		"GE*1*318~" +
		orderIEA

	assert.Empty(t, orderDiagnostics(raw))
}

func TestEnvelopeOrder_FlagsATransactionSetOutsideItsGroup(t *testing.T) {
	t.Parallel()

	raw := orderISA +
		"ST*204*0001~B2A*00~SE*3*0001~" +
		"GS*SM*ACMEFOODS*TRENOVA*20261013*0912*317*X*004010~GE*0*317~" +
		"IEA*1*000000317~"

	assert.Equal(t, []int{2}, orderDiagnostics(raw))
}

func TestEnvelopeOrder_FlagsAGroupAfterTheInterchangeClosed(t *testing.T) {
	t.Parallel()

	raw := orderISA +
		"GS*SM*ACMEFOODS*TRENOVA*20261013*0912*317*X*004010~ST*204*0001~SE*2*0001~GE*1*317~" +
		"IEA*1*000000317~" +
		"GS*SM*ACMEFOODS*TRENOVA*20261013*0912*318*X*004010~ST*204*0001~SE*2*0001~GE*1*318~"

	assert.Equal(t, []int{7}, orderDiagnostics(raw))
}

func TestEnvelopeOrder_FlagsAGroupClosedWhileItsTransactionSetIsOpen(t *testing.T) {
	t.Parallel()

	raw := orderISA +
		"GS*SM*ACMEFOODS*TRENOVA*20261013*0912*317*X*004010~ST*204*0001~B2A*00~GE*1*317~SE*3*0001~" +
		"IEA*1*000000317~"

	assert.Equal(t, []int{5}, orderDiagnostics(raw))
}

func TestEnvelopeOrder_LeavesAMissingEnvelopeToTheMissingSegmentFindings(t *testing.T) {
	t.Parallel()

	assert.Empty(t, orderDiagnostics("ST*204*0001~B2A*00~SE*3*0001~"))
}
