package fuelpurchaseservice_test

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	mu        sync.Mutex
	cards     map[pulid.ID]*fuelpurchase.FuelCard
	purchases map[pulid.ID]*fuelpurchase.FuelPurchase
	batches   map[pulid.ID]*fuelpurchase.ImportBatch
	rows      map[pulid.ID][]*fuelpurchase.ImportRow
	commits   int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		cards:     map[pulid.ID]*fuelpurchase.FuelCard{},
		purchases: map[pulid.ID]*fuelpurchase.FuelPurchase{},
		batches:   map[pulid.ID]*fuelpurchase.ImportBatch{},
		rows:      map[pulid.ID][]*fuelpurchase.ImportRow{},
	}
}

func sameTenant(orgID, buID pulid.ID, tenant pagination.TenantInfo) bool {
	return orgID == tenant.OrgID && buID == tenant.BuID
}

func (f *fakeRepo) ListCards(
	_ context.Context,
	_ *repositories.ListFuelCardsRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error) {
	out := make([]*fuelpurchase.FuelCard, 0, len(f.cards))
	for _, card := range f.cards {
		out = append(out, card)
	}
	return pagination.NewCursorListResult(out, len(out)+1), nil
}

func (f *fakeRepo) GetCardByID(
	_ context.Context,
	req *repositories.GetFuelCardByIDRequest,
) (*fuelpurchase.FuelCard, error) {
	card, ok := f.cards[req.ID]
	if !ok || !sameTenant(card.OrganizationID, card.BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("FuelCard not found within your organization")
	}
	copied := *card
	return &copied, nil
}

func (f *fakeRepo) GetCardsByIDs(
	_ context.Context,
	req *repositories.GetFuelCardsByIDsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	out := make([]*fuelpurchase.FuelCard, 0, len(req.IDs))
	for _, id := range req.IDs {
		if card, ok := f.cards[id]; ok {
			out = append(out, card)
		}
	}
	return out, nil
}

func (f *fakeRepo) FindCardByLastFour(
	_ context.Context,
	req *repositories.FindFuelCardByLastFourRequest,
) (*fuelpurchase.FuelCard, error) {
	for _, card := range f.cards {
		if !sameTenant(card.OrganizationID, card.BusinessUnitID, req.TenantInfo) {
			continue
		}
		if card.LastFour != req.LastFour || card.IsCancelled() {
			continue
		}
		if req.Provider != "" && card.Provider != req.Provider {
			continue
		}
		copied := *card
		return &copied, nil
	}
	return nil, nil
}

func (f *fakeRepo) ListActiveCards(
	_ context.Context,
	req *repositories.ListActiveFuelCardsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	out := make([]*fuelpurchase.FuelCard, 0, len(f.cards))
	for _, card := range f.cards {
		if sameTenant(card.OrganizationID, card.BusinessUnitID, req.TenantInfo) && card.IsActive() {
			out = append(out, card)
		}
	}
	return out, nil
}

