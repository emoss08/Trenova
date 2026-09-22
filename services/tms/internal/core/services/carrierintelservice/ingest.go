package carrierintelservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type ingestInput struct {
	tenant  pagination.TenantInfo
	control *carrierintel.CarrierIntelControl
	subject repositories.CarrierIntelSubject
	bound   *boundProvider
	result  *services.CarrierIntelLookupResult
	current *carrierintel.CarrierIntelSnapshot
	source  carrierintel.EventSource
	purpose carrierintel.Purpose
}

func (s *Service) evaluateSubject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	control *carrierintel.CarrierIntelControl,
	subject repositories.CarrierIntelSubject,
	profile *carrierintel.Profile,
	notFound bool,
) ([]carrierintel.Finding, error) {
	now := s.now()
	var overrides []carrierintel.OverrideRef
	if subject.CarrierID.IsNotNil() {
		active, err := s.overrideRepo.ListActiveByCarrierIDs(
			ctx, tenantInfo, []pulid.ID{subject.CarrierID}, now,
		)
		if err != nil {
			return nil, err
		}
		overrides = carrierintel.OverrideRefs(active, now)
	}

	return carrierintel.EvaluateFindings(&carrierintel.EvaluateInput{
		Profile:         profile,
		Now:             now,
		Subject:         subject.SubjectType,
		BrokerAuthority: subject.Broker,
		AuthorityExempt: subject.Exempt,
		NotFound:        notFound,
		Settings:        control.Rules,
		Overrides:       overrides,
	}), nil
}

func (s *Service) ingest(ctx context.Context, in *ingestInput) (*FetchResult, error) {
	now := s.now()
	result := in.result
	current := in.current
	if current != nil && current.Provider != in.bound.provider {
		current = nil
	}

	resolved := carrierintel.ProfileMergeResult{
		Profile:        &carrierintel.Profile{Coverage: []carrierintel.Section{}},
		Depth:          result.Depth,
		DepthFetchedAt: now,
	}
	if !result.NotFound && result.Profile != nil {
		resolved = carrierintel.ResolveIncomingProfile(&carrierintel.ProfileMergeInput{
			Current:       current,
			Incoming:      result.Profile,
			IncomingDepth: result.Depth,
			Control:       in.control,
			Now:           now,
		})
	}
	profile := resolved.Profile
	depth := resolved.Depth

	hash, err := profile.ContentHash()
	if err != nil {
		return nil, err
	}

	findings, err := s.evaluateSubject(
		ctx,
		in.tenant,
		in.control,
		in.subject,
		profile,
		result.NotFound,
	)
	if err != nil {
		return nil, err
	}

	if current != nil && current.ContentHash == hash && current.NotFound == result.NotFound &&
		current.Depth == depth {
		return s.confirmUnchanged(ctx, in, current, findings, now)
	}

	var priorFindings []carrierintel.Finding
	var changes []carrierintel.FieldChange
	if current != nil {
		priorFindings = current.Findings
		if !current.NotFound && !result.NotFound {
			changes, err = carrierintel.DiffProfiles(
				current.Profile,
				profile,
				carrierintel.DiffOptionsForDepths(current.Depth, result.Depth),
			)
			if err != nil {
				return nil, err
			}
		}
	}

	findings = s.confirmNewBlocks(ctx, in, priorFindings, findings)

	snapshot := &carrierintel.CarrierIntelSnapshot{
		OrganizationID: in.tenant.OrgID,
		BusinessUnitID: in.tenant.BuID,
		SubjectType:    in.subject.SubjectType,
		SubjectID:      in.subject.SubjectID,
		CarrierID:      in.subject.CarrierID,
		DOTNumber:      firstNonEmpty(profile.DOTNumber(), in.subject.DOTNumber),
		DocketNumber:   docketOf(profile, in.subject),
		Provider:       in.bound.provider,
		ProviderRef:    result.ProviderRef,
		Depth:          depth,
		Source:         snapshotSource(in),
		IsCurrent:      true,
		NotFound:       result.NotFound,
		Profile:        profile,
		ContentHash:    hash,
		FetchedAt:      now,
		FetchedDepth:   result.Depth,
		DepthFetchedAt: resolved.DepthFetchedAt,
		SourceAsOf:     result.SourceAsOf,
	}
	snapshot.ApplyFindings(findings, in.control.PolicyVersion)
	snapshot.ReviewState = nextReviewState(current, priorFindings, findings)

	raised := carrierintel.NewlyRaised(priorFindings, findings)
	events := s.buildEvents(in, snapshot, changes, raised, current, now)

	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if len(result.Raw) > 0 {
			raw, rawErr := s.rawRepo.Insert(txCtx, &carrierintel.CarrierIntelRawPayload{
				OrganizationID: in.tenant.OrgID,
				BusinessUnitID: in.tenant.BuID,
				Provider:       in.bound.provider,
				Endpoint:       result.Endpoint.String(),
				DOTNumber:      snapshot.DOTNumber,
				Payload:        jsonutils.RawJSON(result.Raw),
				FetchedAt:      now,
				ExpiresAt:      now + int64(in.control.RawRetentionDays)*timeutils.SecondsPerDay,
			})
			if rawErr != nil {
				return rawErr
			}
			snapshot.RawPayloadID = raw.ID
		}

		inserted, txErr := s.snapshotRepo.InsertCurrent(txCtx, snapshot)
		if txErr != nil {
			return txErr
		}
		snapshot = inserted

		for _, event := range events {
			event.SnapshotID = snapshot.ID
		}
		if events, txErr = s.eventRepo.InsertIgnoreDuplicates(txCtx, events); txErr != nil {
			return txErr
		}

		return s.updateCarrierSummary(txCtx, in.tenant, in.subject, snapshot)
	})
	if err != nil {
		return nil, err
	}

	s.afterIngest(ctx, in, snapshot, raised, events)

	return &FetchResult{Snapshot: snapshot, Changes: changes, Raised: raised}, nil
}

