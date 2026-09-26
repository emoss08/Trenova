package captureservice

import (
	"bytes"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pairingCode(t *testing.T, err error) PairingErrorCode {
	t.Helper()

	var pairingErr *PairingError
	require.ErrorAs(t, err, &pairingErr)

	return pairingErr.Code
}

// pair runs the whole device authorization grant and returns the device's
// first credential.
func pair(t *testing.T, w *world, s *Service) *TokenPair {
	t.Helper()

	grant, err := s.StartPairing(t.Context(), &StartPairingRequest{
		MachineName:  "DISPATCH-07",
		WindowsUser:  "jdoe",
		AgentVersion: "1.0.0",
		Architecture: capture.ArchitectureX64,
	})
	require.NoError(t, err)
	assert.Regexp(t, `^[B-Z]{4}-[B-Z]{4}$`, grant.UserCode)
	assert.Equal(t, "https://app.trenova.test/capture/pair", grant.VerificationURI)

	_, err = s.ExchangePairing(t.Context(), grant.DeviceCode)
	assert.Equal(t, PairingAuthorizationPending, pairingCode(t, err))

	require.NoError(t, s.DecidePairing(t.Context(), &DecidePairingRequest{
		TenantInfo: w.tenant,
		UserCode:   grant.UserCode,
		Approve:    true,
	}))

	for _, p := range w.pairings {
		p.LastPolledAt = nil
	}

	tokens, err := s.ExchangePairing(t.Context(), grant.DeviceCode)
	require.NoError(t, err)

	return tokens
}

func principalFor(t *testing.T, s *Service, tokens *TokenPair) *DevicePrincipal {
	t.Helper()

	principal, err := s.Authenticate(t.Context(), tokens.AccessToken, "10.0.0.7")
	require.NoError(t, err)

	return principal
}

func TestPairingBindsTheDeviceToTheApprover(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)

	assert.Equal(t, w.tenant.UserID, tokens.UserID)
	assert.Equal(t, w.tenant.OrgID, tokens.OrganizationID)
	assert.Equal(t, "DISPATCH-07", tokens.DeviceName)

	device := w.devices[tokens.DeviceID]
	require.NotNil(t, device)
	assert.NotEqual(t, tokens.AccessToken, device.AccessTokenHash, "tokens are stored hashed")
	assert.NotEqual(t, tokens.RefreshToken, device.RefreshTokenHash)
}

func TestPairingIsExchangedExactlyOnce(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()

	grant, err := s.StartPairing(t.Context(), &StartPairingRequest{
		MachineName: "SCAN-1", AgentVersion: "1.0.0", Architecture: capture.ArchitectureX64,
	})
	require.NoError(t, err)
	require.NoError(t, s.DecidePairing(t.Context(), &DecidePairingRequest{
		TenantInfo: w.tenant, UserCode: grant.UserCode, Approve: true,
	}))

	_, err = s.ExchangePairing(t.Context(), grant.DeviceCode)
	require.NoError(t, err)

	_, err = s.ExchangePairing(t.Context(), grant.DeviceCode)
	assert.Equal(t, PairingInvalidGrant, pairingCode(t, err))
	assert.Len(t, w.devices, 1)
}

func TestPairingPollTooSoonSlowsDown(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	grant, err := s.StartPairing(t.Context(), &StartPairingRequest{
		MachineName: "SCAN-1", AgentVersion: "1.0.0", Architecture: capture.ArchitectureX64,
	})
	require.NoError(t, err)

	_, err = s.ExchangePairing(t.Context(), grant.DeviceCode)
	assert.Equal(t, PairingAuthorizationPending, pairingCode(t, err))
	_, err = s.ExchangePairing(t.Context(), grant.DeviceCode)
	assert.Equal(t, PairingSlowDown, pairingCode(t, err))
}

func TestDeniedPairingIsRefused(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	grant, err := s.StartPairing(t.Context(), &StartPairingRequest{
		MachineName: "SCAN-1", AgentVersion: "1.0.0", Architecture: capture.ArchitectureX64,
	})
	require.NoError(t, err)
	require.NoError(t, s.DecidePairing(t.Context(), &DecidePairingRequest{
		TenantInfo: w.tenant, UserCode: "  " + grant.UserCode + " ", Approve: false,
	}))

	_, err = s.ExchangePairing(t.Context(), grant.DeviceCode)
	assert.Equal(t, PairingAccessDenied, pairingCode(t, err))

	err = s.DecidePairing(t.Context(), &DecidePairingRequest{
		TenantInfo: w.tenant, UserCode: grant.UserCode, Approve: true,
	})
	assert.True(t, errortypes.IsNotFoundError(err), "a decided code cannot be decided again")
}