func (f *fakeRepo) CreateCard(
	_ context.Context,
	entity *fuelpurchase.FuelCard,
) (*fuelpurchase.FuelCard, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("fcard_")
	}
	f.cards[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateCard(
	_ context.Context,
	entity *fuelpurchase.FuelCard,
) (*fuelpurchase.FuelCard, error) {
	stored, ok := f.cards[entity.ID]
	if !ok || stored.Version != entity.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"version mismatch",
		)
	}
	entity.Version++
	f.cards[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListPurchases(
	_ context.Context,
	_ *repositories.ListFuelPurchasesRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
	out := make([]*fuelpurchase.FuelPurchase, 0, len(f.purchases))
	for _, purchase := range f.purchases {
		out = append(out, purchase)
	}
	return pagination.NewCursorListResult(out, len(out)+1), nil
}

func (f *fakeRepo) GetPurchaseByID(
	_ context.Context,
	req *repositories.GetFuelPurchaseByIDRequest,
) (*fuelpurchase.FuelPurchase, error) {
	purchase, ok := f.purchases[req.ID]
	if !ok || !sameTenant(purchase.OrganizationID, purchase.BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("FuelPurchase not found within your organization")
	}
	copied := *purchase
	return &copied, nil
}

func (f *fakeRepo) GetPurchasesByIDs(
	_ context.Context,
	req *repositories.GetFuelPurchasesByIDsRequest,
) ([]*fuelpurchase.FuelPurchase, error) {
	out := make([]*fuelpurchase.FuelPurchase, 0, len(req.IDs))
	for _, id := range req.IDs {
		if purchase, ok := f.purchases[id]; ok {
			out = append(out, purchase)
		}
	}
	return out, nil
}

func (f *fakeRepo) FindReferences(
	_ context.Context,
	req *repositories.FindFuelPurchaseReferencesRequest,
) (map[string]pulid.ID, error) {
	wanted := make(map[string]struct{}, len(req.References))
	for _, ref := range req.References {
		wanted[strings.ToUpper(strings.TrimSpace(ref))] = struct{}{}
	}
	found := make(map[string]pulid.ID, len(wanted))
	for _, purchase := range f.purchases {
		if !sameTenant(purchase.OrganizationID, purchase.BusinessUnitID, req.TenantInfo) {
			continue
		}
		if _, ok := wanted[purchase.TransactionReference]; ok {
			found[purchase.TransactionReference] = purchase.ID
		}
	}
	return found, nil
}

func (f *fakeRepo) referenceTaken(entity *fuelpurchase.FuelPurchase) bool {
	if entity.TransactionReference == "" {
		return false
	}
	for _, purchase := range f.purchases {
		if purchase.ID == entity.ID {
			continue
		}
		if purchase.OrganizationID == entity.OrganizationID &&
			purchase.BusinessUnitID == entity.BusinessUnitID &&
			purchase.TransactionReference == entity.TransactionReference {
			return true
		}
	}
	return false
}

func (f *fakeRepo) CreatePurchase(
	_ context.Context,
	entity *fuelpurchase.FuelPurchase,
) (*fuelpurchase.FuelPurchase, error) {
	if f.referenceTaken(entity) {
		return nil, errortypes.NewValidationError(
			"transactionReference",
			errortypes.ErrDuplicate,
			"duplicate",
		)
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("fpur_")
	}
	f.purchases[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdatePurchase(
	_ context.Context,
	entity *fuelpurchase.FuelPurchase,
) (*fuelpurchase.FuelPurchase, error) {
	stored, ok := f.purchases[entity.ID]
	if !ok || stored.Version != entity.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"version mismatch",
		)
	}
	entity.Version++
	f.purchases[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) DeletePurchase(
	_ context.Context,
	req *repositories.DeleteFuelPurchaseRequest,
) error {
	purchase, ok := f.purchases[req.ID]
	if !ok || !sameTenant(purchase.OrganizationID, purchase.BusinessUnitID, req.TenantInfo) {
		return errortypes.NewNotFoundError("FuelPurchase not found within your organization")
	}
	delete(f.purchases, req.ID)
	return nil
}

func (f *fakeRepo) AccumulateFuel(
	_ context.Context,
	_ *repositories.AccumulateFuelRequest,
) ([]*repositories.FuelAccumulationRow, error) {
	return nil, nil
}

func (f *fakeRepo) GetImportBatchByID(
	_ context.Context,
	req *repositories.GetImportBatchByIDRequest,
) (*fuelpurchase.ImportBatch, error) {
	batch, ok := f.batches[req.ID]
	if !ok || !sameTenant(batch.OrganizationID, batch.BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("FuelPurchaseImportBatch not found")
	}
	copied := *batch
	copied.Rows = nil
	if req.IncludeRows {
		rows := f.rows[batch.ID]
		copied.Rows = make([]*fuelpurchase.ImportRow, 0, len(rows))
		for _, row := range rows {
			rowCopy := *row
			copied.Rows = append(copied.Rows, &rowCopy)
		}
	}
	return &copied, nil
}

func (f *fakeRepo) CreateImportBatch(
	_ context.Context,
	entity *fuelpurchase.ImportBatch,
) (*fuelpurchase.ImportBatch, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("fpib_")
	}
	f.batches[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateImportBatch(
	_ context.Context,
	entity *fuelpurchase.ImportBatch,
) (*fuelpurchase.ImportBatch, error) {
	stored, ok := f.batches[entity.ID]
	if !ok || stored.Version != entity.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"version mismatch",
		)
	}
	entity.Version++
	copied := *entity
	copied.Rows = nil
	f.batches[entity.ID] = &copied
	return entity, nil
}

func (f *fakeRepo) ReplaceImportRows(
	_ context.Context,
	batch *fuelpurchase.ImportBatch,
	rows []*fuelpurchase.ImportRow,
) error {
	stored := make([]*fuelpurchase.ImportRow, 0, len(rows))
	for _, row := range rows {
		if row.ID.IsNil() {
			row.ID = pulid.MustNew("fpir_")
		}
		row.ImportBatchID = batch.ID
		row.OrganizationID = batch.OrganizationID
		row.BusinessUnitID = batch.BusinessUnitID
		stored = append(stored, row)
	}
	f.rows[batch.ID] = stored
	return nil
}

func (f *fakeRepo) ListImportRows(
	_ context.Context,
	req *repositories.ListImportRowsRequest,
) (*pagination.CursorListResult[*fuelpurchase.ImportRow], error) {
	rows := f.rows[req.BatchID]
	out := make([]*fuelpurchase.ImportRow, 0, len(rows))
	for _, row := range rows {
		if len(req.Statuses) > 0 {
			matched := false
			for _, status := range req.Statuses {
				if row.Status == status {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, row)
	}
	return pagination.NewCursorListResult(out, len(out)+1), nil
}

func (f *fakeRepo) CommitImport(
	_ context.Context,
	req *repositories.CommitImportRequest,
) (*repositories.CommitImportResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.commits++
	batch := req.Batch
	result := &repositories.CommitImportResult{}
	rowsByID := make(map[pulid.ID]*fuelpurchase.ImportRow, len(f.rows[batch.ID]))
	for _, row := range f.rows[batch.ID] {
		rowsByID[row.ID] = row
	}

	for _, purchase := range req.Purchases {
		purchase.OrganizationID = batch.OrganizationID
		purchase.BusinessUnitID = batch.BusinessUnitID
		purchase.Source = fuelpurchase.PurchaseSourceCardImport
		purchase.ImportBatchID = &batch.ID
		rowID := req.RowIDByReference[purchase.TransactionReference]
		row := rowsByID[rowID]

		if f.referenceTaken(purchase) {
			result.AlreadyImported++
			if row != nil {
				row.Status = fuelpurchase.ImportRowStatusAlreadyImported
			}
			continue
		}
		if purchase.ID.IsNil() {
			purchase.ID = pulid.MustNew("fpur_")
		}
		f.purchases[purchase.ID] = purchase
		result.Committed++
		if row != nil {
			row.Status = fuelpurchase.ImportRowStatusCommitted
			id := purchase.ID
			row.FuelPurchaseID = &id
		}
	}

	stored := f.batches[batch.ID]
	if stored.Version != batch.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"version mismatch",
		)
	}
	committedAt := req.CommittedAt
	stored.Status = fuelpurchase.ImportStatusCommitted
	stored.CommittedAt = &committedAt
	stored.CommittedByID = req.CommittedByID
	stored.CommittedCount = result.Committed
	stored.Version++
	batch.Version = stored.Version

	return result, nil
}

type fakeJurisdictions struct {
	byKey map[string]*ifta.Jurisdiction
}

func newFakeJurisdictions(codes ...string) *fakeJurisdictions {
	f := &fakeJurisdictions{byKey: map[string]*ifta.Jurisdiction{}}
	for _, code := range codes {
		country := "US"
		if code == "ON" || code == "QC" {
			country = "CA"
		}
		f.byKey[country+"_"+code] = &ifta.Jurisdiction{
			ID:           pulid.MustNew("ifj_"),
			CountryCode:  country,
			Code:         code,
			Name:         code,
			IsIftaMember: true,
			Status:       ifta.JurisdictionStatusActive,
		}
	}
	return f
}

func (f *fakeJurisdictions) GetJurisdictionByID(
	_ context.Context,
	id pulid.ID,
) (*ifta.Jurisdiction, error) {
	for _, jurisdiction := range f.byKey {
		if jurisdiction.ID == id {
			return jurisdiction, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Jurisdiction not found")
}

func (f *fakeJurisdictions) GetJurisdictionByCode(
	_ context.Context,
	countryCode, code string,
) (*ifta.Jurisdiction, error) {
	jurisdiction, ok := f.byKey[countryCode+"_"+code]
	if !ok {
		return nil, errortypes.NewNotFoundError("Jurisdiction not found")
	}
	return jurisdiction, nil
}

func (f *fakeJurisdictions) FindJurisdictionsByCodes(
	_ context.Context,
	keys []string,
) (map[string]*ifta.Jurisdiction, error) {
	found := make(map[string]*ifta.Jurisdiction, len(keys))
	for _, key := range keys {
		if jurisdiction, ok := f.byKey[key]; ok {
			found[key] = jurisdiction
		}
	}
	return found, nil
}

type fakeDocuments struct {
	docs    map[pulid.ID]*document.Document
	content map[pulid.ID][]byte
}

func newFakeDocuments() *fakeDocuments {
	return &fakeDocuments{
		docs:    map[pulid.ID]*document.Document{},
		content: map[pulid.ID][]byte{},
	}
}

func (f *fakeDocuments) add(
	tenant pagination.TenantInfo,
	resourceType, resourceID, name string,
	content []byte,
) pulid.ID {
	id := pulid.MustNew("doc_")
	f.docs[id] = &document.Document{
		ID:             id,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		FileName:       name,
		OriginalName:   name,
		FileSize:       int64(len(content)),
		FileType:       "text/csv",
		ResourceType:   resourceType,
		ResourceID:     resourceID,
	}
	f.content[id] = content
	return id
}

func (f *fakeDocuments) Get(
	_ context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	doc, ok := f.docs[req.ID]
	if !ok || !sameTenant(doc.OrganizationID, doc.BusinessUnitID, req.TenantInfo) {
		return nil, errortypes.NewNotFoundError("Document not found")
	}
	return doc, nil
}

func (f *fakeDocuments) GetDownloadContent(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (*services.DocumentContent, error) {
	doc, err := f.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	body := f.content[req.ID]
	return &services.DocumentContent{
		Document:      doc,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}, nil
}

type fakeReferenceChecker struct{}

func (fakeReferenceChecker) CheckReference(
	_ context.Context,
	_ *validationframework.ReferenceRequest,
) (bool, error) {
	return true, nil
}

type auditRecord struct {
	resource  permission.Resource
	operation permission.Operation
}

type harness struct {
	svc       *fuelpurchaseservice.Service
	repo      *fakeRepo
	docs      *fakeDocuments
	tenant    pagination.TenantInfo
	other     pagination.TenantInfo
	userID    pulid.ID
	tractorA  *tractor.Tractor
	tractorB  *tractor.Tractor
	audits    *[]auditRecord
	auditLock *sync.Mutex
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	workerID := pulid.MustNew("wrk_")
	tractorA := &tractor.Tractor{
		ID:                 pulid.MustNew("trac_"),
		OrganizationID:     tenant.OrgID,
		BusinessUnitID:     tenant.BuID,
		Code:               "TRC-001",
		LicensePlateNumber: "IL-1001",
		PrimaryWorkerID:    workerID,
	}
	tractorB := &tractor.Tractor{
		ID:                 pulid.MustNew("trac_"),
		OrganizationID:     tenant.OrgID,
		BusinessUnitID:     tenant.BuID,
		Code:               "TRC-002",
		LicensePlateNumber: "IL-1002",
		PrimaryWorkerID:    workerID,
	}
	byCode := map[string]*tractor.Tractor{
		tractorA.Code: tractorA,
		tractorB.Code: tractorB,
	}

	tractorRepo := mocks.NewMockTractorRepository(t)
	tractorRepo.EXPECT().
		GetByCodes(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req repositories.GetTractorsByCodesRequest,
		) (map[string]*tractor.Tractor, error) {
			found := make(map[string]*tractor.Tractor, len(req.Codes))
			if req.TenantInfo != tenant {
				return found, nil
			}
			for _, code := range req.Codes {
				if entity, ok := byCode[strings.ToUpper(strings.TrimSpace(code))]; ok {
					found[entity.Code] = entity
				}
			}
			for _, plate := range req.LicensePlates {
				for _, entity := range byCode {
					if strings.EqualFold(entity.LicensePlateNumber, plate) {
						found[entity.Code] = entity
					}
				}
			}
			return found, nil
		}).
		Maybe()
	tractorRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req repositories.GetTractorByIDRequest,
		) (*tractor.Tractor, error) {
			for _, entity := range byCode {
				if entity.ID == req.ID && req.TenantInfo == tenant {
					return entity, nil
				}
			}
			return nil, errortypes.NewNotFoundError("Tractor not found")
		}).
		Maybe()

	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req repositories.GetWorkerByIDRequest,
		) (*worker.Worker, error) {
			if req.ID == workerID && req.TenantInfo == tenant {
				return &worker.Worker{
					ID:             workerID,
					OrganizationID: tenant.OrgID,
					BusinessUnitID: tenant.BuID,
				}, nil
			}
			return nil, errortypes.NewNotFoundError("Worker not found")
		}).
		Maybe()

	audits := make([]auditRecord, 0, 8)
	auditLock := &sync.Mutex{}
	auditService := mocks.NewMockAuditService(t)
	auditService.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		RunAndReturn(func(params *services.LogActionParams, _ ...services.LogOption) error {
			auditLock.Lock()
			defer auditLock.Unlock()
			audits = append(audits, auditRecord{
				resource:  params.Resource,
				operation: params.Operation,
			})
			return nil
		}).
		Maybe()

	repo := newFakeRepo()
	docs := newFakeDocuments()
	now := int64(1_784_000_000)

	svc := fuelpurchaseservice.NewWithDeps(fuelpurchaseservice.Deps{
		Repo:          repo,
		Jurisdictions: newFakeJurisdictions("TX", "OK", "ON"),
		TractorRepo:   tractorRepo,
		WorkerRepo:    workerRepo,
		Documents:     docs,
		AuditService:  auditService,
		Validator:     fuelpurchaseservice.NewValidatorWithDeps(fakeReferenceChecker{}, repo),
		Now:           func() int64 { return now },
	})

	return &harness{
		svc:       svc,
		repo:      repo,
		docs:      docs,
		tenant:    tenant,
		other:     other,
		userID:    pulid.MustNew("usr_"),
		tractorA:  tractorA,
		tractorB:  tractorB,
		audits:    &audits,
		auditLock: auditLock,
	}
}

