package documentextractionjobs

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func probingHarness(t *testing.T, probeErr error) *doclingHarness {
	t.Helper()

	h := newDoclingHarness(t, true)
	doc := h.docs.Doc
	_, err := h.controlRepo.GetOrCreate(
		t.Context(),
		doc.OrganizationID,
		doc.BusinessUnitID,
	)
	require.NoError(t, err)
	h.converter.EXPECT().Enabled().Return(true).Maybe()
	fake := imagingfake.New()
	fake.ProbeFunc = func(
		context.Context,
		*kernelports.ImagingProbeRequest,
	) (*kernelports.ImagingProbe, error) {
		return nil, probeErr
	}
	h.activities.imaging = fake
	h.activities.conversionCfg = &documentconversionconfig.Config{
		DoclingURL: "http://docling:5001",
		MaxPages:   10,
	}
	return h
}

func (h *doclingHarness) convert(t *testing.T) (*convertedFile, *conversionFailure, error) {
	t.Helper()

	doc := h.docs.Doc
	return h.activities.convertDocument(
		t.Context(),
		doc,
		pagination.TenantInfo{OrgID: doc.OrganizationID, BuID: doc.BusinessUnitID},
		true,
	)
}

func TestConvertDocument_AnImagingOutageRetriesInsteadOfSkippingThePageGuard(t *testing.T) {
	t.Parallel()

	for name, probeErr := range map[string]error{
		"unavailable":  fmt.Errorf("%w: no endpoint", kernelports.ErrImagingUnavailable),
		"timeout":      fmt.Errorf("%w: deadline", kernelports.ErrImagingTimeout),
		"breaker open": fmt.Errorf("%w: circuit open", kernelports.ErrImagingUnavailable),
		"deadline":     context.DeadlineExceeded,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := probingHarness(t, probeErr)

			converted, failure, err := h.convert(t)

			require.ErrorIs(t, err, probeErr, "the activity retries under its policy")
			assert.Nil(t, converted)
			assert.Nil(t, failure)
		})
	}
}

func TestConvertDocument_AnInvalidOrEncryptedPDFStillGoesToTheConverter(t *testing.T) {
	t.Parallel()

	for name, probeErr := range map[string]error{
		"invalid":   kernelports.ErrImagingInvalidDocument,
		"encrypted": kernelports.ErrImagingEncrypted,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := probingHarness(t, probeErr)
			h.converter.EXPECT().
				Submit(mock.Anything, mock.Anything).
				Return(kernelports.ConversionTaskID(""), kernelports.ErrDocumentNotConvertible).
				Once()

			_, failure, err := h.convert(t)

			require.NoError(t, err)
			require.NotNil(t, failure, "the converter, not the probe, decides")
		})
	}
}
