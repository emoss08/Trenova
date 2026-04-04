package samsarasyncservice

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/emoss08/trenova/shared/samsara/drivers"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	samsaraWorkerExternalIDKey       = "trenovaWorkerId"
	samsaraOrganizationExternalIDKey = "trenovaOrganizationId"
	samsaraBusinessUnitExternalIDKey = "trenovaBusinessUnitId"
	samsaraWorkerSyncPageLimit       = 100
)

var (
	samsaraUsernameSanitizer = regexp.MustCompile(`[^a-z0-9._-]+`)
	errSamsaraEmptyDriverID  = errors.New("samsara returned an empty driver ID")
)

type Params struct {
	fx.In

	Logger             *zap.Logger
	Repo               repositories.WorkerRepository
	SamsaraClient      *sharedsamsara.Client `optional:"true"`
	IntegrationService *integrationservice.Service
}

type Service struct {
	l                  *zap.Logger
	repo               repositories.WorkerRepository
	samsaraClient      *sharedsamsara.Client
	integrationService *integrationservice.Service
}

func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.samsara-sync"),
		repo:               p.Repo,
		samsaraClient:      p.SamsaraClient,
		integrationService: p.IntegrationService,
	}
}

func (s *Service) GetWorkerSyncReadiness(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.WorkerSyncReadinessResponse, error) {
	counts, err := s.repo.GetWorkerSyncReadinessCounts(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to retrieve worker sync readiness",
		).WithInternal(err)
	}

	unsyncedActiveWorkers := max(counts.ActiveWorkers-counts.SyncedActiveWorkers, 0)

	return &services.WorkerSyncReadinessResponse{
		TotalWorkers:           counts.TotalWorkers,
		ActiveWorkers:          counts.ActiveWorkers,
		SyncedActiveWorkers:    counts.SyncedActiveWorkers,
		UnsyncedActiveWorkers:  unsyncedActiveWorkers,
		AllActiveWorkersSynced: unsyncedActiveWorkers == 0,
		LastCalculatedAt:       timeutils.NowUnix(),
	}, nil
}

func (s *Service) SyncWorkersToSamsara(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.SamsaraWorkerSyncResult, error) {
	samsaraClient, err := s.resolveSamsaraClient(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	log := s.l.With(
		zap.String("operation", "SyncWorkersToSamsara"),
		zap.String("organizationID", tenantInfo.OrgID.String()),
		zap.String("businessUnitID", tenantInfo.BuID.String()),
	)

	workers, err := s.listWorkersForSamsaraSync(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	remoteDrivers, err := samsaraClient.Drivers.ListAll(ctx, drivers.ListParams{
		Limit: 512,
	})
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list Samsara drivers").WithInternal(err)
	}

	result := &services.SamsaraWorkerSyncResult{
		TotalWorkers:  len(workers),
		RemoteDrivers: len(remoteDrivers),
		Failures:      make([]services.SamsaraWorkerSyncFailure, 0),
	}

	driverIndex := buildSamsaraDriverIndex(remoteDrivers)
	for _, currentWorker := range workers {
		s.syncWorkerToSamsara(ctx, samsaraClient, currentWorker, result, &driverIndex)
	}

	result.Failed = len(result.Failures)

	log.Info(
		"worker sync to samsara completed",
		zap.Any("result", result),
	)

	return result, nil
}

func (s *Service) DetectWorkerSyncDrift(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.WorkerSyncDriftResponse, error) {
	samsaraClient, err := s.resolveSamsaraClient(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	workers, err := s.listWorkersForSamsaraSync(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	remoteDrivers, err := samsaraClient.Drivers.ListAll(ctx, drivers.ListParams{
		Limit: 512,
	})
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list Samsara drivers").WithInternal(err)
	}

	driverIndex := buildSamsaraDriverIndex(remoteDrivers)
	driverByID := make(map[string]*drivers.Driver, len(remoteDrivers))
	for idx := range remoteDrivers {
		driverRecord := &remoteDrivers[idx]
		driverID := samsaraDriverID(driverRecord)
		if driverID != "" {
			driverByID[driverID] = driverRecord
		}
	}

	detectedAt := timeutils.NowUnix()
	drifts := make([]repositories.WorkerSyncDriftRecord, 0)
	for _, currentWorker := range workers {
		drifts = append(
			drifts,
			detectWorkerDrifts(currentWorker, &driverIndex, driverByID, detectedAt)...,
		)
	}

	if err = s.repo.ReplaceWorkerSyncDrifts(ctx, tenantInfo, drifts); err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to persist worker sync drifts",
		).WithInternal(err)
	}

	return toWorkerSyncDriftResponse(drifts, detectedAt), nil
}