func TestApprovalNeedsCapturePermissionAndCaptureOn(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	grant, err := s.StartPairing(t.Context(), &StartPairingRequest{
		MachineName: "SCAN-1", AgentVersion: "1.0.0", Architecture: capture.ArchitectureX64,
	})
	require.NoError(t, err)

	w.denied[permission.ResourceCaptureBatch.String()+":create"] = true
	err = s.DecidePairing(t.Context(), &DecidePairingRequest{TenantInfo: w.tenant, UserCode: grant.UserCode, Approve: true})
	assert.True(t, errortypes.IsAuthorizationError(err))

	delete(w.denied, permission.ResourceCaptureBatch.String()+":create")
	w.control.EnableCapture = false
	err = s.DecidePairing(t.Context(), &DecidePairingRequest{TenantInfo: w.tenant, UserCode: grant.UserCode, Approve: true})
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestRefreshRotatesAndReuseRevokes(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	first := pair(t, w, s)

	second, err := s.Refresh(t.Context(), &RefreshRequest{RefreshToken: first.RefreshToken, AgentVersion: "1.1.0"})
	require.NoError(t, err)
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken)
	assert.Equal(t, "1.1.0", w.devices[first.DeviceID].AgentVersion)

	_, err = s.Authenticate(t.Context(), first.AccessToken, "")
	require.Error(t, err, "the replaced access token stops working")
	principalFor(t, s, second)

	_, err = s.Refresh(t.Context(), &RefreshRequest{RefreshToken: first.RefreshToken})
	assert.Equal(t, PairingInvalidGrant, pairingCode(t, err))
	assert.Equal(t, capture.DeviceRevoked, w.devices[first.DeviceID].Status,
		"presenting a replaced refresh token revokes the device")

	_, err = s.Authenticate(t.Context(), second.AccessToken, "")
	require.Error(t, err)
	_, err = s.Refresh(t.Context(), &RefreshRequest{RefreshToken: second.RefreshToken})
	assert.Equal(t, PairingInvalidGrant, pairingCode(t, err))
}

func TestRefreshEnforcesMinimumVersion(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)
	w.control.CaptureMinAgentVersion = "1.2.0"

	_, err := s.Refresh(t.Context(), &RefreshRequest{RefreshToken: tokens.RefreshToken, AgentVersion: "1.1.9"})
	var outdated *OutdatedAgentError
	require.ErrorAs(t, err, &outdated)
	assert.Equal(t, "1.2.0", outdated.Minimum)

	_, err = s.Refresh(t.Context(), &RefreshRequest{RefreshToken: tokens.RefreshToken, AgentVersion: "1.2.0"})
	require.NoError(t, err, "a refused refresh does not consume the token")
}

func TestAuthenticateRejectsForeignTokens(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	pair(t, w, s)

	for _, token := range []string{"", "tcd_rt_whatever", "tcd_at_unknown", "Bearer tcd_at_x"} {
		_, err := s.Authenticate(t.Context(), token, "")
		assert.True(t, errortypes.IsAuthenticationError(err), token)
	}
}

func TestRevokingAnotherPersonsDeviceNeedsPermission(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)

	other := w.tenant
	other.UserID = pulid.MustNew("usr_")
	w.denied[permission.ResourceCaptureDevice.String()+":update"] = true
	_, err := s.RevokeDevice(t.Context(), &RevokeDeviceRequest{TenantInfo: other, DeviceID: tokens.DeviceID})
	assert.True(t, errortypes.IsAuthorizationError(err))

	device, err := s.RevokeDevice(t.Context(), &RevokeDeviceRequest{TenantInfo: w.tenant, DeviceID: tokens.DeviceID})
	require.NoError(t, err)
	assert.Equal(t, capture.DeviceRevoked, device.Status)
	assert.Contains(t, w.published, "capture_device:revoked")
}

