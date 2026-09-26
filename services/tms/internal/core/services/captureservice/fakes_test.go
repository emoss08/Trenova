package captureservice

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/pdfassembly"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// world is every store the service touches, in memory. The fakes enforce the
// same rules the real repositories do where the service relies on them:
// tenant scoping, optimistic versions, the (batch, sequence) uniqueness of
// pages, and the replace-only-open-items rule.
type world struct {
	mu sync.Mutex

	tenant      pagination.TenantInfo
	devices     map[pulid.ID]*capture.CaptureDevice
	pairings    map[pulid.ID]*capture.CapturePairing
	profiles    map[pulid.ID]*capture.CaptureProfile
	requests    map[pulid.ID]*capture.CaptureRequest
	batches     map[pulid.ID]*capture.CaptureBatch
	pages       map[pulid.ID]*capture.CapturePage
	items       map[pulid.ID]*capture.CaptureItem
	coverSheets map[pulid.ID]*capture.CaptureCoverSheet
	records     map[string]bool
	docTypes    map[pulid.ID]*documenttype.DocumentType
	objects     map[string][]byte
	control     *tenant.DocumentControl
	denied      map[string]bool
	ownScope    bool
	inspections map[string]*services.CapturePageInspection
	sessions    map[pulid.ID]*documentupload.DocumentUploadSession
	uploaded    map[pulid.ID][]byte
	published   []string
}

func newWorld() *world {
	orgID, buID, userID := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")

	return &world{
		tenant:      pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		devices:     map[pulid.ID]*capture.CaptureDevice{},
		pairings:    map[pulid.ID]*capture.CapturePairing{},
		profiles:    map[pulid.ID]*capture.CaptureProfile{},
		requests:    map[pulid.ID]*capture.CaptureRequest{},
		batches:     map[pulid.ID]*capture.CaptureBatch{},
		pages:       map[pulid.ID]*capture.CapturePage{},
		items:       map[pulid.ID]*capture.CaptureItem{},
		coverSheets: map[pulid.ID]*capture.CaptureCoverSheet{},
		records:     map[string]bool{},
		docTypes:    map[pulid.ID]*documenttype.DocumentType{},
		objects:     map[string][]byte{},
		control:     tenant.NewDefaultDocumentControl(orgID, buID),
		denied:      map[string]bool{},
		inspections: map[string]*services.CapturePageInspection{},
		sessions:    map[pulid.ID]*documentupload.DocumentUploadSession{},
		uploaded:    map[pulid.ID][]byte{},
	}
}

func (w *world) service() *Service {
	return &Service{
		l:             zap.NewNop(),
		cfg:           &config.Config{App: config.AppConfig{WebBaseURL: "https://app.trenova.test"}},
		db:            fakeDB{},
		devices:       &fakeDevices{w},
		pairings:      &fakePairings{w},
		profiles:      &fakeProfiles{w},
		requests:      &fakeRequests{w},
		batches:       &fakeBatches{w},
		pages:         &fakePages{w},
		items:         &fakeItems{w},
		coverSheets:   &fakeCoverSheets{w},
		records:       &fakeRecords{w},
		controls:      &fakeControls{w: w},
		documentTypes: &fakeDocTypes{w: w},
		permissions:   &fakePermissions{w: w},
		storage:       &fakeStorage{w: w},
		cipher:        fakeCipher{},
		uploads:       &fakeUploads{w},
		assembler:     pdfassembly.New(),
		inspector:     &fakeInspector{w},
		realtime:      &fakeRealtime{w},
	}
}

func (w *world) addRecord(resourceType string) pulid.ID {
	id := pulid.MustNew("rec_")
	w.records[resourceType+":"+id.String()] = true

	return id
}

func (w *world) addDocType(code string) pulid.ID {
	id := pulid.MustNew("dt_")
	w.docTypes[id] = &documenttype.DocumentType{ID: id, Code: code}

	return id
}

func versionMismatch(entity string) error {
	return errortypes.NewValidationError("version", errortypes.ErrVersionMismatch, entity+" version mismatch")
}