func (s *Service) confirmUnchanged(
	ctx context.Context,
	in *ingestInput,
	current *carrierintel.CarrierIntelSnapshot,
	findings []carrierintel.Finding,
	now int64,
) (*FetchResult, error) {
	current.DepthFetchedAt = current.DepthAsOf()
	if in.result.Depth.Satisfies(current.Depth) {
		current.DepthFetchedAt = now
	}
	current.FetchedDepth = in.result.Depth
	current.ConfirmedAt = &now
	raised := carrierintel.NewlyRaised(current.Findings, findings)
	priorFindings := current.Findings
	current.ApplyFindings(findings, in.control.PolicyVersion)
	current.ReviewState = nextReviewState(current, priorFindings, findings)

	events := s.buildEvents(in, current, nil, raised, current, now)
	for _, event := range events {
		event.SnapshotID = current.ID
	}

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if txErr := s.snapshotRepo.UpdateEvaluation(txCtx, current); txErr != nil {
			return txErr
		}
		var txErr error
		if events, txErr = s.eventRepo.InsertIgnoreDuplicates(txCtx, events); txErr != nil {
			return txErr
		}
		return s.updateCarrierSummary(txCtx, in.tenant, in.subject, current)
	})
	if err != nil {
		return nil, err
	}

	if len(raised) > 0 {
		s.afterIngest(ctx, in, current, raised, events)
	}
	return &FetchResult{Snapshot: current, Raised: raised}, nil
}

