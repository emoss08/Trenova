package ediinboundservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/stretchr/testify/require"
)

func TestParseAcknowledgments997(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000101*0*P*>~" +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*101*X*004010~" +
		"ST*997*0001~" +
		"AK1*SM*42~" +
		"AK2*204*0042~" +
		"AK5*A~" +
		"AK9*A*1*1*1~" +
		"SE*6*0001~" +
		"GE*1*101~" +
		"IEA*1*000000101~"

	interchange, err := parseTestInterchange(t, raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	transaction := interchange.transactions[0]
	require.Equal(t, edi.TransactionSet997, transaction.set)

	entries := parseAcknowledgments(&transaction)
	require.Len(t, entries, 1)
	entry := entries[0]
	require.Equal(t, "204", entry.originalTransactionSet)
	require.Equal(t, "0042", entry.originalControlNumber)
	require.Equal(t, "A", entry.acknowledgmentCode)
	require.Equal(t, "SM", entry.originalFunctionalGroupID)
	require.Equal(t, "42", entry.originalGroupControl)
	require.EqualValues(t, 1, entry.acceptedCount)

	status, ackErr := acknowledgmentResolution(&entry)
	require.Equal(t, edi.MessageAcknowledgmentStatusAccepted, status)
	require.Empty(t, ackErr)
}

func TestAcknowledgmentResolution_AcceptedWithErrorsKeepsWhatThePartnerNoted(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00501*000000103*0*P*>~" +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*103*X*005010~" +
		"ST*999*0001~" +
		"AK1*SM*44~" +
		"IK2*204*0044~" +
		"IK3*N1*7**8~" +
		"IK5*E*5~" +
		"AK9*E*1*1*1~" +
		"SE*7*0001~" +
		"GE*1*103~" +
		"IEA*1*000000103~"

	interchange, err := parseTestInterchange(t, raw)
	require.NoError(t, err)
	entries := parseAcknowledgments(&interchange.transactions[0])
	require.Len(t, entries, 1)
	require.Equal(t, "E", entries[0].acknowledgmentCode)

	status, ackErr := acknowledgmentResolution(&entries[0])
	require.Equal(t, edi.MessageAcknowledgmentStatusAcceptedWithErrors, status)
	require.Contains(t, ackErr, "acknowledgment code E")
	require.Contains(t, ackErr, "segment N1")
}

func TestParseAcknowledgmentsRejected999(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00501*000000102*0*P*>~" +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*102*X*005010~" +
		"ST*999*0001~" +
		"AK1*SM*43~" +
		"IK2*204*0043~" +
		"IK3*S5*5**8~" +
		"IK5*R*5~" +
		"AK9*R*1*1*0~" +
		"SE*7*0001~" +
		"GE*1*102~" +
		"IEA*1*000000102~"

	interchange, err := parseTestInterchange(t, raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	entries := parseAcknowledgments(&interchange.transactions[0])
	require.Len(t, entries, 1)
	require.Equal(t, "R", entries[0].acknowledgmentCode)
	require.Len(t, entries[0].diagnostics, 1)

	status, ackErr := acknowledgmentResolution(&entries[0])
	require.Equal(t, edi.MessageAcknowledgmentStatusRejected, status)
	require.Contains(t, ackErr, "acknowledgment code R")
}

func TestParseInterchangeKeepsEachTransactionInItsOwnGroup(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000317*0*P*>~" +
		"GS*SM*PARTNER*TRENOVA*20260107*1200*317*X*004010~" +
		"ST*204*0001~" +
		"B2**PART**LT-1**PP~" +
		"SE*3*0001~" +
		"GE*1*317~" +
		"GS*FA*PARTNER*TRENOVA*20260107*1200*318*X*004010~" +
		"ST*997*0001~" +
		"AK1*SM*42~" +
		"AK9*A*1*1*1~" +
		"SE*4*0001~" +
		"GE*1*318~" +
		"IEA*2*000000317~"

	interchange, err := parseTestInterchange(t, raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 2)

	tender := interchange.transactions[0]
	require.Equal(t, edi.TransactionSet204, tender.set)
	require.Equal(t, "SM", tender.functionalGroupID)
	require.Equal(t, "317", tender.groupControlNumber)

	ack := interchange.transactions[1]
	require.Equal(t, edi.TransactionSet997, ack.set)
	require.Equal(t, "FA", ack.functionalGroupID)
	require.Equal(t, "318", ack.groupControlNumber)
}

func parseTestInterchange(t *testing.T, raw string) (*parsedInterchange, error) {
	t.Helper()
	interchange, problem, err := parseInterchange(t.Context(), raw)
	if err != nil {
		return nil, err
	}
	if problem != nil {
		return nil, problem
	}
	return interchange, nil
}

// partnerTender is a 204 as a partner sends it: each stop's S5 loop carries its own dates
// and party, and the tender's bill of lading is in L11 with qualifier BM.
const partnerTender = "ISA*00*          *00*          *ZZ*PARTNER        *ZZ*TRENOVA        " +
	"*260107*1200*^*00401*000000201*0*P*>~" +
	"GS*SM*PARTNER*TRENOVA*20260107*1200*201*X*004010~" +
	"ST*204*0001~" +
	"B2**PTNR**SHIP-778899**PP~" +
	"B2A*00~" +
	"L11*BOL-778899*BM~" +
	"S5*1*LD*42000*L*24*PC~" +
	"G62*69*20260109*U*0800~" +
	"N1*SH*ACME DC*93*ACME-DC~" +
	"N3*100 MAIN ST~" +
	"N4*CHICAGO*IL*60601*US~" +
	"S5*2*UL*42000*L*24*PC~" +
	"G62*70*20260111*X*1600~" +
	"N1*CN*BIGBOX STORE 14*93*BIGBOX-14~" +
	"N3*200 MARKET ST~" +
	"N4*DALLAS*TX*75201*US~" +
	"L3*42000*G***185000~" +
	"SE*16*0001~" +
	"GE*1*201~" +
	"IEA*1*000000201~"