func notFound(entity string) error {
	return errortypes.NewNotFoundError(entity + " not found within your organization")
}

func inTenant(orgID, buID pulid.ID, ti pagination.TenantInfo) bool {
	return orgID == ti.OrgID && buID == ti.BuID
}

// clone copies an entity so callers cannot mutate the store without a write.
func clone[T any](v *T) *T {
	c := *v

	return &c
}

type fakeDB struct{}

func (fakeDB) DB() *bun.DB                                  { return nil }
func (fakeDB) DBForContext(context.Context) bun.IDB         { return nil }
func (fakeDB) HealthCheck(context.Context) error            { return nil }
func (fakeDB) IsHealthy(context.Context) bool               { return true }
func (fakeDB) Close() error                                 { return nil }
func (fakeDB) WithTx(ctx context.Context, _ ports.TxOptions, fn func(context.Context, bun.Tx) error) error {
	return fn(ctx, bun.Tx{})
}

type fakeDevices struct{ w *world }

func (f *fakeDevices) Create(_ context.Context, e *capture.CaptureDevice) (*capture.CaptureDevice, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	f.w.devices[e.ID] = clone(e)

	return e, nil
}

func (f *fakeDevices) Update(_ context.Context, e *capture.CaptureDevice) (*capture.CaptureDevice, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	cur, ok := f.w.devices[e.ID]
	if !ok || cur.Version != e.Version {
		return nil, versionMismatch("Device")
	}
	e.Version++
	f.w.devices[e.ID] = clone(e)

	return e, nil
}

func (f *fakeDevices) GetByID(_ context.Context, req repositories.GetCaptureDeviceByIDRequest) (*capture.CaptureDevice, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	d, ok := f.w.devices[req.ID]
	if !ok || !inTenant(d.OrganizationID, d.BusinessUnitID, req.TenantInfo) {
		return nil, notFound("Device")
	}

	return clone(d), nil
}

func (f *fakeDevices) List(_ context.Context, req *repositories.ListCaptureDevicesRequest) (*pagination.ListResult[*capture.CaptureDevice], error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	out := []*capture.CaptureDevice{}
	for _, d := range f.w.devices {
		if req.UserID.IsNotNil() && d.UserID != req.UserID {
			continue
		}
		out = append(out, clone(d))
	}

	return &pagination.ListResult[*capture.CaptureDevice]{Items: out, Total: len(out)}, nil
}

func (f *fakeDevices) byHash(match func(*capture.CaptureDevice) bool) (*capture.CaptureDevice, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, d := range f.w.devices {
		if match(d) {
			return clone(d), nil
		}
	}

	return nil, notFound("Device")
}

func (f *fakeDevices) GetByAccessTokenHash(_ context.Context, h string) (*capture.CaptureDevice, error) {
	return f.byHash(func(d *capture.CaptureDevice) bool { return d.AccessTokenHash == h })
}

func (f *fakeDevices) GetByRefreshTokenHash(_ context.Context, h string) (*capture.CaptureDevice, error) {
	return f.byHash(func(d *capture.CaptureDevice) bool { return d.RefreshTokenHash == h })
}

func (f *fakeDevices) GetByPreviousRefreshHash(_ context.Context, h string) (*capture.CaptureDevice, error) {
	return f.byHash(func(d *capture.CaptureDevice) bool { return h != "" && d.PreviousRefreshHash == h })
}

func (f *fakeDevices) Touch(_ context.Context, req repositories.TouchCaptureDeviceRequest) error {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	if d, ok := f.w.devices[req.ID]; ok {
		seen := req.SeenAt
		d.LastSeenAt = &seen
		d.LastIP = req.IP
	}

	return nil
}

type fakePairings struct{ w *world }

func (f *fakePairings) Create(_ context.Context, e *capture.CapturePairing) (*capture.CapturePairing, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	if e.ID.IsNil() {
		e.ID = pulid.MustNew("cpair_")
	}
	f.w.pairings[e.ID] = clone(e)

	return e, nil
}