// scan opens a batch, uploads the pages and seals it.
func scan(
	t *testing.T,
	s *Service,
	principal *DevicePrincipal,
	in *OpenBatchInput,
	pages [][]byte,
	patchCodes map[int]string,
) *capture.CaptureBatch {
	t.Helper()

	batch, err := s.OpenBatch(t.Context(), principal, in)
	require.NoError(t, err)

	checksums := make([]string, 0, len(pages))
	for i, data := range pages {
		_, err = s.PutPage(t.Context(), principal, &PutPageInput{
			BatchID:   batch.ID,
			Sequence:  i + 1,
			Body:      bytes.NewReader(data),
			DPI:       300,
			PatchCode: patchCodes[i+1],
		})
		require.NoError(t, err)
		checksums = append(checksums, hashutils.SHA256BytesHex(data))
	}

	sealed, err := s.SealBatch(t.Context(), principal, &SealBatchInput{
		BatchID:        batch.ID,
		PageCount:      len(pages),
		ManifestDigest: ManifestDigest(checksums),
	})
	require.NoError(t, err)

	return sealed
}

func TestPageUploadIsIdempotentAndRefusesAnotherPage(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))

	batch, err := s.OpenBatch(t.Context(), principal, &OpenBatchInput{ClientKey: "k1", Source: capture.SourceScan})
	require.NoError(t, err)
	again, err := s.OpenBatch(t.Context(), principal, &OpenBatchInput{ClientKey: "k1", Source: capture.SourceScan})
	require.NoError(t, err)
	assert.Equal(t, batch.ID, again.ID, "the same client key opens the same batch")

	page := pdfPage(t, 200)
	first, err := s.PutPage(t.Context(), principal, &PutPageInput{BatchID: batch.ID, Sequence: 1, Body: bytes.NewReader(page)})
	require.NoError(t, err)
	retry, err := s.PutPage(t.Context(), principal, &PutPageInput{BatchID: batch.ID, Sequence: 1, Body: bytes.NewReader(page)})
	require.NoError(t, err)
	assert.Equal(t, first.ID, retry.ID)
	assert.Equal(t, 1, w.batches[batch.ID].ReceivedPageCount)

	_, err = s.PutPage(t.Context(), principal, &PutPageInput{
		BatchID: batch.ID, Sequence: 1, Body: bytes.NewReader(pdfPage(t, 100)),
	})
	assert.True(t, errortypes.IsConflictError(err))

	stored := w.objects[first.StoragePath]
	assert.True(t, bytes.HasPrefix(stored, []byte("sealed|capture_page|")), "pages are sealed at rest")
}

func TestPageUploadRejectsWhatIsNotAOnePagePDF(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	batch, err := s.OpenBatch(t.Context(), principal, &OpenBatchInput{ClientKey: "k", Source: capture.SourceScan})
	require.NoError(t, err)

	two, err := s.assembler.Assemble(t.Context(), []services.CaptureAssemblyPage{
		{PDF: pdfPage(t, 10)}, {PDF: pdfPage(t, 20)},
	})
	require.NoError(t, err)

	for name, body := range map[string][]byte{
		"not a pdf": []byte("GIF89a..."),
		"empty":     {},
		"two pages": two,
	} {
		_, err = s.PutPage(t.Context(), principal, &PutPageInput{BatchID: batch.ID, Sequence: 1, Body: bytes.NewReader(body)})
		assert.Error(t, err, name)
	}

	_, err = s.PutPage(t.Context(), principal, &PutPageInput{
		BatchID: batch.ID, Sequence: 1, Body: bytes.NewReader(pdfPage(t, 1)), PatchCode: "5",
	})
	assert.Error(t, err, "patch 5 does not exist")
}

