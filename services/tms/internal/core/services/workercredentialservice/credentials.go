package workercredentialservice

import (
	"context"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	supersededReason     = "Superseded by renewal"
	clearedOnProfile     = "Cleared on worker profile"
	workerResourceType   = "worker"
	credentialResourceID = "worker_credential"
)

func (s *Service) ListForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	includeArchived bool,
) ([]*worker.WorkerCredential, error) {
	return s.repo.ListForWorker(ctx, &repositories.ListWorkerCredentialsRequest{
		TenantInfo:      tenantInfo,
		WorkerID:        workerID,
		IncludeArchived: includeArchived,
		IncludeType:     true,
		IncludeDocument: true,
	})
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerCredential, error) {
	return s.repo.GetByID(ctx, &repositories.GetWorkerCredentialByIDRequest{
		ID:              id,
		TenantInfo:      tenantInfo,
		IncludeType:     true,
		IncludeDocument: true,
	})
}

type CreateRequest struct {
	Entity *worker.WorkerCredential
	// Renew archives the worker's current active credential of the same type
	// so this one takes its slot; without it a second active row is refused.
	Renew  bool
	UserID pulid.ID
}

func (s *Service) Create(
	ctx context.Context,
	req *CreateRequest,
) (*worker.WorkerCredential, error) {
	entity := req.Entity
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("workerId", entity.WorkerID.String()),
	)
	tenantInfo := credentialTenant(entity)

	credentialType, err := s.prepare(ctx, entity)
	if err != nil {
		return nil, err
	}
	if _, err = s.loadWorker(ctx, tenantInfo, entity.WorkerID); err != nil {
		return nil, err
	}

	createReq := &repositories.CreateWorkerCredentialRequest{Entity: entity}
	if req.Renew {
		createReq.SupersedeReason = supersededReason
		createReq.SupersededByID = req.UserID
	}
	created, err := s.repo.Create(ctx, createReq)
	if err != nil {
		log.Error("failed to create worker credential", zap.Error(err))
		return nil, err
	}
	created.CredentialType = credentialType

	comment := "Credential added"
	if req.Renew {
		comment = "Credential renewed"
	}
	s.auditCredential(created, nil, permission.OpCreate, req.UserID, comment, log)
	s.afterChange(ctx, created, permission.OpCreate, req.UserID)

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *worker.WorkerCredential,
	userID pulid.ID,
) (*worker.WorkerCredential, error) {
	log := s.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))
	tenantInfo := credentialTenant(entity)

	original, err := s.repo.GetByID(ctx, &repositories.GetWorkerCredentialByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !original.IsActive() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Archived credentials are read-only. Add a new credential instead",
		)
	}

	entity.WorkerID = original.WorkerID
	entity.CredentialTypeID = original.CredentialTypeID
	entity.Status = original.Status
	entity.VerifiedByID = original.VerifiedByID
	entity.VerifiedAt = original.VerifiedAt
	entity.ArchivedByID = original.ArchivedByID
	entity.ArchivedAt = original.ArchivedAt
	entity.ArchiveReason = original.ArchiveReason
	entity.CreatedAt = original.CreatedAt
	if entity.DocumentID.IsNil() {
		entity.DocumentID = original.DocumentID
	}

	credentialType, err := s.prepare(ctx, entity)
	if err != nil {
		return nil, err
	}

	if s.factsChanged(original, entity) && original.IsVerified() {
		entity.VerifiedByID = pulid.Nil
		entity.VerifiedAt = nil
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update worker credential", zap.Error(err))
		return nil, err
	}
	updated.CredentialType = credentialType

	s.auditCredential(updated, original, permission.OpUpdate, userID, "Credential updated", log)
	s.afterChange(ctx, updated, permission.OpUpdate, userID)

	return updated, nil
}

// factsChanged reports whether the fields a verifier vouched for moved, in
// which case the verification stamp no longer applies.
func (s *Service) factsChanged(original, next *worker.WorkerCredential) bool {
	return original.Number != next.Number ||
		original.IssuingAuthority != next.IssuingAuthority ||
		!equalInt64Ptr(original.IssuedAt, next.IssuedAt) ||
		!equalInt64Ptr(original.ExpiresAt, next.ExpiresAt) ||
		original.DocumentID != next.DocumentID
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

type StatusRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	Reason     string
	UserID     pulid.ID
}