func (f *fakePairings) Update(_ context.Context, e *capture.CapturePairing) (*capture.CapturePairing, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	cur, ok := f.w.pairings[e.ID]
	if !ok || cur.Version != e.Version {
		return nil, versionMismatch("Pairing")
	}
	e.Version++
	f.w.pairings[e.ID] = clone(e)

	return e, nil
}

func (f *fakePairings) GetByDeviceCodeHash(_ context.Context, h string) (*capture.CapturePairing, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, p := range f.w.pairings {
		if p.DeviceCodeHash == h {
			return clone(p), nil
		}
	}

	return nil, notFound("Pairing")
}

func (f *fakePairings) GetOpenByUserCode(_ context.Context, code string) (*capture.CapturePairing, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, p := range f.w.pairings {
		if p.UserCode == code && (p.Status == capture.PairingPending || p.Status == capture.PairingApproved) {
			return clone(p), nil
		}
	}

	return nil, notFound("Pairing")
}

func (f *fakePairings) ExpireStale(_ context.Context, now int64) (int, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	n := 0
	for _, p := range f.w.pairings {
		if !p.Status.Terminal() && p.ExpiresAt <= now {
			p.Status = capture.PairingExpired
			n++
		}
	}

	return n, nil
}

type fakeProfiles struct{ w *world }

func (f *fakeProfiles) List(context.Context, *repositories.ListCaptureProfilesRequest) (*pagination.ListResult[*capture.CaptureProfile], error) {
	return &pagination.ListResult[*capture.CaptureProfile]{}, nil
}

func (f *fakeProfiles) GetByID(_ context.Context, req repositories.GetCaptureProfileByIDRequest) (*capture.CaptureProfile, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	p, ok := f.w.profiles[req.ID]
	if !ok {
		return nil, notFound("Capture profile")
	}

	return clone(p), nil
}

func (f *fakeProfiles) GetDefault(_ context.Context, _ pagination.TenantInfo) (*capture.CaptureProfile, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, p := range f.w.profiles {
		if p.IsDefault {
			return clone(p), nil
		}
	}

	return nil, notFound("Capture profile")
}

func (f *fakeProfiles) Create(_ context.Context, e *capture.CaptureProfile) (*capture.CaptureProfile, error) {
	f.w.profiles[e.ID] = clone(e)

	return e, nil
}

func (f *fakeProfiles) Update(_ context.Context, e *capture.CaptureProfile) (*capture.CaptureProfile, error) {
	f.w.profiles[e.ID] = clone(e)

	return e, nil
}

func (f *fakeProfiles) Delete(context.Context, repositories.DeleteCaptureProfileRequest) error {
	return nil
}

type fakeRequests struct{ w *world }

func (f *fakeRequests) Create(_ context.Context, e *capture.CaptureRequest) (*capture.CaptureRequest, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	if e.ID.IsNil() {
		e.ID = pulid.MustNew("creq_")
	}
	f.w.requests[e.ID] = clone(e)

	return e, nil
}

func (f *fakeRequests) Update(_ context.Context, e *capture.CaptureRequest) (*capture.CaptureRequest, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	cur, ok := f.w.requests[e.ID]
	if !ok || cur.Version != e.Version {
		return nil, versionMismatch("Capture request")
	}
	e.Version++
	f.w.requests[e.ID] = clone(e)

	return e, nil
}

func (f *fakeRequests) GetByID(_ context.Context, req repositories.GetCaptureRequestByIDRequest) (*capture.CaptureRequest, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	r, ok := f.w.requests[req.ID]
	if !ok || !inTenant(r.OrganizationID, r.BusinessUnitID, req.TenantInfo) {
		return nil, notFound("Capture request")
	}

	return clone(r), nil
}