func (h *harness) auditOps(resource permission.Resource) []permission.Operation {
	h.auditLock.Lock()
	defer h.auditLock.Unlock()
	ops := make([]permission.Operation, 0, len(*h.audits))
	for _, record := range *h.audits {
		if record.resource == resource {
			ops = append(ops, record.operation)
		}
	}
	return ops
}

func (h *harness) newCard(lastFour string) *fuelpurchase.FuelCard {
	return &fuelpurchase.FuelCard{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Provider:       fuelpurchase.CardProviderComdata,
		LastFour:       lastFour,
		Label:          "Card " + lastFour,
	}
}

const statementHeader = "Trans Date,Unit,Card Number,State,Product,Quantity,Unit Price," +
	"Total Amount,Trans ID\n"

func (h *harness) createImport(t *testing.T) *fuelpurchase.ImportBatch {
	t.Helper()

	batch, err := h.svc.CreateImport(t.Context(), &fuelpurchaseservice.CreateImportRequest{
		TenantInfo:      h.tenant,
		Provider:        fuelpurchase.CardProviderComdata,
		DefaultCurrency: "usd",
		UserID:          h.userID,
	})
	require.NoError(t, err)
	require.Equal(t, fuelpurchase.ImportStatusPending, batch.Status)
	require.Equal(t, "USD", batch.DefaultCurrency)

	return batch
}