func TestSealVerifiesTheManifest(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	batch, err := s.OpenBatch(t.Context(), principal, &OpenBatchInput{ClientKey: "k", Source: capture.SourceScan})
	require.NoError(t, err)
	page := pdfPage(t, 5)
	_, err = s.PutPage(t.Context(), principal, &PutPageInput{BatchID: batch.ID, Sequence: 1, Body: bytes.NewReader(page)})
	require.NoError(t, err)

	_, err = s.SealBatch(t.Context(), principal, &SealBatchInput{BatchID: batch.ID, PageCount: 2, ManifestDigest: "x"})
	assert.True(t, errortypes.IsConflictError(err), "a missing page refuses the seal")

	_, err = s.SealBatch(t.Context(), principal, &SealBatchInput{BatchID: batch.ID, PageCount: 1, ManifestDigest: "wrong"})
	assert.True(t, errortypes.IsConflictError(err), "a mismatched digest refuses the seal")

	sealed, err := s.SealBatch(t.Context(), principal, &SealBatchInput{
		BatchID: batch.ID, PageCount: 1, ManifestDigest: ManifestDigest([]string{hashutils.SHA256BytesHex(page)}),
	})
	require.NoError(t, err)
	assert.Equal(t, capture.BatchSealed, sealed.Status)

	_, err = s.PutPage(t.Context(), principal, &PutPageInput{
		BatchID: batch.ID, Sequence: 2, Body: bytes.NewReader(pdfPage(t, 6)),
	})
	assert.True(t, errortypes.IsConflictError(err), "a sealed batch takes no more pages")
}

func TestProcessSplitsOnPatchCodesAndCoverSheets(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))

	shipmentID := w.addRecord(permission.ResourceShipment.String())
	podType := w.addDocType("POD")
	issued, err := s.CreateCoverSheets(t.Context(), &CreateCoverSheetsInput{
		TenantInfo: w.tenant,
		Sheets: []CoverSheetSpec{{
			TargetType:     permission.ResourceShipment.String(),
			TargetID:       &shipmentID,
			DocumentTypeID: &podType,
		}},
	})
	require.NoError(t, err)
	require.Len(t, issued, 1)
	require.NotNil(t, issued[0].QRCode, "a sheet comes with the code to print on it")
	assert.Positive(t, issued[0].QRCode.Size)

	pages := [][]byte{pdfPage(t, 1), pdfPage(t, 2), pdfPage(t, 3), pdfPage(t, 4), pdfPage(t, 5), pdfPage(t, 6)}
	w.inspections[string(pages[3])] = &services.CapturePageInspection{
		InkCoverage: 0.1, Thumbnail: []byte("jpeg"), Codes: []string{issued[0].Payload},
	}
	w.inspections[string(pages[5])] = &services.CapturePageInspection{InkCoverage: 0.0001, Thumbnail: []byte("j")}

	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "stack", Source: capture.SourceScan},
		pages, map[int]string{2: "T"})

	result, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)

	items := w.itemsOf(batch.ID)
	require.Len(t, items, 3, "patch sheet after page 1, cover sheet before page 5")
	assert.Len(t, items[0].PageIDs, 1)
	assert.Len(t, items[1].PageIDs, 1)
	assert.Len(t, items[2].PageIDs, 2, "a blank page is kept without a profile saying to drop it")
	assert.Equal(t, capture.SuggestionCoverSheet, items[2].SuggestionSource)
	assert.Equal(t, shipmentID, *items[2].SuggestedID)
	assert.Equal(t, []pulid.ID{items[2].ID}, result.AutoFile, "cover-sheet routes are trusted by default")

	processed := w.batches[batch.ID]
	assert.Equal(t, capture.BatchReady, processed.Status)
	assert.Equal(t, 3, processed.ItemCount)

	for _, page := range w.pagesOf(batch.ID) {
		assert.Equal(t, capture.PageProcessed, page.Status)
		assert.NotEmpty(t, page.ThumbnailPath)
	}

	again, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)
	assert.True(t, again.Skipped, "processing a ready batch again does nothing")
}

func TestCoverSheetsDoNotAutoFileWhenTheOrganizationSaysNot(t *testing.T) {
	t.Parallel()

	w := newWorld()
	w.control.CaptureAutoFileCoverSheets = false
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))

	shipmentID := w.addRecord(permission.ResourceShipment.String())
	docType := w.addDocType("BOL")
	issued, err := s.CreateCoverSheets(t.Context(), &CreateCoverSheetsInput{
		TenantInfo: w.tenant,
		Sheets:     []CoverSheetSpec{{TargetType: "shipment", TargetID: &shipmentID, DocumentTypeID: &docType}},
	})
	require.NoError(t, err)

	pages := [][]byte{pdfPage(t, 1), pdfPage(t, 2)}
	w.inspections[string(pages[0])] = &services.CapturePageInspection{
		InkCoverage: 0.1, Codes: []string{issued[0].Payload},
	}
	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "c", Source: capture.SourceScan}, pages, nil)

	result, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)
	assert.Empty(t, result.AutoFile)
	require.Len(t, w.itemsOf(batch.ID), 1)
	assert.Equal(t, capture.SuggestionCoverSheet, w.itemsOf(batch.ID)[0].SuggestionSource)
}