func (f *fakeRequests) ListOpen(_ context.Context, req repositories.ListOpenCaptureRequestsRequest) ([]*capture.CaptureRequest, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	out := []*capture.CaptureRequest{}
	for _, r := range f.w.requests {
		if r.DeviceID == req.DeviceID && !r.Status.Terminal() {
			out = append(out, clone(r))
		}
	}
	slices.SortFunc(out, func(a, b *capture.CaptureRequest) int { return strings.Compare(a.ID.String(), b.ID.String()) })

	return out, nil
}

func (f *fakeRequests) ListForTarget(context.Context, repositories.ListCaptureRequestsForTargetRequest) ([]*capture.CaptureRequest, error) {
	return nil, nil
}

func (f *fakeRequests) ListExpired(_ context.Context, now int64, _ int) ([]*capture.CaptureRequest, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	out := []*capture.CaptureRequest{}
	for _, r := range f.w.requests {
		if r.IsExpired(now) {
			out = append(out, clone(r))
		}
	}

	return out, nil
}

type fakeBatches struct{ w *world }

func (f *fakeBatches) Create(_ context.Context, e *capture.CaptureBatch) (*capture.CaptureBatch, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, b := range f.w.batches {
		if b.DeviceID == e.DeviceID && b.ClientKey == e.ClientKey {
			return nil, fmt.Errorf("duplicate client key")
		}
	}
	f.w.batches[e.ID] = clone(e)

	return e, nil
}

func (f *fakeBatches) Update(_ context.Context, e *capture.CaptureBatch) (*capture.CaptureBatch, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	cur, ok := f.w.batches[e.ID]
	if !ok || cur.Version != e.Version {
		return nil, versionMismatch("Capture batch")
	}
	e.Version++
	e.ReceivedPageCount = cur.ReceivedPageCount
	stored := clone(e)
	stored.Pages, stored.Items = nil, nil
	f.w.batches[e.ID] = stored

	return e, nil
}

func (f *fakeBatches) GetByID(_ context.Context, req *repositories.GetCaptureBatchByIDRequest) (*capture.CaptureBatch, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	b, ok := f.w.batches[req.ID]
	if !ok || !inTenant(b.OrganizationID, b.BusinessUnitID, req.TenantInfo) {
		return nil, notFound("Capture batch")
	}
	out := clone(b)
	if req.IncludePages {
		out.Pages = f.w.pagesOf(b.ID)
	}
	if req.IncludeItems {
		out.Items = f.w.itemsOf(b.ID)
	}

	return out, nil
}

func (w *world) pagesOf(batchID pulid.ID) []*capture.CapturePage {
	out := []*capture.CapturePage{}
	for _, p := range w.pages {
		if p.BatchID == batchID {
			out = append(out, clone(p))
		}
	}
	slices.SortFunc(out, func(a, b *capture.CapturePage) int { return a.Sequence - b.Sequence })

	return out
}

func (w *world) itemsOf(batchID pulid.ID) []*capture.CaptureItem {
	out := []*capture.CaptureItem{}
	for _, i := range w.items {
		if i.BatchID == batchID {
			out = append(out, clone(i))
		}
	}
	slices.SortFunc(out, func(a, b *capture.CaptureItem) int { return a.Position - b.Position })

	return out
}

func (f *fakeBatches) GetByClientKey(_ context.Context, req repositories.GetCaptureBatchByClientKeyRequest) (*capture.CaptureBatch, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, b := range f.w.batches {
		if b.DeviceID == req.DeviceID && b.ClientKey == req.ClientKey {
			return clone(b), nil
		}
	}

	return nil, notFound("Capture batch")
}

func (f *fakeBatches) ListCursor(_ context.Context, req *repositories.ListCaptureBatchesRequest) (*pagination.CursorListResult[*capture.CaptureBatch], error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	out := []*capture.CaptureBatch{}
	for _, b := range f.w.batches {
		if req.UserID.IsNotNil() && b.UserID != req.UserID {
			continue
		}
		out = append(out, clone(b))
	}

	total := len(out)

	return &pagination.CursorListResult[*capture.CaptureBatch]{Items: out, TotalCount: &total}, nil
}