func (s *Service) GetWorkerSyncDrift(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.WorkerSyncDriftResponse, error) {
	drifts, err := s.repo.ListWorkerSyncDrifts(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to retrieve worker sync drifts",
		).WithInternal(err)
	}

	var lastCalculatedAt int64
	if len(drifts) > 0 {
		lastCalculatedAt = drifts[0].DetectedAt
		for idx := range drifts {
			if drifts[idx].DetectedAt > lastCalculatedAt {
				lastCalculatedAt = drifts[idx].DetectedAt
			}
		}
	}

	return toWorkerSyncDriftResponse(drifts, lastCalculatedAt), nil
}

func (s *Service) RepairWorkerSyncDrift(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req services.RepairWorkerSyncDriftRequest,
) (*services.RepairWorkerSyncDriftResponse, error) {
	samsaraClient, err := s.resolveSamsaraClient(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	existingDrifts, err := s.repo.ListWorkerSyncDrifts(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to retrieve worker sync drifts",
		).WithInternal(err)
	}

	if len(existingDrifts) == 0 {
		return &services.RepairWorkerSyncDriftResponse{
			RequestedWorkers: 0,
			RepairedWorkers:  0,
			FailedWorkers:    0,
			Failures:         []services.SamsaraWorkerSyncFailure{},
		}, nil
	}

	targetWorkerIDs := buildTargetWorkerIDs(req.WorkerIDs, existingDrifts)

	if len(targetWorkerIDs) == 0 {
		return &services.RepairWorkerSyncDriftResponse{
			RequestedWorkers: 0,
			RepairedWorkers:  0,
			FailedWorkers:    0,
			Failures:         []services.SamsaraWorkerSyncFailure{},
		}, nil
	}

	workers, err := s.listWorkersForSamsaraSync(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	workersByID := buildWorkersByID(workers)

	remoteDrivers, err := samsaraClient.Drivers.ListAll(ctx, drivers.ListParams{
		Limit: 512,
	})
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list Samsara drivers").WithInternal(err)
	}
	driverIndex := buildSamsaraDriverIndex(remoteDrivers)

	response := &services.RepairWorkerSyncDriftResponse{
		RequestedWorkers: len(targetWorkerIDs),
		Failures:         make([]services.SamsaraWorkerSyncFailure, 0),
	}

	for idx := range targetWorkerIDs {
		workerID := targetWorkerIDs[idx]
		currentWorker, ok := workersByID[workerID]
		if !ok {
			response.Failures = append(response.Failures, services.SamsaraWorkerSyncFailure{
				WorkerID:  workerID,
				Worker:    workerID,
				Operation: "repair",
				Message:   "worker not found",
			})
			continue
		}

		result := &services.SamsaraWorkerSyncResult{
			Failures: make([]services.SamsaraWorkerSyncFailure, 0),
		}
		s.syncWorkerToSamsara(ctx, samsaraClient, currentWorker, result, &driverIndex)
		if len(result.Failures) > 0 {
			response.Failures = append(response.Failures, result.Failures...)
			continue
		}

		response.RepairedWorkers++
	}

	response.FailedWorkers = len(response.Failures)

	// Refresh persisted drifts after repair attempt.
	if _, detectErr := s.DetectWorkerSyncDrift(ctx, tenantInfo); detectErr != nil {
		return nil, detectErr
	}

	return response, nil
}