func TestAnotherTenantsCoverSheetRoutesNothing(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))

	foreign := &capture.CaptureCoverSheet{
		ID:             pulid.MustNew("ccs_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		TokenHash:      hashutils.SHA256Hex("stolen"),
		ExpiresAt:      timeutils.NowUnix() + 1000,
	}
	w.coverSheets[foreign.ID] = foreign

	pages := [][]byte{pdfPage(t, 1), pdfPage(t, 2), pdfPage(t, 3)}
	w.inspections[string(pages[1])] = &services.CapturePageInspection{
		InkCoverage: 0.1, Codes: []string{capture.CoverSheetPayload("stolen")},
	}
	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "f", Source: capture.SourceScan}, pages, nil)

	_, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)

	items := w.itemsOf(batch.ID)
	require.Len(t, items, 2, "the sheet still divides the stack")
	for _, item := range items {
		assert.Nil(t, item.CoverSheetID)
		assert.Empty(t, item.SuggestionSource)
	}
}

func TestScanFromARecordFilesItself(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)
	principal := principalFor(t, s, tokens)

	shipmentID := w.addRecord("shipment")
	podType := w.addDocType("POD")
	req, err := s.CreateRequest(t.Context(), &CreateRequestInput{
		TenantInfo:     w.tenant,
		DeviceID:       tokens.DeviceID,
		Mode:           capture.RequestModeScan,
		TargetType:     "shipment",
		TargetID:       shipmentID,
		DocumentTypeID: &podType,
	})
	require.NoError(t, err)
	assert.Contains(t, w.published, "capture_batch:request.created")

	open, err := s.OpenRequests(t.Context(), principal)
	require.NoError(t, err)
	require.Len(t, open, 1)

	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "r", Source: capture.SourceScan, RequestID: &req.ID},
		[][]byte{pdfPage(t, 1), pdfPage(t, 2)}, nil)
	assert.Equal(t, capture.RequestCompleted, w.requests[req.ID].Status)

	result, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)
	require.Len(t, result.AutoFile, 1)

	item := w.itemsOf(batch.ID)[0]
	filed, err := s.FileItem(t.Context(), &FileItemInput{
		TenantInfo:     w.tenant,
		ItemID:         item.ID,
		TargetType:     item.SuggestedType,
		TargetID:       *item.SuggestedID,
		DocumentTypeID: item.SuggestedDocTypeID,
		Automatic:      true,
	})
	require.NoError(t, err)
	assert.Equal(t, capture.ItemFiled, filed.Status)
	require.NotNil(t, filed.DocumentID)

	settled := w.batches[batch.ID]
	assert.Equal(t, capture.BatchFiled, settled.Status)
	assert.Equal(t, 1, settled.FiledItemCount)

	session := w.sessions[*filed.UploadSessionID]
	assert.Equal(t, "shipment", session.ResourceType)
	assert.Equal(t, shipmentID.String(), session.ResourceID)
	count, err := s.assembler.PageCount(t.Context(), w.uploaded[session.ID])
	require.NoError(t, err)
	assert.Equal(t, 2, count, "both pages go into the one document")
}