func (h *harness) stage(
	t *testing.T,
	batch *fuelpurchase.ImportBatch,
	csv string,
) *fuelpurchase.ImportBatch {
	t.Helper()

	docID := h.docs.add(
		h.tenant,
		fuelpurchaseservice.ImportDocumentResourceType,
		batch.ID.String(),
		"statement.csv",
		[]byte(csv),
	)
	staged, err := h.svc.Stage(t.Context(), &fuelpurchaseservice.StageRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		DocumentID: docID,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	return staged
}

func rowsByNumber(rows []*fuelpurchase.ImportRow) map[int]*fuelpurchase.ImportRow {
	out := make(map[int]*fuelpurchase.ImportRow, len(rows))
	for _, row := range rows {
		out[row.RowNumber] = row
	}
	return out
}

func TestImportCreateStageCommit(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	staged := h.stage(t, batch, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n"+
		"07/15/2026,TRC-002,5678,OK,ULSD,80,3.799,303.92,T-2\n")

	require.Equal(t, fuelpurchase.ImportStatusParsed, staged.Status)
	require.NotNil(t, staged.Summary)
	assert.Equal(t, 2, staged.Summary.RowCount)
	assert.Equal(t, 2, staged.Summary.NewCount)
	assert.Equal(t, 0, staged.Summary.ErrorCount)
	assert.Equal(t, "180.000", staged.Summary.TotalGallons)
	assert.Equal(t, int64(69382), staged.Summary.TotalAmountMinor)
	assert.Equal(t, "180.000", staged.Summary.ByFuelType["Diesel"])
	assert.Equal(t, "100.000", staged.Summary.ByJurisdiction["US_TX"])
	assert.Equal(t, fuelpurchase.SourceFormatCSV, staged.SourceFormat)
	assert.Equal(t, "statement.csv", staged.FileName)
	assert.NotNil(t, staged.DocumentID)
	assert.NotNil(t, staged.StagedAt)
	assert.NotEmpty(t, staged.Mapping)

	rows := rowsByNumber(h.repo.rows[batch.ID])
	require.Len(t, rows, 2)
	assert.Equal(t, fuelpurchase.ImportRowStatusNew, rows[2].Status)
	require.NotNil(t, rows[2].ResolvedTractorID)
	assert.Equal(t, h.tractorA.ID, *rows[2].ResolvedTractorID)
	require.NotNil(t, rows[2].Parsed)
	assert.Equal(t, h.tractorA.ID, rows[2].Parsed.TractorID)
	require.NotNil(t, rows[2].Parsed.WorkerID)
	assert.Equal(t, h.tractorA.PrimaryWorkerID, *rows[2].Parsed.WorkerID)
	assert.Equal(t, "T-1", rows[2].TransactionReference)

	committed, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.ImportStatusCommitted, committed.Status)
	assert.Equal(t, 2, committed.CommittedCount)
	assert.Equal(t, h.userID, committed.CommittedByID)
	require.NotNil(t, committed.CommittedAt)
	assert.Len(t, h.repo.purchases, 2)

	for _, purchase := range h.repo.purchases {
		assert.Equal(t, fuelpurchase.PurchaseSourceCardImport, purchase.Source)
		require.NotNil(t, purchase.ImportBatchID)
		assert.Equal(t, batch.ID, *purchase.ImportBatchID)
		assert.Equal(t, h.userID, purchase.CreatedByID)
		assert.True(t, purchase.TaxPaid)
	}

	rows = rowsByNumber(h.repo.rows[batch.ID])
	assert.Equal(t, fuelpurchase.ImportRowStatusCommitted, rows[2].Status)
	assert.NotNil(t, rows[2].FuelPurchaseID)

	ops := h.auditOps(permission.ResourceFuelPurchaseImport)
	assert.Equal(t, []permission.Operation{
		permission.OpCreate,
		permission.OpImport,
		permission.OpImport,
	}, ops)
}