func detectWorkerDrifts(
	currentWorker *worker.Worker,
	driverIndex *samsaraDriverIndex,
	driverByID map[string]*drivers.Driver,
	detectedAt int64,
) []repositories.WorkerSyncDriftRecord {
	if currentWorker == nil || currentWorker.ID.IsNil() {
		return nil
	}
	if currentWorker.Status != domaintypes.StatusActive {
		return nil
	}

	workerID := currentWorker.ID.String()
	workerName := buildWorkerSyncName(currentWorker)
	localExternalID := strings.TrimSpace(currentWorker.ExternalID)
	remoteMappedID := strings.TrimSpace(driverIndex.remoteDriverByWorkerID[workerID])

	drifts := make([]repositories.WorkerSyncDriftRecord, 0, 3)
	if localExternalID == "" && remoteMappedID == "" {
		drifts = append(drifts, repositories.WorkerSyncDriftRecord{
			WorkerID:        workerID,
			WorkerName:      workerName,
			DriftType:       repositories.WorkerSyncDriftTypeMissingMapping,
			Message:         "active worker is missing a Samsara driver mapping",
			LocalExternalID: localExternalID,
			RemoteDriverID:  remoteMappedID,
			DetectedAt:      detectedAt,
		})
		return drifts
	}

	if localExternalID != "" {
		if _, ok := driverByID[localExternalID]; !ok {
			drifts = append(drifts, repositories.WorkerSyncDriftRecord{
				WorkerID:        workerID,
				WorkerName:      workerName,
				DriftType:       repositories.WorkerSyncDriftTypeMissingRemoteDriver,
				Message:         "worker points to a Samsara driver that no longer exists",
				LocalExternalID: localExternalID,
				DetectedAt:      detectedAt,
			})
		}
	}

	if localExternalID != "" && remoteMappedID != "" && localExternalID != remoteMappedID {
		drifts = append(drifts, repositories.WorkerSyncDriftRecord{
			WorkerID:        workerID,
			WorkerName:      workerName,
			DriftType:       repositories.WorkerSyncDriftTypeMappingMismatch,
			Message:         "local worker mapping differs from Samsara external ID mapping",
			LocalExternalID: localExternalID,
			RemoteDriverID:  remoteMappedID,
			DetectedAt:      detectedAt,
		})
	}

	effectiveRemoteID := remoteMappedID
	if effectiveRemoteID == "" {
		effectiveRemoteID = localExternalID
	}
	if effectiveRemoteID != "" {
		if remoteDriver := driverByID[effectiveRemoteID]; remoteDriver != nil &&
			strings.EqualFold(samsaraDriverActivationStatus(remoteDriver), "deactivated") {
			drifts = append(drifts, repositories.WorkerSyncDriftRecord{
				WorkerID:        workerID,
				WorkerName:      workerName,
				DriftType:       repositories.WorkerSyncDriftTypeRemoteDeactivated,
				Message:         "mapped Samsara driver is deactivated",
				LocalExternalID: localExternalID,
				RemoteDriverID:  effectiveRemoteID,
				DetectedAt:      detectedAt,
			})
		}
	}

	return drifts
}

func buildWorkerSyncName(currentWorker *worker.Worker) string {
	if currentWorker == nil || currentWorker.ID.IsNil() {
		return ""
	}
	workerName := strings.TrimSpace(currentWorker.FullName())
	if workerName != "" {
		return workerName
	}
	return currentWorker.ID.String()
}