func (f *fakeBatches) ListStale(_ context.Context, req repositories.ListStaleCaptureBatchesRequest) ([]*capture.CaptureBatch, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	out := []*capture.CaptureBatch{}
	for _, b := range f.w.batches {
		if slices.Contains(req.Statuses, b.Status) && b.UpdatedAt < req.UpdatedBefore {
			out = append(out, clone(b))
		}
	}

	return out, nil
}

func (f *fakeBatches) ListRetentionDue(_ context.Context, req repositories.ListRetentionDueCaptureBatchesRequest) ([]*capture.CaptureBatch, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	out := []*capture.CaptureBatch{}
	for _, b := range f.w.batches {
		inFlight := b.Status == capture.BatchReceiving || b.Status == capture.BatchSealed || b.Status == capture.BatchProcessing
		if !inFlight && b.RetainUntil <= req.Now {
			out = append(out, clone(b))
		}
	}

	return out, nil
}

func (f *fakeBatches) IncrementReceived(_ context.Context, req repositories.IncrementCaptureBatchPagesRequest) error {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	f.w.batches[req.ID].ReceivedPageCount++

	return nil
}

func (f *fakeBatches) Delete(_ context.Context, req repositories.DeleteCaptureBatchRequest) error {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	delete(f.w.batches, req.ID)
	for id, p := range f.w.pages {
		if p.BatchID == req.ID {
			delete(f.w.pages, id)
		}
	}
	for id, i := range f.w.items {
		if i.BatchID == req.ID {
			delete(f.w.items, id)
		}
	}

	return nil
}

type fakePages struct{ w *world }

func (f *fakePages) Insert(_ context.Context, e *capture.CapturePage) (*capture.CapturePage, bool, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, p := range f.w.pages {
		if p.BatchID == e.BatchID && p.Sequence == e.Sequence {
			return clone(p), false, nil
		}
	}
	if e.ID.IsNil() {
		e.ID = pulid.MustNew("cpg_")
	}
	f.w.pages[e.ID] = clone(e)

	return e, true, nil
}

func (f *fakePages) Update(_ context.Context, e *capture.CapturePage) (*capture.CapturePage, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	f.w.pages[e.ID] = clone(e)

	return e, nil
}

func (f *fakePages) GetByID(_ context.Context, req repositories.GetCapturePageByIDRequest) (*capture.CapturePage, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	p, ok := f.w.pages[req.ID]
	if !ok {
		return nil, notFound("Capture page")
	}

	return clone(p), nil
}

func (f *fakePages) GetBySequence(_ context.Context, req repositories.GetCapturePageBySequenceRequest) (*capture.CapturePage, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for _, p := range f.w.pages {
		if p.BatchID == req.BatchID && p.Sequence == req.Sequence {
			return clone(p), nil
		}
	}

	return nil, notFound("Capture page")
}

func (f *fakePages) ListByBatch(_ context.Context, req repositories.ListCapturePagesRequest) ([]*capture.CapturePage, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()

	return f.w.pagesOf(req.BatchID), nil
}

type fakeItems struct{ w *world }

func (f *fakeItems) Create(_ context.Context, e *capture.CaptureItem) (*capture.CaptureItem, error) {
	f.w.items[e.ID] = clone(e)

	return e, nil
}

func (f *fakeItems) Update(_ context.Context, e *capture.CaptureItem) (*capture.CaptureItem, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	cur, ok := f.w.items[e.ID]
	if !ok || cur.Version != e.Version {
		return nil, versionMismatch("Capture item")
	}
	e.Version++
	f.w.items[e.ID] = clone(e)

	return e, nil
}

func (f *fakeItems) GetByID(_ context.Context, req repositories.GetCaptureItemByIDRequest) (*capture.CaptureItem, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	i, ok := f.w.items[req.ID]
	if !ok || !inTenant(i.OrganizationID, i.BusinessUnitID, req.TenantInfo) {
		return nil, notFound("Capture item")
	}

	return clone(i), nil
}