func (s *Service) Verify(
	ctx context.Context,
	req *StatusRequest,
) (*worker.WorkerCredential, error) {
	log := s.l.With(zap.String("operation", "Verify"), zap.String("id", req.ID.String()))

	original, err := s.loadForChange(ctx, req)
	if err != nil {
		return nil, err
	}
	if !original.IsActive() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only active credentials can be verified",
		)
	}
	if original.CredentialType != nil && original.CredentialType.RequiresDocument &&
		original.DocumentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrRequired,
			"Attach the supporting document before verifying",
		)
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.VerifiedByID = req.UserID
	updated.VerifiedAt = &now

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to verify worker credential", zap.Error(err))
		return nil, err
	}
	saved.CredentialType = original.CredentialType

	s.auditCredential(saved, original, permission.OpApprove, req.UserID, "Credential verified", log)
	s.publish(ctx, credentialTenant(saved), realtimeResource, permission.OpApprove, saved.ID, req.UserID)

	return saved, nil
}

func (s *Service) Archive(
	ctx context.Context,
	req *StatusRequest,
) (*worker.WorkerCredential, error) {
	log := s.l.With(zap.String("operation", "Archive"), zap.String("id", req.ID.String()))

	original, err := s.loadForChange(ctx, req)
	if err != nil {
		return nil, err
	}
	if !original.IsActive() {
		return original, nil
	}

	now := timeutils.NowUnix()
	updated := *original
	updated.Status = worker.CredentialStatusArchived
	updated.ArchivedAt = &now
	updated.ArchivedByID = req.UserID
	updated.ArchiveReason = strings.TrimSpace(req.Reason)

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to archive worker credential", zap.Error(err))
		return nil, err
	}
	saved.CredentialType = original.CredentialType

	s.auditCredential(saved, original, permission.OpArchive, req.UserID, "Credential archived", log)
	s.afterChange(ctx, saved, permission.OpArchive, req.UserID)

	return saved, nil
}

type AttachDocumentRequest struct {
	ID         pulid.ID
	DocumentID pulid.ID
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
}

// AttachDocument links an uploaded file to the credential. The file must live
// in the same tenant and be filed against either the credential itself or the
// worker who holds it, so a document from another worker's packet can never be
// stapled on by id.
func (s *Service) AttachDocument(
	ctx context.Context,
	req *AttachDocumentRequest,
) (*worker.WorkerCredential, error) {
	log := s.l.With(zap.String("operation", "AttachDocument"), zap.String("id", req.ID.String()))

	original, err := s.loadForChange(ctx, &StatusRequest{ID: req.ID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if !original.IsActive() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Archived credentials cannot take new documents",
		)
	}

	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	ownedByCredential := doc.ResourceType == credentialResourceID && doc.ResourceID == original.ID.String()
	ownedByWorker := doc.ResourceType == workerResourceType && doc.ResourceID == original.WorkerID.String()
	if !ownedByCredential && !ownedByWorker {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document does not belong to this worker",
		)
	}

	updated := *original
	updated.DocumentID = doc.ID
	if original.DocumentID != doc.ID && original.IsVerified() {
		updated.VerifiedByID = pulid.Nil
		updated.VerifiedAt = nil
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		log.Error("failed to attach document to credential", zap.Error(err))
		return nil, err
	}
	saved.CredentialType = original.CredentialType
	saved.Document = doc

	s.auditCredential(saved, original, permission.OpUpdate, req.UserID, "Document attached", log)
	s.publish(ctx, credentialTenant(saved), realtimeResource, permission.OpUpdate, saved.ID, req.UserID)

	return saved, nil
}

func (s *Service) loadForChange(
	ctx context.Context,
	req *StatusRequest,
) (*worker.WorkerCredential, error) {
	original, err := s.repo.GetByID(ctx, &repositories.GetWorkerCredentialByIDRequest{
		ID:          req.ID,
		TenantInfo:  req.TenantInfo,
		IncludeType: true,
	})
	if err != nil {
		return nil, err
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Credential was changed by someone else. Reload and try again",
		)
	}
	return original, nil
}