func buildTargetWorkerIDs(
	requestedWorkerIDs []string,
	existingDrifts []repositories.WorkerSyncDriftRecord,
) []string {
	targetWorkerIDs := make([]string, 0)
	seenTargets := make(map[string]struct{})

	appendTarget := func(workerID string) {
		cleanWorkerID := strings.TrimSpace(workerID)
		if cleanWorkerID == "" {
			return
		}
		if _, ok := seenTargets[cleanWorkerID]; ok {
			return
		}
		seenTargets[cleanWorkerID] = struct{}{}
		targetWorkerIDs = append(targetWorkerIDs, cleanWorkerID)
	}

	if len(requestedWorkerIDs) > 0 {
		for idx := range requestedWorkerIDs {
			appendTarget(requestedWorkerIDs[idx])
		}
		return targetWorkerIDs
	}

	for idx := range existingDrifts {
		appendTarget(existingDrifts[idx].WorkerID)
	}

	return targetWorkerIDs
}

func buildWorkersByID(workers []*worker.Worker) map[string]*worker.Worker {
	workersByID := make(map[string]*worker.Worker, len(workers))
	for idx := range workers {
		currentWorker := workers[idx]
		if currentWorker == nil || currentWorker.ID.IsNil() {
			continue
		}
		workersByID[currentWorker.ID.String()] = currentWorker
	}
	return workersByID
}

func toWorkerSyncDriftResponse(
	drifts []repositories.WorkerSyncDriftRecord,
	lastCalculatedAt int64,
) *services.WorkerSyncDriftResponse {
	response := &services.WorkerSyncDriftResponse{
		Drifts:           make([]services.WorkerSyncDrift, 0, len(drifts)),
		WorkersWithDrift: 0,
		LastCalculatedAt: lastCalculatedAt,
	}
	seenWorkers := make(map[string]struct{}, len(drifts))

	for idx := range drifts {
		drift := drifts[idx]
		response.Drifts = append(response.Drifts, services.WorkerSyncDrift{
			WorkerID:        drift.WorkerID,
			WorkerName:      drift.WorkerName,
			DriftType:       drift.DriftType,
			Message:         drift.Message,
			LocalExternalID: drift.LocalExternalID,
			RemoteDriverID:  drift.RemoteDriverID,
			DetectedAt:      drift.DetectedAt,
		})

		if _, ok := seenWorkers[drift.WorkerID]; !ok {
			seenWorkers[drift.WorkerID] = struct{}{}
			response.WorkersWithDrift++
		}

		switch drift.DriftType {
		case repositories.WorkerSyncDriftTypeMissingMapping:
			response.MissingMapping++
		case repositories.WorkerSyncDriftTypeMissingRemoteDriver:
			response.MissingRemoteDriver++
		case repositories.WorkerSyncDriftTypeMappingMismatch:
			response.MappingMismatch++
		case repositories.WorkerSyncDriftTypeRemoteDeactivated:
			response.RemoteDeactivated++
		}
	}

	response.TotalDrifts = len(response.Drifts)
	return response
}

type samsaraDriverIndex struct {
	remoteDriverByWorkerID map[string]string
	remoteDriverByID       map[string]struct{}
}

func buildSamsaraDriverIndex(remoteDrivers []drivers.Driver) samsaraDriverIndex {
	index := samsaraDriverIndex{
		remoteDriverByWorkerID: make(map[string]string, len(remoteDrivers)),
		remoteDriverByID:       make(map[string]struct{}, len(remoteDrivers)),
	}

	for idx := range remoteDrivers {
		driverRecord := &remoteDrivers[idx]
		driverID := samsaraDriverID(driverRecord)
		if driverID != "" {
			index.remoteDriverByID[driverID] = struct{}{}
		}

		workerID := samsaraDriverExternalID(driverRecord, samsaraWorkerExternalIDKey)
		if workerID != "" && driverID != "" {
			index.remoteDriverByWorkerID[workerID] = driverID
		}
	}

	return index
}