func TestStageRefusesMismatchedDocument(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	otherBatch := h.createImport(t)

	wrongResource := h.docs.add(
		h.tenant,
		fuelpurchaseservice.ImportDocumentResourceType,
		otherBatch.ID.String(),
		"statement.csv",
		[]byte(statementHeader),
	)
	_, err := h.svc.Stage(t.Context(), &fuelpurchaseservice.StageRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		DocumentID: wrongResource,
		UserID:     h.userID,
	})
	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "documentId", fieldErr.Field)

	wrongType := h.docs.add(
		h.tenant,
		"shipment",
		batch.ID.String(),
		"statement.csv",
		[]byte(statementHeader),
	)
	_, err = h.svc.Stage(t.Context(), &fuelpurchaseservice.StageRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		DocumentID: wrongType,
		UserID:     h.userID,
	})
	require.Error(t, err)

	otherTenantDoc := h.docs.add(
		h.other,
		fuelpurchaseservice.ImportDocumentResourceType,
		batch.ID.String(),
		"statement.csv",
		[]byte(statementHeader),
	)
	_, err = h.svc.Stage(t.Context(), &fuelpurchaseservice.StageRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		DocumentID: otherTenantDoc,
		UserID:     h.userID,
	})
	require.Error(t, err)

	assert.Equal(t, fuelpurchase.ImportStatusPending, h.repo.batches[batch.ID].Status)
}

func TestStageOnCommittedBatchRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	staged := h.stage(t, batch, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n")
	_, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	docID := h.docs.add(
		h.tenant,
		fuelpurchaseservice.ImportDocumentResourceType,
		batch.ID.String(),
		"statement.csv",
		[]byte(statementHeader),
	)
	_, err = h.svc.Stage(t.Context(), &fuelpurchaseservice.StageRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		DocumentID: docID,
		UserID:     h.userID,
	})
	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "status", fieldErr.Field)
}

func TestStageDedupesAgainstFileAndDatabase(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	first := h.createImport(t)
	staged := h.stage(t, first, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n")
	_, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    first.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	second := h.createImport(t)
	staged = h.stage(t, second, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n"+
		"07/15/2026,TRC-002,5678,OK,ULSD,80,3.799,303.92,T-2\n"+
		"07/15/2026,TRC-002,5678,OK,ULSD,80,3.799,303.92,t-2\n"+
		"07/16/2026,TRC-002,5678,OK,ULSD,50,3.799,189.95,\n"+
		"07/16/2026,TRC-002,5678,OK,ULSD,50,3.799,189.95,\n")

	require.Equal(t, fuelpurchase.ImportStatusParsed, staged.Status)
	assert.Equal(t, 5, staged.Summary.RowCount)
	assert.Equal(t, 2, staged.Summary.NewCount)
	assert.Equal(t, 2, staged.Summary.DuplicateInFileCount)
	assert.Equal(t, 1, staged.Summary.AlreadyImportedCount)

	rows := rowsByNumber(h.repo.rows[second.ID])
	assert.Equal(t, fuelpurchase.ImportRowStatusAlreadyImported, rows[2].Status)
	assert.NotNil(t, rows[2].FuelPurchaseID)
	assert.Equal(t, fuelpurchase.ImportRowStatusNew, rows[3].Status)
	assert.Equal(t, fuelpurchase.ImportRowStatusDuplicateInFile, rows[4].Status)
	assert.Equal(t, fuelpurchase.ImportRowStatusNew, rows[5].Status)
	assert.True(t, strings.HasPrefix(rows[5].TransactionReference, "SYNTH:"))
	assert.Equal(t, fuelpurchase.ImportRowStatusDuplicateInFile, rows[6].Status)

	committed, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    second.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, committed.CommittedCount)
	assert.Len(t, h.repo.purchases, 3)

	third := h.createImport(t)
	staged = h.stage(t, third, statementHeader+
		"07/16/2026,TRC-002,5678,OK,ULSD,50,3.799,189.95,\n")
	assert.Equal(t, 1, staged.Summary.AlreadyImportedCount)
	assert.Equal(t, 0, staged.Summary.NewCount)

	_, err = h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    third.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)
	assert.Equal(t, 2, h.repo.commits)
}