// prepare normalises the row and validates it against its type; it returns the
// type so callers can hang it on the saved row without a second read.
func (s *Service) prepare(
	ctx context.Context,
	entity *worker.WorkerCredential,
) (*worker.WorkerCredentialType, error) {
	entity.Number = strings.TrimSpace(entity.Number)
	entity.IssuingAuthority = strings.TrimSpace(entity.IssuingAuthority)
	entity.Notes = strings.TrimSpace(entity.Notes)
	if entity.Status == "" {
		entity.Status = worker.CredentialStatusActive
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	var credentialType *worker.WorkerCredentialType
	if !entity.CredentialTypeID.IsNil() {
		typ, err := s.repo.GetTypeByID(ctx, &repositories.GetCredentialTypeByIDRequest{
			ID:         entity.CredentialTypeID,
			TenantInfo: credentialTenant(entity),
		})
		if err != nil {
			return nil, err
		}
		credentialType = typ
		if typ.Status != "Active" {
			multiErr.Add(
				"credentialTypeId",
				errortypes.ErrInvalid,
				"This credential type is no longer active",
			)
		}
		entity.ValidateAgainstType(typ, multiErr)
	}

	if !entity.DocumentID.IsNil() {
		if err := s.validateDocumentOwnership(ctx, entity); err != nil {
			return nil, err
		}
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return credentialType, nil
}

func (s *Service) validateDocumentOwnership(
	ctx context.Context,
	entity *worker.WorkerCredential,
) error {
	doc, err := s.documentRepo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         entity.DocumentID,
		TenantInfo: credentialTenant(entity),
	})
	if err != nil {
		return err
	}
	ownedByCredential := doc.ResourceType == credentialResourceID &&
		!entity.ID.IsNil() && doc.ResourceID == entity.ID.String()
	ownedByWorker := doc.ResourceType == workerResourceType &&
		doc.ResourceID == entity.WorkerID.String()
	if !ownedByCredential && !ownedByWorker {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document does not belong to this worker",
		)
	}
	return nil
}

func (s *Service) loadWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.Worker, error) {
	return s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             workerID,
		TenantInfo:     tenantInfo,
		IncludeProfile: true,
	})
}

// afterChange is the fan-out every credential write shares: mirror the row into
// the profile column it backs, re-grade the worker, and tell connected clients.
func (s *Service) afterChange(
	ctx context.Context,
	entity *worker.WorkerCredential,
	operation permission.Operation,
	userID pulid.ID,
) {
	tenantInfo := credentialTenant(entity)
	s.mirrorToProfile(ctx, entity)
	if _, err := s.RefreshCompliance(ctx, tenantInfo, entity.WorkerID); err != nil {
		s.l.Warn("failed to refresh worker compliance after credential change",
			zap.String("workerId", entity.WorkerID.String()),
			zap.Error(err))
	}
	s.publish(ctx, tenantInfo, realtimeResource, operation, entity.ID, userID)
	s.publish(ctx, tenantInfo, realtimeWorkers, permission.OpUpdate, entity.WorkerID, userID)
}

// mirrorToProfile copies a mirrored credential's expiry (and number, for the
// CDL) onto the worker profile. Archiving leaves the column alone unless no
// active credential of that type remains, in which case the nullable columns
// are cleared; the CDL column is NOT NULL and is never cleared.
func (s *Service) mirrorToProfile(ctx context.Context, entity *worker.WorkerCredential) {
	if entity.CredentialType == nil || !entity.CredentialType.ProfileField.IsSet() {
		return
	}
	field := entity.CredentialType.ProfileField
	tenantInfo := credentialTenant(entity)

	req := &repositories.PatchProfileCredentialFieldRequest{
		TenantInfo: tenantInfo,
		WorkerID:   entity.WorkerID,
		Field:      field,
	}
	if entity.IsActive() {
		req.ExpiresAt = entity.ExpiresAt
		if field.CarriesNumber() {
			req.Number = entity.Number
		}
	} else {
		if field == worker.CredentialProfileFieldLicenseExpiry {
			return
		}
		active, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerCredentialsRequest{
			TenantInfo: tenantInfo,
			WorkerID:   entity.WorkerID,
		})
		if err != nil {
			s.l.Warn("failed to check remaining credentials before clearing profile field",
				zap.Error(err))
			return
		}
		for _, cred := range active {
			if cred.CredentialTypeID == entity.CredentialTypeID {
				return
			}
		}
	}

	if err := s.workerRepo.PatchProfileCredentialField(ctx, req); err != nil {
		s.l.Warn("failed to mirror credential to worker profile",
			zap.String("workerId", entity.WorkerID.String()),
			zap.String("field", field.String()),
			zap.Error(err))
	}
}