func (s *Service) syncWorkerToSamsara(
	ctx context.Context,
	samsaraClient *sharedsamsara.Client,
	currentWorker *worker.Worker,
	result *services.SamsaraWorkerSyncResult,
	driverIndex *samsaraDriverIndex,
) {
	if currentWorker == nil || currentWorker.ID.IsNil() {
		return
	}

	if currentWorker.Status != domaintypes.StatusActive {
		result.SkippedInactive++
		return
	}
	result.ActiveWorkers++

	workerID := currentWorker.ID.String()
	workerName := strings.TrimSpace(currentWorker.FullName())
	if workerName == "" {
		workerName = workerID
	}

	if s.mapWorkerFromRemoteExternalID(ctx, currentWorker, result, driverIndex, workerName) {
		return
	}

	knownDriverID := strings.TrimSpace(currentWorker.ExternalID)
	if knownDriverID != "" {
		if _, ok := driverIndex.remoteDriverByID[knownDriverID]; ok {
			if updateErr := s.ensureSamsaraDriverExternalIDs(
				ctx,
				samsaraClient,
				currentWorker,
				knownDriverID,
			); updateErr != nil {
				appendSamsaraWorkerSyncFailure(
					result,
					workerID,
					workerName,
					"update",
					updateErr,
				)
				return
			}

			result.AlreadyMapped++
			result.UpdatedRemoteDrivers++
			return
		}
	}

	createdDriverID, operation, err := s.createAndMapWorkerToSamsara(
		ctx,
		samsaraClient,
		currentWorker,
	)
	if err != nil {
		appendSamsaraWorkerSyncFailure(
			result,
			workerID,
			workerName,
			operation,
			err,
		)
		return
	}

	result.CreatedDrivers++
	result.UpdatedMappings++
	driverIndex.remoteDriverByID[createdDriverID] = struct{}{}
}

func (s *Service) mapWorkerFromRemoteExternalID(
	ctx context.Context,
	currentWorker *worker.Worker,
	result *services.SamsaraWorkerSyncResult,
	driverIndex *samsaraDriverIndex,
	workerName string,
) bool {
	workerID := currentWorker.ID.String()
	mappedDriverID := strings.TrimSpace(driverIndex.remoteDriverByWorkerID[workerID])
	if mappedDriverID == "" {
		return false
	}

	if strings.TrimSpace(currentWorker.ExternalID) == mappedDriverID {
		result.AlreadyMapped++
		return true
	}

	if err := s.persistSamsaraWorkerMapping(ctx, currentWorker, mappedDriverID); err != nil {
		appendSamsaraWorkerSyncFailure(result, workerID, workerName, "map", err)
		return true
	}

	result.MappedFromExternalIDs++
	result.UpdatedMappings++
	return true
}

func (s *Service) createAndMapWorkerToSamsara(
	ctx context.Context,
	samsaraClient *sharedsamsara.Client,
	currentWorker *worker.Worker,
) (driverID, operation string, err error) {
	createReq := buildSamsaraDriverCreateRequest(currentWorker)
	createdDriver, createErr := samsaraClient.Drivers.Create(ctx, createReq)
	if createErr != nil {
		return "", "create", createErr
	}

	createdDriverID := samsaraDriverID(&createdDriver)
	if createdDriverID == "" {
		return "", "create", errSamsaraEmptyDriverID
	}

	if persistErr := s.persistSamsaraWorkerMapping(
		ctx,
		currentWorker,
		createdDriverID,
	); persistErr != nil {
		return "", "map", persistErr
	}

	return createdDriverID, "", nil
}

func appendSamsaraWorkerSyncFailure(
	result *services.SamsaraWorkerSyncResult,
	workerID string,
	workerName string,
	operation string,
	err error,
) {
	if result == nil || err == nil {
		return
	}

	result.Failures = append(result.Failures, services.SamsaraWorkerSyncFailure{
		WorkerID:  workerID,
		Worker:    workerName,
		Operation: operation,
		Message:   err.Error(),
	})
}