func TestCommitTwiceRefused(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	staged := h.stage(t, batch, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n")

	committed, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    committed.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "status", fieldErr.Field)

	_, err = h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "version", fieldErr.Field)

	assert.Len(t, h.repo.purchases, 1)
	assert.Equal(t, 1, h.repo.commits)
}

func TestDiscardGuards(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)

	_, err := h.svc.Discard(t.Context(), &fuelpurchaseservice.DiscardRequest{
		TenantInfo: h.other,
		BatchID:    batch.ID,
		Version:    batch.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)

	_, err = h.svc.Discard(t.Context(), &fuelpurchaseservice.DiscardRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    batch.Version + 5,
		UserID:     h.userID,
	})
	require.Error(t, err)

	discarded, err := h.svc.Discard(t.Context(), &fuelpurchaseservice.DiscardRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    batch.Version,
		Reason:     "Wrong month",
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.ImportStatusDiscarded, discarded.Status)

	_, err = h.svc.Discard(t.Context(), &fuelpurchaseservice.DiscardRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    discarded.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)

	committedBatch := h.createImport(t)
	staged := h.stage(t, committedBatch, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n")
	committed, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    committedBatch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	_, err = h.svc.Discard(t.Context(), &fuelpurchaseservice.DiscardRequest{
		TenantInfo: h.tenant,
		BatchID:    committedBatch.ID,
		Version:    committed.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)

	assert.Contains(t, h.auditOps(permission.ResourceFuelPurchaseImport), permission.OpCancel)
}

func TestStageSizeCap(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	docID := h.docs.add(
		h.tenant,
		fuelpurchaseservice.ImportDocumentResourceType,
		batch.ID.String(),
		"statement.csv",
		[]byte(statementHeader),
	)
	h.docs.docs[docID].FileSize = (32 << 20) + 1

	_, err := h.svc.Stage(t.Context(), &fuelpurchaseservice.StageRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		DocumentID: docID,
		UserID:     h.userID,
	})
	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "documentId", fieldErr.Field)
	assert.Contains(t, fieldErr.Message, "too large")

	unsupported := h.docs.add(
		h.tenant,
		fuelpurchaseservice.ImportDocumentResourceType,
		batch.ID.String(),
		"statement.pdf",
		[]byte("%PDF"),
	)
	_, err = h.svc.Stage(t.Context(), &fuelpurchaseservice.StageRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		DocumentID: unsupported,
		UserID:     h.userID,
	})
	require.Error(t, err)
}