func (s *Service) confirmNewBlocks(
	ctx context.Context,
	in *ingestInput,
	prior, findings []carrierintel.Finding,
) []carrierintel.Finding {
	if !in.control.ConfirmBlockingChanges || in.current == nil || in.bound.fallback ||
		in.source == carrierintel.EventSourceRuleEvaluation {
		return findings
	}
	added := carrierintel.NewBlockingCodes(prior, findings)
	if len(added) == 0 {
		return findings
	}

	fallback, ok := s.resolveFallback(ctx, in.tenant, in.control)
	if !ok || fallback.provider == in.bound.provider || !s.providerAvailable(fallback) {
		return findings
	}

	confirmation, err := s.lookup(ctx, &lookupAttempt{
		tenant:  in.tenant,
		control: in.control,
		primary: fallback,
		subject: in.subject,
		wanted:  carrierintel.LookupDepthFMCSA,
	}, fallback)
	if err != nil || confirmation.NotFound || confirmation.Profile == nil {
		return findings
	}
	confirmation.Profile.NormalizeCoverage()

	confirmed, err := s.evaluateSubject(
		ctx, in.tenant, in.control, in.subject, confirmation.Profile, false,
	)
	if err != nil {
		return findings
	}

	for idx := range findings {
		finding := &findings[idx]
		if !slices.ContainsFunc(
			added,
			func(f carrierintel.Finding) bool { return f.Code == finding.Code },
		) {
			continue
		}
		if !confirmation.Profile.CoversAll(ruleSections(finding.Code)) {
			continue
		}
		reproduced := slices.ContainsFunc(confirmed, func(f carrierintel.Finding) bool {
			return f.Code == finding.Code && !f.Unverifiable
		})
		if !reproduced {
			finding.MarkUnconfirmed(fallback.provider.String())
		}
	}
	return findings
}

func ruleSections(code carrierintel.RuleCode) []carrierintel.Section {
	def, ok := carrierintel.RuleByCode(code)
	if !ok {
		return nil
	}
	return def.RequiredSections
}

func snapshotSource(in *ingestInput) carrierintel.SnapshotSource {
	if in.bound.fallback {
		return carrierintel.SnapshotSourceFallback
	}
	if in.source == carrierintel.EventSourceNativeChangeFeed {
		return carrierintel.SnapshotSourceChangeFeedPatch
	}
	return carrierintel.SnapshotSourcePrimary
}