func (f *fakeItems) ListByBatch(_ context.Context, req repositories.ListCaptureItemsRequest) ([]*capture.CaptureItem, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()

	return f.w.itemsOf(req.BatchID), nil
}

func (f *fakeItems) ReplaceOpen(_ context.Context, req *repositories.ReplaceOpenCaptureItemsRequest) error {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	for id, i := range f.w.items {
		if i.BatchID == req.BatchID && i.Status.Open() {
			delete(f.w.items, id)
		}
	}
	for _, i := range req.Items {
		f.w.items[i.ID] = clone(i)
	}

	return nil
}

type fakeCoverSheets struct{ w *world }

func (f *fakeCoverSheets) CreateMany(_ context.Context, sheets []*capture.CaptureCoverSheet) error {
	for _, s := range sheets {
		f.w.coverSheets[s.ID] = clone(s)
	}

	return nil
}

func (f *fakeCoverSheets) GetByID(_ context.Context, req repositories.GetCaptureCoverSheetByIDRequest) (*capture.CaptureCoverSheet, error) {
	s, ok := f.w.coverSheets[req.ID]
	if !ok {
		return nil, notFound("Cover sheet")
	}

	return clone(s), nil
}

func (f *fakeCoverSheets) GetByTokenHash(_ context.Context, req repositories.GetCaptureCoverSheetByTokenRequest) (*capture.CaptureCoverSheet, error) {
	for _, s := range f.w.coverSheets {
		if s.TokenHash == req.TokenHash && inTenant(s.OrganizationID, s.BusinessUnitID, req.TenantInfo) {
			return clone(s), nil
		}
	}

	return nil, notFound("Cover sheet")
}

func (f *fakeCoverSheets) MarkUsed(_ context.Context, req repositories.MarkCaptureCoverSheetUsedRequest) error {
	if s, ok := f.w.coverSheets[req.ID]; ok {
		s.UseCount++
	}

	return nil
}

type fakeRecords struct{ w *world }

func (f *fakeRecords) Exists(_ context.Context, _ pagination.TenantInfo, resourceType string, id pulid.ID) (bool, error) {
	return f.w.records[resourceType+":"+id.String()], nil
}

type fakeControls struct {
	repositories.DocumentControlRepository

	w *world
}

func (f *fakeControls) Get(context.Context, repositories.GetDocumentControlRequest) (*tenant.DocumentControl, error) {
	return clone(f.w.control), nil
}

type fakeDocTypes struct {
	repositories.DocumentTypeRepository

	w *world
}

func (f *fakeDocTypes) GetByID(_ context.Context, req repositories.GetDocumentTypeByIDRequest) (*documenttype.DocumentType, error) {
	dt, ok := f.w.docTypes[req.ID]
	if !ok {
		return nil, notFound("Document type")
	}

	return dt, nil
}

func (f *fakeDocTypes) GetByCode(_ context.Context, req repositories.GetDocumentTypeByCodeRequest) (*documenttype.DocumentType, error) {
	for _, dt := range f.w.docTypes {
		if dt.Code == req.Code {
			return dt, nil
		}
	}

	return nil, notFound("Document type")
}

type fakePermissions struct {
	services.PermissionEngine

	w *world
}

func (f *fakePermissions) Check(_ context.Context, req *services.PermissionCheckRequest) (*services.PermissionCheckResult, error) {
	if f.w.denied[req.Resource+":"+string(req.Operation)] {
		return &services.PermissionCheckResult{Allowed: false, Reason: "no_permission"}, nil
	}
	scope := permission.DataScopeOrganization
	if f.w.ownScope {
		scope = permission.DataScopeOwn
	}

	return &services.PermissionCheckResult{Allowed: true, DataScope: scope}, nil
}

type fakeStorage struct {
	storage.Client

	w *world
}

func (f *fakeStorage) Upload(_ context.Context, p *storage.UploadParams) (*storage.FileInfo, error) {
	data, err := io.ReadAll(p.Body)
	if err != nil {
		return nil, err
	}
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	f.w.objects[p.Key] = data

	return &storage.FileInfo{}, nil
}