func TestStageMarksUnmatchedTractorAsError(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	staged := h.stage(t, batch, statementHeader+
		"07/14/2026,TRC-999,1234,TX,ULSD,100,3.899,389.90,T-1\n"+
		"07/14/2026,,9999,TX,ULSD,100,3.899,389.90,T-2\n"+
		"07/14/2026,TRC-001,1234,ZZ,ULSD,100,3.899,389.90,T-3\n"+
		"07/14/2026,TRC-001,1234,QC,ULSD,100,3.899,389.90,T-4\n"+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,-389.90,T-5\n")

	require.Equal(t, fuelpurchase.ImportStatusParsed, staged.Status)
	assert.Equal(t, 5, staged.Summary.RowCount)
	assert.Equal(t, 5, staged.Summary.ErrorCount)
	assert.Equal(t, 0, staged.Summary.NewCount)

	rows := rowsByNumber(h.repo.rows[batch.ID])
	assert.Equal(t, fuelpurchase.ImportRowStatusError, rows[2].Status)
	assert.Contains(t, rows[2].Error, "TRC-999")
	assert.Equal(t, fuelpurchase.ImportRowStatusError, rows[3].Status)
	assert.Contains(t, rows[3].Error, "9999")
	assert.Equal(t, fuelpurchase.ImportRowStatusError, rows[4].Status)
	assert.Equal(t, fuelpurchase.ImportRowStatusError, rows[5].Status)
	assert.Contains(t, rows[5].Error, "QC")
	assert.Equal(t, fuelpurchase.ImportRowStatusError, rows[6].Status)
	assert.Contains(t, rows[6].Error, "reversal")

	_, err := h.svc.Commit(t.Context(), &fuelpurchaseservice.CommitRequest{
		TenantInfo: h.tenant,
		BatchID:    batch.ID,
		Version:    staged.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)
	assert.Empty(t, h.repo.purchases)
}

func TestStageResolvesTractorThroughCardAndDefaultCard(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	assigned := h.newCard("1234")
	assigned.AssignedTractorID = &h.tractorB.ID
	created, err := h.svc.CreateCard(t.Context(), assigned, h.userID)
	require.NoError(t, err)

	fallback := h.newCard("0000")
	fallback.AssignedTractorID = &h.tractorA.ID
	defaultCard, err := h.svc.CreateCard(t.Context(), fallback, h.userID)
	require.NoError(t, err)

	batch, err := h.svc.CreateImport(t.Context(), &fuelpurchaseservice.CreateImportRequest{
		TenantInfo:        h.tenant,
		Provider:          fuelpurchase.CardProviderComdata,
		DefaultFuelCardID: defaultCard.ID,
		UserID:            h.userID,
	})
	require.NoError(t, err)

	staged := h.stage(t, batch, statementHeader+
		"07/14/2026,,1234,TX,ULSD,100,3.899,389.90,T-1\n"+
		"07/14/2026,,4321,TX,ULSD,100,3.899,389.90,T-2\n")
	assert.Equal(t, 2, staged.Summary.NewCount)

	rows := rowsByNumber(h.repo.rows[batch.ID])
	require.NotNil(t, rows[2].ResolvedTractorID)
	assert.Equal(t, h.tractorB.ID, *rows[2].ResolvedTractorID)
	require.NotNil(t, rows[2].ResolvedFuelCardID)
	assert.Equal(t, created.ID, *rows[2].ResolvedFuelCardID)
	assert.NotEmpty(t, rows[2].ResolutionNotes)

	require.NotNil(t, rows[3].ResolvedTractorID)
	assert.Equal(t, h.tractorA.ID, *rows[3].ResolvedTractorID)
	require.NotNil(t, rows[3].ResolvedFuelCardID)
	assert.Equal(t, defaultCard.ID, *rows[3].ResolvedFuelCardID)
	assert.Equal(t, "4321", rows[3].Parsed.CardLastFour)
}

func TestStageFailsOnUnreadableSheet(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	batch := h.createImport(t)
	failed := h.stage(t, batch, "Just a title\n")
	assert.Equal(t, fuelpurchase.ImportStatusFailed, failed.Status)
	assert.NotEmpty(t, failed.Error)

	failed = h.stage(t, batch, "Alpha,Beta\n1,2\n")
	assert.Equal(t, fuelpurchase.ImportStatusFailed, failed.Status)
	assert.Contains(t, failed.Error, "transaction date")
	assert.ElementsMatch(t, []string{"Alpha", "Beta"}, failed.UnmappedHeaders)

	recovered := h.stage(t, batch, statementHeader+
		"07/14/2026,TRC-001,1234,TX,ULSD,100,3.899,389.90,T-1\n")
	assert.Equal(t, fuelpurchase.ImportStatusParsed, recovered.Status)
	assert.Empty(t, recovered.Error)
}

func TestCardCancelRequiresReason(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	card, err := h.svc.CreateCard(t.Context(), h.newCard("1234"), h.userID)
	require.NoError(t, err)

	_, err = h.svc.CancelCard(t.Context(), &fuelpurchaseservice.CancelCardRequest{
		TenantInfo: h.tenant,
		ID:         card.ID,
		Version:    card.Version,
		Reason:     "lost",
		UserID:     h.userID,
	})
	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "reason", fieldErr.Field)

	cancelled, err := h.svc.CancelCard(t.Context(), &fuelpurchaseservice.CancelCardRequest{
		TenantInfo: h.tenant,
		ID:         card.ID,
		Version:    card.Version,
		Reason:     "Card reported lost by the driver",
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.CardStatusCancelled, cancelled.Status)
	require.NotNil(t, cancelled.CancelledAt)
	assert.Equal(t, int64(1_784_000_000), *cancelled.CancelledAt)

	_, err = h.svc.CancelCard(t.Context(), &fuelpurchaseservice.CancelCardRequest{
		TenantInfo: h.tenant,
		ID:         card.ID,
		Version:    cancelled.Version,
		Reason:     "Card reported lost by the driver",
		UserID:     h.userID,
	})
	require.Error(t, err)

	replacement, err := h.svc.CreateCard(t.Context(), h.newCard("1234"), h.userID)
	require.NoError(t, err)
	assert.NotEqual(t, card.ID, replacement.ID)

	assert.Contains(t, h.auditOps(permission.ResourceFuelCard), permission.OpCancel)
}

func TestCardUniquenessAndTransitions(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	card, err := h.svc.CreateCard(t.Context(), h.newCard("1234"), h.userID)
	require.NoError(t, err)

	_, err = h.svc.CreateCard(t.Context(), h.newCard("1234"), h.userID)
	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "lastFour", multiErr.Errors[0].Field)

	suspended := *card
	suspended.Status = fuelpurchase.CardStatusSuspended
	updated, err := h.svc.UpdateCard(t.Context(), &suspended, h.userID)
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.CardStatusSuspended, updated.Status)

	direct := *updated
	direct.Status = fuelpurchase.CardStatusCancelled
	_, err = h.svc.UpdateCard(t.Context(), &direct, h.userID)
	require.Error(t, err)

	reactivated := *updated
	reactivated.Status = fuelpurchase.CardStatusActive
	reactivated.Label = "Reactivated"
	updated, err = h.svc.UpdateCard(t.Context(), &reactivated, h.userID)
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.CardStatusActive, updated.Status)
	assert.Equal(t, "Reactivated", updated.Label)

	_, err = h.svc.CancelCard(t.Context(), &fuelpurchaseservice.CancelCardRequest{
		TenantInfo: h.tenant,
		ID:         updated.ID,
		Version:    updated.Version,
		Reason:     "Replaced by a new programme",
		UserID:     h.userID,
	})
	require.NoError(t, err)

	edit := *h.repo.cards[updated.ID]
	edit.Label = "Nope"
	_, err = h.svc.UpdateCard(t.Context(), &edit, h.userID)
	require.Error(t, err)
}