func TestFilingIsCheckedAsThePersonFiling(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "p", Source: capture.SourceScan},
		[][]byte{pdfPage(t, 1)}, nil)
	_, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)
	item := w.itemsOf(batch.ID)[0]

	missing := pulid.MustNew("shp_")
	_, err = s.FileItem(t.Context(), &FileItemInput{
		TenantInfo: w.tenant, ItemID: item.ID, TargetType: "shipment", TargetID: missing, Version: item.Version,
	})
	require.Error(t, err, "a record that does not exist cannot be filed onto")

	carrierID := w.addRecord("carrier")
	w.denied["carrier:read"] = true
	_, err = s.FileItem(t.Context(), &FileItemInput{
		TenantInfo: w.tenant, ItemID: item.ID, TargetType: "carrier", TargetID: carrierID, Version: item.Version,
	})
	assert.True(t, errortypes.IsAuthorizationError(err))

	_, err = s.FileItem(t.Context(), &FileItemInput{
		TenantInfo: w.tenant, ItemID: item.ID, TargetType: "invoice", TargetID: carrierID, Version: item.Version,
	})
	require.Error(t, err, "invoices do not take captures")

	_, err = s.FileItem(t.Context(), &FileItemInput{
		TenantInfo: w.tenant, ItemID: item.ID, TargetType: "shipment", TargetID: missing, Version: item.Version + 5,
	})
	assert.True(t, errortypes.IsConflictError(err), "a stale version is refused")
	assert.Equal(t, capture.ItemProposed, w.items[item.ID].Status, "nothing was filed")
}

func TestOwnScopeSeesOnlyOwnBatches(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "o", Source: capture.SourceScan},
		[][]byte{pdfPage(t, 1)}, nil)

	w.ownScope = true
	other := w.tenant
	other.UserID = pulid.MustNew("usr_")

	_, err := s.GetBatch(t.Context(), other, batch.ID)
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = s.GetBatch(t.Context(), w.tenant, batch.ID)
	require.NoError(t, err)

	w.ownScope = false
	_, err = s.GetBatch(t.Context(), other, batch.ID)
	require.NoError(t, err, "organization scope sees everybody's intake")
}

func TestEditItemsSplitsKeepingTheRoute(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)
	principal := principalFor(t, s, tokens)
	shipmentID := w.addRecord("shipment")
	req, err := s.CreateRequest(t.Context(), &CreateRequestInput{
		TenantInfo: w.tenant, DeviceID: tokens.DeviceID, Mode: capture.RequestModeScan,
		TargetType: "shipment", TargetID: shipmentID,
	})
	require.NoError(t, err)

	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "e", Source: capture.SourceScan, RequestID: &req.ID},
		[][]byte{pdfPage(t, 1), pdfPage(t, 2), pdfPage(t, 3)}, nil)
	_, err = s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)

	current := w.batches[batch.ID]
	item := w.itemsOf(batch.ID)[0]
	require.Len(t, item.PageIDs, 3)

	edited, err := s.EditItems(t.Context(), &EditItemsInput{
		TenantInfo: w.tenant,
		BatchID:    batch.ID,
		Version:    current.Version,
		Items: []ItemLayout{
			{PageIDs: item.PageIDs[:1]},
			{PageIDs: []pulid.ID{item.PageIDs[2], item.PageIDs[1]}},
		},
		Rotations: map[pulid.ID]int{item.PageIDs[1]: -90},
	})
	require.NoError(t, err)
	require.Len(t, edited.Items, 2)
	assert.Equal(t, 2, edited.ItemCount)
	for _, split := range edited.Items {
		assert.Equal(t, capture.SuggestionRequest, split.SuggestionSource, "both halves keep the route")
		assert.Equal(t, shipmentID, *split.SuggestedID)
	}
	assert.Equal(t, 270, w.pages[item.PageIDs[1]].Rotation)

	_, err = s.EditItems(t.Context(), &EditItemsInput{
		TenantInfo: w.tenant, BatchID: batch.ID, Version: current.Version,
		Items: []ItemLayout{{PageIDs: item.PageIDs}},
	})
	assert.True(t, errortypes.IsConflictError(err), "an edit against an old version is refused")

	latest := w.batches[batch.ID]
	_, err = s.EditItems(t.Context(), &EditItemsInput{
		TenantInfo: w.tenant, BatchID: batch.ID, Version: latest.Version,
		Items: []ItemLayout{{PageIDs: []pulid.ID{item.PageIDs[0], item.PageIDs[0]}}},
	})
	require.Error(t, err, "a page cannot be in two places")
}

