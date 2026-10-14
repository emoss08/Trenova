package ediinboundservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const simulatorTender = "ISA*00*          *00*          *ZZ*SIMPARTNER     *ZZ*TRENOVA        " +
	"*261014*1513*^*00401*000000042*0*T*>~" +
	"GS*SM*SIMPARTNER*TRENOVA*20261014*1513*42*X*004010~" +
	"ST*204*0001~B2**SIML**SIM000042**PP~B2A*00~L11*SIM000042*CR~" +
	"S5*1*CL~G62*10*20261014*Y*1513~N1*SF*Simulator Shipper*93*SIM-SHIP~" +
	"N3*100 Simulator Way~N4*Chicago*IL*60601*US~" +
	"S5*2*CU~G62*68*20261016*Z*1513~N1*ST*Simulator Consignee*93*SIM-CONS~" +
	"N3*200 Receiver Road~N4*Dallas*TX*75201*US~L3*1000*G***150000~SE*16*0001~" +
	"GE*1*42~IEA*1*000000042~"

func TestInspectInboundFile_LaysOutTheInterchangeTheFileCarried(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, simulatorTender)
	req := ediports.GetEDIInboundFileByIDRequest{ID: fixture.file.ID, TenantInfo: fixture.tenantInfo()}
	fixture.inboundFileRepo.EXPECT().GetInboundFileByID(t.Context(), req).Return(fixture.file, nil).Once()

	result, err := fixture.service.InspectInboundFile(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, simulatorTender, result.RawX12)
	require.Len(t, result.Transactions, 1)
	assert.Equal(t, "204", result.Transactions[0].TransactionSet)
	assert.Equal(t, 16, result.Transactions[0].ActualSegments)
	require.Len(t, result.Groups, 1)
	assert.Equal(t, "000000042", result.Envelope.ISAControlNumber)
	assert.Empty(t, result.Diagnostics, "a well-formed interchange has nothing to flag")
}

func TestInspectInboundFile_FlagsWhatTheFileGotWrong(t *testing.T) {
	t.Parallel()

	miscounted := "ISA*00*          *00*          *ZZ*SIMPARTNER     *ZZ*TRENOVA        " +
		"*261014*1513*^*00401*000000043*0*T*>~GS*SM*SIMPARTNER*TRENOVA*20261014*1513*43*X*004010~" +
		"ST*204*0001~B2**SIML**SIM000043**PP~B2A*00~SE*15*0001~GE*1*43~IEA*1*000000043~"
	fixture := newInboundFixture(t, miscounted)
	req := ediports.GetEDIInboundFileByIDRequest{ID: fixture.file.ID, TenantInfo: fixture.tenantInfo()}
	fixture.inboundFileRepo.EXPECT().GetInboundFileByID(t.Context(), req).Return(fixture.file, nil).Once()

	result, err := fixture.service.InspectInboundFile(t.Context(), req)
	require.NoError(t, err)

	codes := make([]string, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	assert.Contains(t, codes, "x12.count.se01_mismatch")
}

func TestInspectInboundFile_RefusesAFileWhoseRawContentWasPurged(t *testing.T) {
	t.Parallel()

	fixture := newInboundFixture(t, "")
	purgedAt := int64(1_760_000_000)
	fixture.file.RawPurgedAt = &purgedAt
	req := ediports.GetEDIInboundFileByIDRequest{ID: fixture.file.ID, TenantInfo: fixture.tenantInfo()}
	fixture.inboundFileRepo.EXPECT().GetInboundFileByID(t.Context(), req).Return(fixture.file, nil).Once()

	result, err := fixture.service.InspectInboundFile(t.Context(), req)

	require.Nil(t, result)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "rawContent", validation.Field)
}