func (f *fakeStorage) Download(_ context.Context, key string) (*storage.DownloadResult, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	data, ok := f.w.objects[key]
	if !ok {
		return nil, fmt.Errorf("no object %s", key)
	}

	return &storage.DownloadResult{Body: io.NopCloser(bytes.NewReader(data))}, nil
}

func (f *fakeStorage) Exists(_ context.Context, key string) (bool, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	_, ok := f.w.objects[key]

	return ok, nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	delete(f.w.objects, key)

	return nil
}

// fakeCipher seals by binding the bytes to their AAD, so a test catches an
// object opened under a different key than it was sealed with.
type fakeCipher struct{}

func (fakeCipher) EncryptBytesWithAAD(plaintext []byte, aad encryptionservice.AAD) (string, error) {
	return "sealed|" + string(aad.Purpose) + "|" + aad.ResourceID + "|" + string(plaintext), nil
}

func (fakeCipher) DecryptBytesWithAAD(value string, aad encryptionservice.AAD) ([]byte, error) {
	prefix := "sealed|" + string(aad.Purpose) + "|" + aad.ResourceID + "|"
	if !strings.HasPrefix(value, prefix) {
		return nil, fmt.Errorf("aad mismatch")
	}

	return []byte(strings.TrimPrefix(value, prefix)), nil
}

type fakeUploads struct{ w *world }

func (f *fakeUploads) CreateSession(_ context.Context, req *services.CreateSessionRequest) (*documentupload.DocumentUploadSession, error) {
	session := &documentupload.DocumentUploadSession{
		ID:           pulid.MustNew("dus_"),
		ResourceID:   req.ResourceID,
		ResourceType: req.ResourceType,
		OriginalName: req.FileName,
	}
	f.w.sessions[session.ID] = session

	return session, nil
}

func (f *fakeUploads) UploadPart(_ context.Context, req *services.UploadPartRequest) (*documentupload.DocumentUploadSession, error) {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	f.w.uploaded[req.SessionID] = data

	return f.w.sessions[req.SessionID], nil
}

func (f *fakeUploads) Complete(_ context.Context, req *services.CompletionRequest) (*documentupload.DocumentUploadSession, error) {
	session := f.w.sessions[req.SessionID]
	docID := pulid.MustNew("doc_")
	session.DocumentID = &docID

	return session, nil
}

// fakeInspector reports a page as the test configured its bytes, or as an
// ordinary page of text.
type fakeInspector struct{ w *world }

func (f *fakeInspector) Inspect(_ context.Context, pdf []byte) (*services.CapturePageInspection, error) {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	if inspection, ok := f.w.inspections[string(pdf)]; ok {
		return inspection, nil
	}

	return &services.CapturePageInspection{WidthPx: 100, HeightPx: 130, Thumbnail: []byte("jpeg"), InkCoverage: 0.2}, nil
}

type fakeRealtime struct{ w *world }

func (f *fakeRealtime) PublishResourceInvalidation(_ context.Context, req *services.PublishResourceInvalidationRequest) error {
	f.w.mu.Lock()
	defer f.w.mu.Unlock()
	f.w.published = append(f.w.published, req.Resource+":"+req.Action)

	return nil
}

// pdfPage makes a distinct one-page PDF; shade makes each one's bytes unique.
func pdfPage(t *testing.T, shade uint8) []byte {
	t.Helper()

	img := image.NewGray(image.Rect(0, 0, 20, 26))
	for i := range img.Pix {
		img.Pix[i] = shade
	}
	img.Set(1, 1, color.Black)

	encoded := new(bytes.Buffer)
	require.NoError(t, png.Encode(encoded, img))

	out := new(bytes.Buffer)
	require.NoError(t, api.ImportImages(nil, out, []io.Reader{encoded}, pdfcpu.DefaultImportConfig(),
		model.NewDefaultConfiguration()))

	return out.Bytes()
}