func TestPrintJobLandsOnTheArmedDestination(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)
	principal := principalFor(t, s, tokens)

	first := w.addRecord("shipment")
	second := w.addRecord("shipment")
	armed, err := s.CreateRequest(t.Context(), &CreateRequestInput{
		TenantInfo: w.tenant, DeviceID: tokens.DeviceID, Mode: capture.RequestModePrint,
		TargetType: "shipment", TargetID: first,
	})
	require.NoError(t, err)
	rearmed, err := s.CreateRequest(t.Context(), &CreateRequestInput{
		TenantInfo: w.tenant, DeviceID: tokens.DeviceID, Mode: capture.RequestModePrint,
		TargetType: "shipment", TargetID: second,
	})
	require.NoError(t, err)
	assert.Equal(t, capture.RequestCanceled, w.requests[armed.ID].Status, "arming again disarms the last")

	batch, err := s.OpenBatch(t.Context(), principal, &OpenBatchInput{
		ClientKey: "print-1", Source: capture.SourcePrint, JobName: "Rate con 4471.pdf",
	})
	require.NoError(t, err)
	require.NotNil(t, batch.RequestID)
	assert.Equal(t, rearmed.ID, *batch.RequestID)
	assert.Equal(t, second, *batch.TargetID)

	job, err := s.assembler.Assemble(t.Context(), []services.CaptureAssemblyPage{
		{PDF: pdfPage(t, 1)}, {PDF: pdfPage(t, 2)}, {PDF: pdfPage(t, 3)},
	})
	require.NoError(t, err)

	sealed, err := s.PutPrintJob(t.Context(), principal, &PutPrintJobInput{BatchID: batch.ID, Body: bytes.NewReader(job)})
	require.NoError(t, err)
	assert.Equal(t, capture.BatchSealed, sealed.Status)
	assert.Len(t, w.pagesOf(batch.ID), 3)
	assert.Equal(t, capture.RequestCompleted, w.requests[rearmed.ID].Status)
}

func TestRequestsOnlyDriveYourOwnDevice(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)
	shipmentID := w.addRecord("shipment")

	other := w.tenant
	other.UserID = pulid.MustNew("usr_")
	_, err := s.CreateRequest(t.Context(), &CreateRequestInput{
		TenantInfo: other, DeviceID: tokens.DeviceID, Mode: capture.RequestModeScan,
		TargetType: "shipment", TargetID: shipmentID,
	})
	require.Error(t, err)
	var validation *errortypes.Error
	require.True(t, errors.As(err, &validation))
	assert.Equal(t, "deviceId", validation.Field)
}

func TestRetentionExpiresAndPurges(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	batch := scan(t, s, principal, &OpenBatchInput{ClientKey: "old", Source: capture.SourceScan},
		[][]byte{pdfPage(t, 1)}, nil)
	_, err := s.ProcessBatch(t.Context(), w.tenant, batch.ID, nil)
	require.NoError(t, err)
	require.NotEmpty(t, w.objects)

	w.batches[batch.ID].RetainUntil = timeutils.NowUnix() - 1
	result, err := s.Maintain(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, result.BatchesExpired)
	assert.Equal(t, 1, result.BatchesPurged)
	assert.Empty(t, w.objects, "every page and thumbnail is removed from storage")
	assert.NotContains(t, w.batches, batch.ID)
}

func TestExpiredRequestsCloseAsNotDelivered(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	tokens := pair(t, w, s)
	shipmentID := w.addRecord("shipment")
	req, err := s.CreateRequest(t.Context(), &CreateRequestInput{
		TenantInfo: w.tenant, DeviceID: tokens.DeviceID, Mode: capture.RequestModeScan,
		TargetType: "shipment", TargetID: shipmentID,
	})
	require.NoError(t, err)

	w.requests[req.ID].ExpiresAt = timeutils.NowUnix() - 1
	count, err := s.ExpireRequests(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, capture.RequestExpired, w.requests[req.ID].Status)
	assert.Equal(t, capture.FailureNotDelivered, w.requests[req.ID].FailureCode)
}

func TestListBatchesNarrowsOwnScope(t *testing.T) {
	t.Parallel()

	w := newWorld()
	s := w.service()
	principal := principalFor(t, s, pair(t, w, s))
	scan(t, s, principal, &OpenBatchInput{ClientKey: "a", Source: capture.SourceScan}, [][]byte{pdfPage(t, 1)}, nil)

	w.ownScope = true
	other := w.tenant
	other.UserID = pulid.MustNew("usr_")
	result, err := s.ListBatches(t.Context(), &ListBatchesInput{
		TenantInfo: other, Filter: &pagination.QueryOptions{},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Items)
}
