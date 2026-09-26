package aitrainingservice

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	testOrg = pulid.ID("org_test")
	testBU  = pulid.ID("bu_test")
	testNow = int64(1_790_000_000)
)

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: testOrg, BuID: testBU}
}

type exportStore struct {
	repositories.AITrainingExportRepository
	items          map[pulid.ID]*aitraining.TrainingExport
	consent        repositories.TrainingConsent
	consentAfter   *repositories.TrainingConsent
	consentReads   int
	consentChanges int
	people         []string
	onStatusRead   func(entity *aitraining.TrainingExport)
}

func newExportStore() *exportStore {
	return &exportStore{
		items: map[pulid.ID]*aitraining.TrainingExport{},
		consent: repositories.TrainingConsent{
			OrganizationID: testOrg,
			BusinessUnitID: testBU,
			Granted:        true,
			GrantedAt:      testNow - 1000,
		},
		people: []string{"Dana Whitfield"},
	}
}

func (f *exportStore) Create(
	_ context.Context,
	entity *aitraining.TrainingExport,
) (*aitraining.TrainingExport, error) {
	for _, item := range f.items {
		if item.Status.IsActive() {
			return nil, errortypes.NewBusinessError("A training export is already running")
		}
	}
	entity.ID = pulid.MustNew("aitx_")
	entity.CreatedAt = testNow
	stored := *entity
	f.items[entity.ID] = &stored
	return entity, nil
}

func (f *exportStore) GetByID(_ context.Context, id pulid.ID) (*aitraining.TrainingExport, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Training export not found")
	}
	if f.onStatusRead != nil {
		f.onStatusRead(item)
	}
	clone := *item
	clone.Parts = slices.Clone(item.Parts)
	clone.Dropped = maps.Clone(item.Dropped)
	clone.Progress = slices.Clone(item.Progress)
	return &clone, nil
}

func (f *exportStore) Update(
	_ context.Context,
	entity *aitraining.TrainingExport,
) (*aitraining.TrainingExport, error) {
	current, ok := f.items[entity.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Training export not found")
	}
	if current.Version != entity.Version {
		return nil, errors.New("version conflict")
	}
	entity.Version++
	stored := *entity
	f.items[entity.ID] = &stored
	return entity, nil
}

func (f *exportStore) GetConsent(
	_ context.Context,
	_ pagination.TenantInfo,
) (repositories.TrainingConsent, error) {
	f.consentReads++
	if f.consentReads > 1 && f.consentAfter != nil {
		return *f.consentAfter, nil
	}
	return f.consent, nil
}

func (f *exportStore) ListConsentingOrganizations(
	context.Context,
	repositories.ListConsentingOrganizationsRequest,
) ([]repositories.TrainingConsent, error) {
	return []repositories.TrainingConsent{f.consent}, nil
}

func (f *exportStore) ListOrganizationPeople(
	context.Context,
	pagination.TenantInfo,
	int,
) ([]string, error) {
	return f.people, nil
}

type recordStore struct {
	repositories.AITrainingRecordRepository
	records map[pulid.ID][]*aitraining.TrainingExportRecord
	deletes int
}

func newRecordStore() *recordStore {
	return &recordStore{records: map[pulid.ID][]*aitraining.TrainingExportRecord{}}
}

func (f *recordStore) ReplaceForOrganization(
	_ context.Context,
	req *repositories.ReplaceAITrainingRecordsRequest,
) error {
	f.records[req.ExportID] = slices.Clone(req.Records)
	return nil
}

func (f *recordStore) DeleteForOrganization(context.Context, pulid.ID, pagination.TenantInfo) error {
	f.deletes++
	return nil
}

type correctionStore struct {
	repositories.AICorrectionRepository
	items []*aicorrection.Correction
	calls int
}

func (f *correctionStore) ListForTraining(
	_ context.Context,
	req *repositories.ListAICorrectionsForTrainingRequest,
) ([]*aicorrection.Correction, error) {
	f.calls++
	out := make([]*aicorrection.Correction, 0, req.Limit)
	for _, item := range f.items {
		if req.AfterID.IsNotNil() && (item.CapturedAt < req.AfterCapturedAt ||
			(item.CapturedAt == req.AfterCapturedAt && item.ID <= req.AfterID)) {
			continue
		}
		out = append(out, item)
		if len(out) == req.Limit {
			break
		}
	}
	return out, nil
}

type contentStore struct {
	repositories.DocumentContentRepository
	pages map[pulid.ID][]*documentcontent.Page
}

