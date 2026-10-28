package ediconnectiondraft

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pdfSamplePage = "9. Sample 204\n" +
	"ISA*00*   *00*   *01*007912345   *02*TRNV   *261014*0930*U*00401*000000\n" +
	"123*0*T*>~\nGS*SM*ACMERETAIL*TRNV*20261014*0930*123*X*004010~\nST*204*0001~\n" +
	"B2**TRNV**ACM-55821**PP~\nB2A*00~\nL11*BOL-77812*BM~\nS5*1*CL~\nN1*SH*ACME DC 14~\n" +
	"S5*2*CU~\nN1*CN*ACME STORE 220~\nSE*9*0001~\nGE*1*123~\nIEA*1*000000123~"

func TestFindSamples_ReadsASamplePrintedInAPDF(t *testing.T) {
	t.Parallel()

	samples := findSamples(
		newPageIndex([]ediconnectiondraftports.SpecPage{{Number: 7, Text: pdfSamplePage}}),
	)
	require.Len(t, samples, 1)
	sample := samples[0]
	sample.parse(t.Context())
	require.NoError(
		t,
		sample.err,
		"a PDF squeezes the ISA's padding and wraps it; the sample still reads",
	)

	assert.Equal(t, 7, sample.page)
	assert.Equal(t, edi.TransactionSet204, sample.transactionSet())
	assert.Equal(t, "01", sample.parsed.SenderQualifier)
	assert.Equal(t, "007912345", sample.parsed.SenderID)
	assert.Equal(t, "02", sample.parsed.ReceiverQualifier)
	assert.Equal(t, "TRNV", sample.parsed.ReceiverID)
	assert.Equal(t, "T", sample.parsed.UsageIndicator)
	assert.Equal(t, ">", sample.parsed.Separators.Component)
	assert.Equal(t, "~", sample.parsed.Separators.Segment)
	assert.Equal(t,
		"ISA*00*   *00*   *01*007912345   *02*TRNV   *261014*0930*U*00401*000000123*0*T*>~",
		sample.isaSegment(), "the quote is the ISA as the page prints it, its wrap joined")
}