// Summary grades every required slot and every held optional credential.
func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerCredentialSummary, error) {
	wrk, err := s.loadWorker(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	return s.summarize(ctx, tenantInfo, wrk)
}

func (s *Service) summarize(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
) (*worker.WorkerCredentialSummary, error) {
	types, err := s.ActiveTypes(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	credentials, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerCredentialsRequest{
		TenantInfo:      tenantInfo,
		WorkerID:        wrk.ID,
		IncludeType:     true,
		IncludeDocument: true,
	})
	if err != nil {
		return nil, err
	}
	return worker.BuildCredentialSummary(wrk, types, credentials, timeutils.NowUnix()), nil
}

// RefreshCompliance re-grades the worker and writes the roll-up onto the
// profile when it moved. It returns the summary it graded from.
func (s *Service) RefreshCompliance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerCredentialSummary, error) {
	wrk, err := s.loadWorker(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	summary, err := s.summarize(ctx, tenantInfo, wrk)
	if err != nil {
		return nil, err
	}
	nextExpiry := summary.NextExpiry()
	if wrk.Profile != nil &&
		wrk.Profile.ComplianceStatus == summary.ComplianceStatus &&
		sameOptionalUnix(wrk.Profile.NextCredentialExpiry, nextExpiry) {
		return summary, nil
	}
	if err = s.workerRepo.UpdateProfileComplianceStatus(
		ctx,
		&repositories.UpdateProfileComplianceStatusRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
			Status:     summary.ComplianceStatus,
			NextExpiry: nextExpiry,
		},
	); err != nil {
		return nil, err
	}
	return summary, nil
}

func sameOptionalUnix(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// SyncFromProfile is the forward mirror: after the worker form saves, every
// system type's profile column is compared with the worker's active credential
// of that type and the registry is brought in line. Rows the office touched
// only through the profile carry no verification.
func (s *Service) SyncFromProfile(ctx context.Context, wrk *worker.Worker, userID pulid.ID) error {
	if wrk == nil || wrk.Profile == nil {
		return nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID}
	log := s.l.With(zap.String("operation", "SyncFromProfile"), zap.String("workerId", wrk.ID.String()))

	types, err := s.ActiveTypes(ctx, tenantInfo)
	if err != nil {
		return err
	}
	credentials, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerCredentialsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   wrk.ID,
	})
	if err != nil {
		return err
	}
	activeByType := make(map[pulid.ID]*worker.WorkerCredential, len(credentials))
	for _, cred := range credentials {
		activeByType[cred.CredentialTypeID] = cred
	}

	changed := false
	for _, typ := range types {
		if !typ.ProfileField.IsSet() {
			continue
		}
		expiry := typ.ProfileField.Expiry(wrk.Profile)
		number := ""
		if typ.ProfileField.CarriesNumber() {
			number = strings.TrimSpace(wrk.Profile.LicenseNumber)
		}
		existing := activeByType[typ.ID]

		switch {
		case expiry == nil && existing == nil:
			continue
		case expiry == nil:
			if err = s.archiveFromProfile(ctx, existing, typ, userID); err != nil {
				log.Warn("failed to archive credential cleared on profile", zap.Error(err))
				continue
			}
			changed = true
		case existing == nil:
			if err = s.createFromProfile(ctx, wrk, typ, expiry, number); err != nil {
				log.Warn("failed to create credential from profile", zap.Error(err))
				continue
			}
			changed = true
		default:
			if equalInt64Ptr(existing.ExpiresAt, expiry) && (number == "" || existing.Number == number) {
				continue
			}
			if err = s.updateFromProfile(ctx, existing, typ, expiry, number); err != nil {
				log.Warn("failed to update credential from profile", zap.Error(err))
				continue
			}
			changed = true
		}
	}

	if changed {
		if _, err = s.RefreshCompliance(ctx, tenantInfo, wrk.ID); err != nil {
			log.Warn("failed to refresh compliance after profile sync", zap.Error(err))
		}
		s.publish(ctx, tenantInfo, realtimeResource, permission.OpUpdate, wrk.ID, userID)
	}
	return nil
}