func (h *harness) newPurchase() *fuelpurchase.FuelPurchase {
	return &fuelpurchase.FuelPurchase{
		TractorID:        h.tractorA.ID,
		PurchasedAt:      1_783_900_000,
		Vendor:           "Pilot",
		FuelType:         domaintypes.IFTAFuelTypeDiesel,
		Quantity:         decimal.RequireFromString("100"),
		QuantityUnit:     fuelpurchase.QuantityUnitGallon,
		TotalAmountMinor: 38990,
		TaxPaid:          true,
	}
}

func TestCreatePurchaseResolvesJurisdictionAndCard(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	cardInput := h.newCard("1234")
	workerID := h.tractorA.PrimaryWorkerID
	cardInput.AssignedWorkerID = &workerID
	card, err := h.svc.CreateCard(t.Context(), cardInput, h.userID)
	require.NoError(t, err)

	purchase := h.newPurchase()
	purchase.CardLastFour = "1234"
	purchase.TransactionReference = " abc-1 "
	created, err := h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         purchase,
		JurisdictionCode: "us-tx",
		UserID:           h.userID,
	})
	require.NoError(t, err)

	assert.False(t, created.JurisdictionID.IsNil())
	require.NotNil(t, created.FuelCardID)
	assert.Equal(t, card.ID, *created.FuelCardID)
	require.NotNil(t, created.WorkerID)
	assert.Equal(t, workerID, *created.WorkerID)
	assert.Equal(t, "ABC-1", created.TransactionReference)
	assert.Equal(t, fuelpurchase.PurchaseSourceManual, created.Source)
	assert.Equal(t, "USD", created.CurrencyCode)
	assert.Equal(t, h.userID, created.CreatedByID)
	assert.Equal(t, "100.000", created.Gallons.StringFixed(3))

	_, err = h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         h.newPurchase(),
		JurisdictionCode: "Narnia",
		UserID:           h.userID,
	})
	require.Error(t, err)

	_, err = h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         h.newPurchase(),
		JurisdictionCode: "QC",
		UserID:           h.userID,
	})
	require.Error(t, err)

	foreign := h.newPurchase()
	foreign.TractorID = pulid.MustNew("trac_")
	_, err = h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         foreign,
		JurisdictionCode: "TX",
		UserID:           h.userID,
	})
	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "tractorId", fieldErr.Field)

	duplicate := h.newPurchase()
	duplicate.TransactionReference = "ABC-1"
	_, err = h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         duplicate,
		JurisdictionCode: "TX",
		UserID:           h.userID,
	})
	require.Error(t, err)
}

func TestUpdatePurchaseForbidsSourceChange(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.CreatePurchase(t.Context(), &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo:       h.tenant,
		Purchase:         h.newPurchase(),
		JurisdictionCode: "TX",
		UserID:           h.userID,
	})
	require.NoError(t, err)

	flipped := *created
	flipped.Source = fuelpurchase.PurchaseSourceCardImport
	batchID := pulid.MustNew("fpib_")
	flipped.ImportBatchID = &batchID
	_, err = h.svc.UpdatePurchase(t.Context(), &fuelpurchaseservice.UpdatePurchaseRequest{
		TenantInfo: h.tenant,
		Purchase:   &flipped,
		UserID:     h.userID,
	})
	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "source", fieldErr.Field)

	edited := *created
	edited.Source = ""
	edited.Quantity = decimal.RequireFromString("200")
	edited.QuantityUnit = fuelpurchase.QuantityUnitLitre
	updated, err := h.svc.UpdatePurchase(t.Context(), &fuelpurchaseservice.UpdatePurchaseRequest{
		TenantInfo: h.tenant,
		Purchase:   &edited,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, fuelpurchase.PurchaseSourceManual, updated.Source)
	assert.Equal(t, "52.834", updated.Gallons.StringFixed(3))
	assert.Equal(t, created.Version+1, updated.Version)

	err = h.svc.DeletePurchase(t.Context(), &fuelpurchaseservice.DeletePurchaseRequest{
		TenantInfo: h.tenant,
		ID:         updated.ID,
		Version:    created.Version,
		UserID:     h.userID,
	})
	require.Error(t, err)

	err = h.svc.DeletePurchase(t.Context(), &fuelpurchaseservice.DeletePurchaseRequest{
		TenantInfo: h.tenant,
		ID:         updated.ID,
		Version:    updated.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Empty(t, h.repo.purchases)

	ops := h.auditOps(permission.ResourceFuelPurchase)
	sort.Slice(ops, func(i, j int) bool { return ops[i] < ops[j] })
	assert.Equal(t, []permission.Operation{
		permission.OpCreate,
		permission.OpDelete,
		permission.OpUpdate,
	}, ops)
}

func TestTemplate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	fileName, content := h.svc.Template(fuelpurchase.CardProviderEFS)
	assert.Equal(t, "fuel-purchases-efs-template.csv", fileName)
	assert.True(t, strings.HasPrefix(content, "Transaction Date,"))

	fileName, _ = h.svc.Template("bogus")
	assert.Equal(t, "fuel-purchases-other-template.csv", fileName)
}