func (s *Service) listWorkersForSamsaraSync(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.Worker, error) {
	offset := 0
	workers := make([]*worker.Worker, 0)

	for {
		page, err := s.repo.List(ctx, &repositories.ListWorkersRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{
					Limit:  samsaraWorkerSyncPageLimit,
					Offset: offset,
				},
			},
		})
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Items) == 0 {
			break
		}

		workers = append(workers, page.Items...)
		offset += len(page.Items)
		if page.Total > 0 && offset >= page.Total {
			break
		}
		if len(page.Items) < samsaraWorkerSyncPageLimit {
			break
		}
	}

	return workers, nil
}

func (s *Service) ensureSamsaraDriverExternalIDs(
	ctx context.Context,
	samsaraClient *sharedsamsara.Client,
	currentWorker *worker.Worker,
	externalID string,
) error {
	if currentWorker == nil {
		return errortypes.NewBusinessError("worker is required")
	}

	externalIDs := buildSamsaraWorkerExternalIDs(currentWorker)
	if len(externalIDs) == 0 {
		return nil
	}

	if _, err := samsaraClient.Drivers.Update(ctx, externalID, drivers.UpdateRequest{
		ExternalIds: &externalIDs,
	}); err != nil {
		return fmt.Errorf(
			"update samsara driver %s external ids: %w",
			externalID,
			err,
		)
	}
	return nil
}

func (s *Service) persistSamsaraWorkerMapping(
	ctx context.Context,
	currentWorker *worker.Worker,
	externalID string,
) error {
	cleanDriverID := strings.TrimSpace(externalID)
	if currentWorker == nil || currentWorker.ID.IsNil() || cleanDriverID == "" {
		return errortypes.NewBusinessError("failed to persist worker mapping")
	}

	currentWorker.ExternalID = cleanDriverID
	if _, err := s.repo.Update(ctx, currentWorker); err != nil {
		return fmt.Errorf("update worker %s with samsara driver id: %w", currentWorker.ID, err)
	}
	return nil
}

func (s *Service) resolveSamsaraClient(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*sharedsamsara.Client, error) {
	if s.samsaraClient != nil && s.samsaraClient.Drivers != nil {
		return s.samsaraClient, nil
	}

	if s.integrationService == nil {
		return nil, errortypes.NewBusinessError("Samsara integration is not configured")
	}

	runtimeCfg, err := s.integrationService.GetRuntimeConfig(
		ctx, tenantInfo, integration.TypeSamsara,
	)
	if err != nil {
		return nil, err
	}

	client, err := sharedsamsara.New(
		runtimeCfg.Config["token"],
		sharedsamsara.WithBaseURL(runtimeCfg.Config["baseUrl"]),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to initialize Samsara client",
		).WithInternal(err)
	}

	return client, nil
}

func buildSamsaraDriverCreateRequest(currentWorker *worker.Worker) drivers.CreateRequest {
	name := strings.TrimSpace(currentWorker.FullName())
	if name == "" {
		name = currentWorker.ID.String()
	}

	username := buildSamsaraUsername(currentWorker)
	password := buildSamsaraPassword(currentWorker)
	request := drivers.CreateRequest{
		Name:     name,
		Username: username,
		Password: password,
	}

	externalIDs := buildSamsaraWorkerExternalIDs(currentWorker)
	request.ExternalIds = &externalIDs

	if normalizedPhone, ok := normalizeSamsaraPhone(currentWorker.PhoneNumber); ok {
		request.Phone = &normalizedPhone
	}

	if currentWorker.Profile != nil {
		licenseNumber := strings.TrimSpace(currentWorker.Profile.LicenseNumber)
		if licenseNumber != "" {
			request.LicenseNumber = &licenseNumber
		}
		request.EldExempt = &currentWorker.Profile.ELDExempt
	}

	if currentWorker.State != nil {
		licenseState := strings.TrimSpace(currentWorker.State.Abbreviation)
		if licenseState != "" {
			request.LicenseState = &licenseState
		}
	}

	return request
}