func nextReviewState(
	current *carrierintel.CarrierIntelSnapshot,
	prior, findings []carrierintel.Finding,
) carrierintel.ReviewState {
	if len(carrierintel.NewBlockingCodes(prior, findings)) > 0 {
		return carrierintel.ReviewStateNeedsReview
	}
	if len(carrierintel.BlockingCodes(findings)) == 0 {
		return carrierintel.ReviewStateNone
	}
	if current != nil && current.ReviewState.IsValid() {
		return current.ReviewState
	}
	return carrierintel.ReviewStateNeedsReview
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func docketOf(profile *carrierintel.Profile, subject repositories.CarrierIntelSubject) string {
	if profile != nil && profile.Identity != nil && profile.Identity.DocketNumber != "" {
		return profile.Identity.DocketNumber
	}
	return subject.DocketNumber
}

func subjectLabel(subject repositories.CarrierIntelSubject, profile *carrierintel.Profile) string {
	name := subject.Name
	if name == "" && profile != nil && profile.Identity != nil {
		name = profile.Identity.LegalName
	}
	if name == "" {
		name = "USDOT " + subject.DOTNumber
	}
	return name
}

func (s *Service) buildEvents(
	in *ingestInput,
	snapshot *carrierintel.CarrierIntelSnapshot,
	changes []carrierintel.FieldChange,
	raised []carrierintel.Finding,
	current *carrierintel.CarrierIntelSnapshot,
	now int64,
) []*carrierintel.CarrierIntelEvent {
	day := timeutils.DayIndexUTC(now)
	name := subjectLabel(in.subject, snapshot.Profile)
	events := make([]*carrierintel.CarrierIntelEvent, 0, len(changes)+len(raised)+1)

	base := func() *carrierintel.CarrierIntelEvent {
		return &carrierintel.CarrierIntelEvent{
			OrganizationID: in.tenant.OrgID,
			BusinessUnitID: in.tenant.BuID,
			SubjectType:    in.subject.SubjectType,
			SubjectID:      in.subject.SubjectID,
			CarrierID:      in.subject.CarrierID,
			DOTNumber:      snapshot.DOTNumber,
			SubjectName:    name,
			Provider:       in.bound.provider,
			DetectedAt:     now,
			Status:         carrierintel.EventStatusOpen,
		}
	}

	for idx := range changes {
		change := changes[idx]
		if change.Severity.Rank() < carrierintel.SeverityLow.Rank() {
			continue
		}
		event := base()
		event.Source = in.source
		event.Category = change.Section
		event.FieldPath = change.Path
		event.Severity = change.Severity
		event.PriorValue = change.Prior
		event.CurrentValue = change.Current
		event.Summary = fmt.Sprintf("%s changed from %s to %s",
			change.Label,
			carrierintel.FormatFieldValue(change.Path, change.Prior),
			carrierintel.FormatFieldValue(change.Path, change.Current))
		event.Fingerprint = carrierintel.EventFingerprint(carrierintel.EventFingerprintInput{
			Provider:    in.bound.provider,
			SubjectType: in.subject.SubjectType,
			SubjectID:   in.subject.SubjectID,
			Discriminat: "field:" + change.Path,
			Prior:       change.Prior,
			Current:     change.Current,
			Day:         day,
		})
		events = append(events, event)
	}

	for idx := range raised {
		finding := raised[idx]
		if finding.Action == carrierintel.RuleActionOff {
			continue
		}
		event := base()
		event.Source = carrierintel.EventSourceRuleEvaluation
		event.Category = finding.Category
		event.RuleCode = finding.Code
		event.Action = finding.Action
		event.Severity = finding.Severity
		event.Summary = finding.Message
		event.Fingerprint = carrierintel.EventFingerprint(carrierintel.EventFingerprintInput{
			Provider:    in.bound.provider,
			SubjectType: in.subject.SubjectType,
			SubjectID:   in.subject.SubjectID,
			Discriminat: "rule:" + finding.Code.String() + ":" + finding.Action.String(),
			Current:     finding.Message,
			Day:         day,
		})
		events = append(events, event)
	}

	if current != nil && current.NotFound != snapshot.NotFound {
		event := base()
		event.Source = in.source
		event.Category = carrierintel.SectionIdentity
		event.FieldPath = "identity.found"
		event.Severity = carrierintel.SeverityHigh
		event.PriorValue = !current.NotFound
		event.CurrentValue = !snapshot.NotFound
		event.Summary = "Provider record is now available"
		if snapshot.NotFound {
			event.Summary = "Provider no longer has a record for this USDOT number"
		}
		event.Fingerprint = carrierintel.EventFingerprint(carrierintel.EventFingerprintInput{
			Provider:    in.bound.provider,
			SubjectType: in.subject.SubjectType,
			SubjectID:   in.subject.SubjectID,
			Discriminat: "found",
			Prior:       !current.NotFound,
			Current:     !snapshot.NotFound,
			Day:         day,
		})
		events = append(events, event)
	}

	return events
}

func (s *Service) updateCarrierSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subject repositories.CarrierIntelSubject,
	snapshot *carrierintel.CarrierIntelSnapshot,
) error {
	if subject.SubjectType != carrierintel.SubjectTypeCarrier || subject.CarrierID.IsNil() {
		return nil
	}
	risk := snapshot.RiskLevel.String()
	if snapshot.RiskLevel == carrierintel.RiskLevelUnknown {
		risk = ""
	}
	return s.snapshotRepo.UpdateCarrierSummary(ctx, &repositories.UpdateCarrierIntelSummaryRequest{
		TenantInfo:     tenantInfo,
		CarrierID:      subject.CarrierID,
		RiskLevel:      risk,
		ReviewRequired: snapshot.ReviewState == carrierintel.ReviewStateNeedsReview,
		BlockingCount:  len(snapshot.BlockingCodes),
	})
}