func (f *contentStore) ListPagesByDocumentID(
	_ context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) ([]*documentcontent.Page, error) {
	return f.pages[documentID], nil
}

type documentStore struct {
	repositories.DocumentRepository
	docs map[pulid.ID]*document.Document
}

func (f *documentStore) GetByID(
	_ context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	doc, ok := f.docs[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Document not found")
	}
	return doc, nil
}

type organizationStore struct {
	repositories.OrganizationRepository
}

func (organizationStore) GetByID(
	context.Context,
	repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{
		ID:           testOrg,
		Name:         "Northstar Hauling",
		ScacCode:     "NSHL",
		DOTNumber:    "2931177",
		AddressLine1: "3100 Commerce Street",
		City:         "Irving",
		PostalCode:   "75062",
		BusinessUnit: &tenant.BusinessUnit{Name: "Northstar Group"},
	}, nil
}

type objectStore struct {
	storage.Client
	mu      sync.Mutex
	objects map[string][]byte
	deleted []string
}

func newObjectStore() *objectStore {
	return &objectStore{objects: map[string][]byte{}}
}

func (f *objectStore) Upload(_ context.Context, params *storage.UploadParams) (*storage.FileInfo, error) {
	body, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[params.Key] = body
	return &storage.FileInfo{Key: params.Key, Size: int64(len(body))}, nil
}

func (f *objectStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	f.deleted = append(f.deleted, key)
	return nil
}

func (f *objectStore) lines(key string) [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return bytes.Split(bytes.TrimSpace(f.objects[key]), []byte("\n"))
}

type runnerFixture struct {
	runner      *Runner
	exports     *exportStore
	records     *recordStore
	corrections *correctionStore
	contents    *contentStore
	documents   *documentStore
	objects     *objectStore
	export      *aitraining.TrainingExport
}

func newRunnerFixture() *runnerFixture {
	f := &runnerFixture{
		exports:     newExportStore(),
		records:     newRecordStore(),
		corrections: &correctionStore{},
		contents:    &contentStore{pages: map[pulid.ID][]*documentcontent.Page{}},
		documents:   &documentStore{docs: map[pulid.ID]*document.Document{}},
		objects:     newObjectStore(),
	}
	f.runner = &Runner{
		l:             zap.NewNop(),
		exports:       f.exports,
		records:       f.records,
		corrections:   f.corrections,
		contents:      f.contents,
		documents:     f.documents,
		organizations: organizationStore{},
		storage:       f.objects,
		now:           func() int64 { return testNow },
		seed:          func() ([32]byte, error) { return [32]byte{7}, nil },
		exampleID:     sequentialIDs(),
	}
	f.export = &aitraining.TrainingExport{
		ID:                 pulid.MustNew("aitx_"),
		Task:               aicorrection.TaskShipmentDraftExtraction,
		Status:             aitraining.ExportStatusRunning,
		Format:             aitraining.ExampleFormat,
		CapturedFrom:       testNow - 10_000,
		CapturedTo:         testNow,
		MaxPerOrganization: aitraining.DefaultMaxPerOrganization,
		ValidationPercent:  0,
		RequestedBy:        "ops",
	}
	f.exports.items[f.export.ID] = f.export

	return f
}

func sequentialIDs() func() (string, error) {
	next := 0
	return func() (string, error) {
		next++
		return "example-" + string(rune('a'+next-1)), nil
	}
}

func (f *runnerFixture) addCorrection(text string, confirmed *aicorrection.Snapshot) *aicorrection.Correction {
	documentID := pulid.MustNew("doc_")
	correction := &aicorrection.Correction{
		ID:             pulid.MustNew("aicr_"),
		OrganizationID: testOrg,
		BusinessUnitID: testBU,
		Task:           aicorrection.TaskShipmentDraftExtraction,
		DocumentID:     &documentID,
		DocumentKind:   "RateConfirmation",
		Predicted:      confirmed,
		Confirmed:      confirmed,
		FieldResults: []aicorrection.FieldResult{
			{Key: aicorrection.FieldReference, Outcome: aicorrection.OutcomeCorrect},
		},
		ScoredCount:  1,
		CorrectCount: 1,
		CapturedAt:   testNow - 5000 + int64(len(f.corrections.items)),
	}
	f.corrections.items = append(f.corrections.items, correction)
	f.documents.docs[documentID] = &document.Document{ID: documentID, OriginalName: "Load 88123 Northstar.pdf"}
	f.contents.pages[documentID] = []*documentcontent.Page{{PageNumber: 1, ExtractedText: text}}

	return correction
}
