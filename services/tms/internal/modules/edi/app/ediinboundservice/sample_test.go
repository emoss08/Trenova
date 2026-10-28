package ediinboundservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSample_ReadsTheEnvelopeAndEachTransaction(t *testing.T) {
	t.Parallel()

	raw := "ISA*00*          *00*          *ZZ*ACMERETAIL     *02*TRNV           " +
		"*260107*1200*U*00401*000000103*0*T*>~" +
		"GS*SM*ACMERETAIL*TRNV*20260107*1200*103*X*004010~" +
		"ST*204*0001~" +
		"B2**TRNV**LOAD-77**CC~" +
		"B2A*00~" +
		"SE*4*0001~" +
		"GE*1*103~" +
		"IEA*1*000000103~"

	sample, err := ParseSample(t.Context(), raw)
	require.NoError(t, err)
	assert.Equal(t, "ZZ", sample.SenderQualifier)
	assert.Equal(t, "ACMERETAIL", sample.SenderID)
	assert.Equal(t, "02", sample.ReceiverQualifier)
	assert.Equal(t, "TRNV", sample.ReceiverID)
	assert.Equal(t, "00401", sample.InterchangeVersion)
	assert.Equal(t, "T", sample.UsageIndicator)
	assert.Equal(t, "ACMERETAIL", sample.ApplicationSender)
	assert.Equal(t, "TRNV", sample.ApplicationReceiver)
	assert.Equal(t, "SM", sample.FunctionalGroupID)
	assert.Equal(t, "004010", sample.GroupVersion)
	require.Len(t, sample.Transactions, 1)
	transaction := sample.Transactions[0]
	assert.Equal(t, edidocument.TransactionSet204, transaction.TransactionSet)
	assert.Equal(t, "0001", transaction.ControlNumber)
}

func TestParseSample_RefusesTextThatIsNotAnInterchange(t *testing.T) {
	t.Parallel()

	_, err := ParseSample(t.Context(), "ST*204*0001~SE*1*0001~")
	require.Error(t, err)
}