func (s *Service) afterIngest(
	ctx context.Context,
	in *ingestInput,
	snapshot *carrierintel.CarrierIntelSnapshot,
	raised []carrierintel.Finding,
	events []*carrierintel.CarrierIntelEvent,
) {
	ctx = context.WithoutCancel(ctx)

	s.notifyRaised(ctx, in, snapshot, raised)

	if in.subject.SubjectType == carrierintel.SubjectTypeCarrier &&
		in.subject.CarrierID.IsNotNil() {
		s.syncCarrierRecord(ctx, in, snapshot, raised)
	}

	if in.purpose.IsInteractive() {
		if err := s.auditService.LogAction(&services.LogActionParams{
			Resource:       permission.ResourceCarrierIntelligence,
			ResourceID:     snapshot.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         in.tenant.UserID,
			PrincipalType:  services.PrincipalTypeUser,
			PrincipalID:    in.tenant.UserID,
			CurrentState:   jsonutils.MustToJSON(snapshotAuditView(snapshot)),
			OrganizationID: in.tenant.OrgID,
			BusinessUnitID: in.tenant.BuID,
		}); err != nil {
			s.l.Warn("failed to audit carrier intelligence lookup", zap.Error(err))
		}
	}

	s.publish(ctx, in.tenant, "carrier_intelligence", "updated", snapshot.ID)
	if in.subject.CarrierID.IsNotNil() {
		s.publish(ctx, in.tenant, "carriers", "updated", in.subject.CarrierID)
	}
	if len(events) > 0 {
		s.publish(ctx, in.tenant, "carrier_intel_events", "created", pulid.Nil)
		s.projectEventsToWatchtower(ctx, events)
		s.publishEventsToAgents(ctx, in.tenant, events)
	}
}

// projectEventsToWatchtower puts the changes worth acting on from this
// ingest on the feed. A finding that is only part of the record stays on
// the carrier's page.
func (s *Service) projectEventsToWatchtower(
	ctx context.Context,
	events []*carrierintel.CarrierIntelEvent,
) {
	if s.watchtower == nil {
		return
	}

	for _, event := range events {
		if !watchtowersources.CarrierIntelEventOnTower(event) {
			continue
		}
		s.watchtower.Upsert(ctx, watchtowersources.DescribeCarrierIntelEvent(event))
	}
}

// publishEventsToAgents wakes the carrier risk desk on the same findings
// the tower shows, and on no others: the filter that decides what is worth
// a person's attention decides what is worth a run.
func (s *Service) publishEventsToAgents(
	ctx context.Context,
	tenant pagination.TenantInfo,
	events []*carrierintel.CarrierIntelEvent,
) {
	if s.publisher == nil {
		return
	}

	for _, event := range events {
		if !watchtowersources.CarrierIntelEventOnTower(event) {
			continue
		}
		services.PublishAgentEvent(ctx, s.publisher, services.AgentEvent{
			Kind:       agent.EventCarrierIntelEventOpened,
			SubjectID:  event.ID,
			TenantInfo: tenant,
		})
	}
}

func snapshotAuditView(snapshot *carrierintel.CarrierIntelSnapshot) map[string]any {
	return map[string]any{
		"subjectType":   snapshot.SubjectType,
		"subjectId":     snapshot.SubjectID,
		"dotNumber":     snapshot.DOTNumber,
		"provider":      snapshot.Provider,
		"depth":         snapshot.Depth,
		"riskLevel":     snapshot.RiskLevel,
		"blockingCodes": snapshot.BlockingCodes,
		"advisoryCodes": snapshot.AdvisoryCodes,
		"notFound":      snapshot.NotFound,
		"fetchedAt":     snapshot.FetchedAt,
	}
}

func (s *Service) publish(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource, action string,
	recordID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	actorType := services.PrincipalTypeSystem
	actorID := services.SystemPrincipalID
	if tenantInfo.UserID.IsNotNil() {
		actorType = services.PrincipalTypeUser
		actorID = tenantInfo.UserID
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    tenantInfo.UserID,
		ActorType:      actorType,
		ActorID:        actorID,
		Resource:       resource,
		Action:         action,
		RecordID:       recordID,
	}); err != nil {
		s.l.Warn("failed to publish carrier intelligence invalidation", zap.Error(err))
	}
}