func (s *Service) createFromProfile(
	ctx context.Context,
	wrk *worker.Worker,
	typ *worker.WorkerCredentialType,
	expiry *int64,
	number string,
) error {
	entity := &worker.WorkerCredential{
		OrganizationID:   wrk.OrganizationID,
		BusinessUnitID:   wrk.BusinessUnitID,
		WorkerID:         wrk.ID,
		CredentialTypeID: typ.ID,
		Status:           worker.CredentialStatusActive,
		Number:           number,
		ExpiresAt:        expiry,
	}
	if typ.ProfileField.CarriesNumber() && wrk.Profile.LicenseState != nil {
		entity.IssuingAuthority = wrk.Profile.LicenseState.Abbreviation
	}
	_, err := s.repo.Create(ctx, &repositories.CreateWorkerCredentialRequest{Entity: entity})
	return err
}

func (s *Service) updateFromProfile(
	ctx context.Context,
	existing *worker.WorkerCredential,
	typ *worker.WorkerCredentialType,
	expiry *int64,
	number string,
) error {
	updated := *existing
	updated.ExpiresAt = expiry
	if typ.ProfileField.CarriesNumber() && number != "" {
		updated.Number = number
	}
	if existing.IsVerified() {
		updated.VerifiedByID = pulid.Nil
		updated.VerifiedAt = nil
	}
	_, err := s.repo.Update(ctx, &updated)
	return err
}

func (s *Service) archiveFromProfile(
	ctx context.Context,
	existing *worker.WorkerCredential,
	typ *worker.WorkerCredentialType,
	userID pulid.ID,
) error {
	if typ.ProfileField == worker.CredentialProfileFieldLicenseExpiry {
		return nil
	}
	now := timeutils.NowUnix()
	updated := *existing
	updated.Status = worker.CredentialStatusArchived
	updated.ArchivedAt = &now
	updated.ArchivedByID = userID
	updated.ArchiveReason = clearedOnProfile
	_, err := s.repo.Update(ctx, &updated)
	return err
}

// ExpiryForecastItem is one credential inside the look-ahead window.
type ExpiryForecastItem struct {
	Credential      *worker.WorkerCredential
	Health          worker.CredentialHealth
	DaysUntilExpiry int64
}

type ExpiryForecast struct {
	HorizonDays   int
	ExpiringCount int
	ExpiredCount  int
	Items         []*ExpiryForecastItem
}

// ExpiryForecast lists every active credential that has expired inside the
// grace period or expires within the horizon, soonest first.
func (s *Service) ExpiryForecast(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	horizonDays, graceDays, limit int,
) (*ExpiryForecast, error) {
	if horizonDays <= 0 {
		horizonDays = 30
	}
	if graceDays < 0 {
		graceDays = 0
	}
	if limit <= 0 || limit > 1000 {
		limit = 250
	}

	page, err := s.repo.ListExpiring(ctx, &repositories.ListExpiringWorkerCredentialsRequest{
		TenantInfo:  tenantInfo,
		HorizonDays: horizonDays,
		GraceDays:   graceDays,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	forecast := &ExpiryForecast{
		HorizonDays: horizonDays,
		Items:       make([]*ExpiryForecastItem, 0, len(page)),
	}
	for _, cred := range page {
		health := cred.Health(now)
		item := &ExpiryForecastItem{Credential: cred, Health: health}
		if days := cred.DaysUntilExpiry(now); days != nil {
			item.DaysUntilExpiry = *days
		}
		forecast.Items = append(forecast.Items, item)
		if health == worker.CredentialHealthExpired {
			forecast.ExpiredCount++
		} else {
			forecast.ExpiringCount++
		}
	}

	sortForecast(forecast.Items)
	return forecast, nil
}

func sortForecast(items []*ExpiryForecastItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].DaysUntilExpiry < items[j].DaysUntilExpiry
	})
}