func buildSamsaraWorkerExternalIDs(currentWorker *worker.Worker) map[string]string {
	externalIDs := map[string]string{
		samsaraWorkerExternalIDKey: currentWorker.ID.String(),
	}
	if !currentWorker.OrganizationID.IsNil() {
		externalIDs[samsaraOrganizationExternalIDKey] = currentWorker.OrganizationID.String()
	}
	if !currentWorker.BusinessUnitID.IsNil() {
		externalIDs[samsaraBusinessUnitExternalIDKey] = currentWorker.BusinessUnitID.String()
	}
	return externalIDs
}

func buildSamsaraUsername(currentWorker *worker.Worker) string {
	candidate := strings.TrimSpace(currentWorker.Email)
	if before, _, ok := strings.Cut(candidate, "@"); ok {
		candidate = before
	}

	if candidate == "" {
		candidate = strings.TrimSpace(
			currentWorker.FirstName + "." + currentWorker.LastName,
		)
		candidate = strings.ToLower(candidate)
	}

	candidate = strings.ToLower(candidate)
	candidate = strings.ReplaceAll(candidate, "@", "")
	candidate = strings.ReplaceAll(candidate, " ", ".")
	candidate = samsaraUsernameSanitizer.ReplaceAllString(candidate, ".")
	candidate = strings.Trim(candidate, ".-_")
	if candidate == "" {
		candidate = "worker"
	}

	idSuffix := strings.ToLower(strings.TrimSpace(currentWorker.ID.String()))
	idSuffix = samsaraUsernameSanitizer.ReplaceAllString(idSuffix, "")
	if idSuffix == "" {
		idSuffix = "id"
	}

	username := "trenova." + candidate + "." + idSuffix
	if len(username) > 64 {
		username = username[:64]
	}
	username = strings.Trim(username, ".-_")
	if username == "" {
		username = "trenova.worker." + idSuffix
	}
	return username
}

func buildSamsaraPassword(currentWorker *worker.Worker) string {
	idSuffix := strings.ToLower(strings.TrimSpace(currentWorker.ID.String()))
	idSuffix = samsaraUsernameSanitizer.ReplaceAllString(idSuffix, "")
	if len(idSuffix) > 16 {
		idSuffix = idSuffix[len(idSuffix)-16:]
	}
	if idSuffix == "" {
		idSuffix = "worker"
	}
	return "Trenova!" + idSuffix + "#2026"
}

func normalizeSamsaraPhone(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	digitsBuilder := strings.Builder{}
	digitsBuilder.Grow(len(trimmed))
	for _, char := range trimmed {
		if char >= '0' && char <= '9' {
			digitsBuilder.WriteRune(char)
		}
	}
	digits := digitsBuilder.String()
	if digits == "" {
		return "", false
	}

	if len(digits) == 10 {
		return "+1" + digits, true
	}
	if len(digits) == 11 && strings.HasPrefix(digits, "1") {
		return "+" + digits, true
	}
	if len(digits) >= 11 && len(digits) <= 15 {
		return "+" + digits, true
	}
	if strings.HasPrefix(trimmed, "+") && len(digits) >= 8 && len(digits) <= 15 {
		return "+" + digits, true
	}
	return "", false
}

func samsaraDriverID(driverRecord *drivers.Driver) string {
	if driverRecord == nil || driverRecord.Id == nil {
		return ""
	}
	return strings.TrimSpace(*driverRecord.Id)
}

func samsaraDriverActivationStatus(driverRecord *drivers.Driver) string {
	if driverRecord == nil || driverRecord.DriverActivationStatus == nil {
		return ""
	}
	return strings.TrimSpace(string(*driverRecord.DriverActivationStatus))
}

func samsaraDriverExternalID(driverRecord *drivers.Driver, key string) string {
	if driverRecord == nil || driverRecord.ExternalIds == nil {
		return ""
	}
	rawValue, ok := (*driverRecord.ExternalIds)[key]
	if !ok || rawValue == nil {
		return ""
	}
	switch typed := rawValue.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		value := strings.TrimSpace(fmt.Sprint(typed))
		if value == "<nil>" {
			return ""
		}
		return value
	}
}