func (s *Service) notifyRaised(
	ctx context.Context,
	in *ingestInput,
	snapshot *carrierintel.CarrierIntelSnapshot,
	raised []carrierintel.Finding,
) {
	name := subjectLabel(in.subject, snapshot.Profile)
	link := monitoringLink
	if in.subject.CarrierID.IsNotNil() {
		link = carrierLinkPrefix + in.subject.CarrierID.String()
	}
	related := map[string]any{
		"subjectType": in.subject.SubjectType.String(),
		"subjectId":   in.subject.SubjectID,
		"dotNumber":   snapshot.DOTNumber,
	}
	if in.subject.CarrierID.IsNotNil() {
		related["carrierId"] = in.subject.CarrierID.String()
	}
	day := timeutils.FormatDateKeyUTC(s.now())

	for idx := range raised {
		finding := raised[idx]
		var priority notification.Priority
		var eventType, title string
		switch finding.Action {
		case carrierintel.RuleActionBlock:
			priority = notification.PriorityCritical
			eventType = EventCarrierIntelBlock
			title = name + " is blocked by carrier intelligence"
		case carrierintel.RuleActionNotify:
			priority = notification.PriorityMedium
			eventType = EventCarrierIntelChange
			title = name + " needs attention"
		default:
			continue
		}
		s.sendNotification(ctx, &notificationRequest{
			tenant:      in.tenant,
			eventType:   eventType,
			correlation: fmt.Sprintf("ci-%s-%s-%s", in.subject.SubjectID, finding.Code, day),
			priority:    priority,
			title:       title,
			message:     finding.Message,
			link:        link,
			related:     related,
			data:        map[string]any{"ruleCode": finding.Code.String()},
		})
	}
}

func (s *Service) syncCarrierRecord(
	ctx context.Context,
	in *ingestInput,
	snapshot *carrierintel.CarrierIntelSnapshot,
	raised []carrierintel.Finding,
) {
	if snapshot.NotFound || s.carrierService == nil {
		return
	}

	newBlocks := make([]string, 0, len(raised))
	for idx := range raised {
		if raised[idx].IsBlocking() {
			newBlocks = append(newBlocks, raised[idx].Message)
		}
	}

	for attempt := range 2 {
		entity, err := s.carrierRepo.GetByID(ctx, repositories.GetCarrierByIDRequest{
			ID:         in.subject.CarrierID,
			TenantInfo: in.tenant,
			CarrierFilterOptions: repositories.CarrierFilterOptions{
				IncludeContacts:          true,
				IncludeInsurancePolicies: true,
				IncludeEDIChannels:       true,
			},
		})
		if err != nil {
			s.l.Warn("failed to load carrier for intelligence sync", zap.Error(err))
			return
		}

		plan := carrierintel.PlanCarrierSync(
			entity,
			snapshot.Profile,
			in.control.SyncSettings(),
			s.now(),
		)
		applied := carrierintel.ApplyFieldUpdates(entity, plan.AutoApply)
		insuranceApplied := carrierintel.ApplyInsuranceChanges(entity, plan.InsuranceAutoApply)

		disqualify := in.control.AutoDisqualifyOnBlock && len(newBlocks) > 0 &&
			entity.ComplianceStatus != carrier.ComplianceStatusDisqualified
		if disqualify {
			entity.ComplianceStatus = carrier.ComplianceStatusDisqualified
			entity.DisqualifiedReason = "Carrier intelligence: " + strings.Join(newBlocks, "; ")
		}

		if len(applied) == 0 && insuranceApplied == 0 && !disqualify {
			return
		}

		_, err = s.carrierService.Update(ctx, entity, &services.RequestActor{
			PrincipalType:  services.PrincipalTypeSystem,
			PrincipalID:    services.SystemPrincipalID,
			OrganizationID: in.tenant.OrgID,
			BusinessUnitID: in.tenant.BuID,
		})
		if err == nil {
			return
		}
		if attempt == 0 && isVersionConflict(err) {
			continue
		}
		s.l.Warn("failed to apply carrier intelligence sync", zap.Error(err),
			zap.String("carrierId", in.subject.CarrierID.String()))
		return
	}
}
