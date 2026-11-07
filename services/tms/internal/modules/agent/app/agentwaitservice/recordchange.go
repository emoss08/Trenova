package agentwaitservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/subjectaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentwaitjobs/agentwaitcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// streamedKinds are the kinds the change stream ends.
var streamedKinds = []agentwait.Kind{agentwait.KindRecordChanged, agentwait.KindReportRun}

func NewRecordChangeNotifier(s *Service) agentwaitports.RecordChangeNotifier { return s }

// checkRecord reads the record a RecordChanged wait names, as the person the
// work is for: its kind from its id, their grant to read that kind, and that
// it is the organization's. The wait keeps its kind and label for what it
// says about the record later. A record the person cannot read is refused
// the way a missing one is, so a wait never confirms what they may not see.
func (s *Service) checkRecord(
	ctx context.Context,
	actor *permissionports.RequestActor,
	wait *agentwait.Wait,
) error {
	c := wait.Condition
	prefix, _, cut := strings.Cut(c.RecordID.String(), "_")
	kind, known := recordkind.OfIDPrefix(prefix + "_")
	if !cut || !known {
		return errortypes.NewValidationError("recordId", errortypes.ErrInvalid,
			"recordId does not name a kind of record Trenova keeps. Use the id a tool returned.")
	}
	resource := kind.Resource()
	allowed, err := subjectaccess.MayReadResource(ctx, s.permissions, actor, resource)
	if err != nil {
		return fmt.Errorf("check access to the watched record: %w", err)
	}
	missing := errortypes.NewBusinessError(fmt.Sprintf(
		"No %s with that id is in this organization that the person may read, so there is "+
			"nothing to wait on.", kind.Noun()))
	if !allowed {
		return missing
	}

	found, err := s.records.Find(ctx, tenantOf(wait), resource, c.RecordID)
	if err != nil {
		return fmt.Errorf("read the watched record: %w", err)
	}
	if found.Known && !found.Exists {
		return missing
	}
	c.RecordKind = string(kind)
	c.RecordLabel = found.Label

	return nil
}

// watch registers a streamed wait's record with the change stream for as
// long as the wait is open.
func (s *Service) watch(ctx context.Context, wait *agentwait.Wait) error {
	if !wait.Kind.Streamed() {
		return nil
	}

	return s.feed.Watch(ctx, &realtimeports.WatchRecordRequest{
		OrganizationID: wait.OrganizationID,
		BusinessUnitID: wait.BusinessUnitID,
		RecordID:       wait.WatchID,
		Until:          wait.ExpiresAt,
	})
}

// release takes a closed streamed wait's record off the change stream when
// no other open wait still watches it. The register's own expiry covers a
// release that fails.
func (s *Service) release(ctx context.Context, wait *agentwait.Wait) {
	if wait == nil || !wait.Kind.Streamed() || wait.Status.Open() {
		return
	}
	ctx = context.WithoutCancel(ctx)
	open, err := s.waits.ListOpenWatching(ctx, &agentwaitports.ListOpenWaitsWatchingRequest{
		TenantInfo: tenantOf(wait),
		Kinds:      streamedKinds,
		WatchIDs:   []pulid.ID{wait.WatchID},
	})
	if err != nil {
		s.l.Debug("could not read whether another wait watches the record; it stays "+
			"registered until it expires", zap.String("wait", wait.ID.String()), zap.Error(err))
		return
	}
	if slices.ContainsFunc(open, func(other *agentwait.Wait) bool { return other.ID != wait.ID }) {
		return
	}
	if err = s.feed.Unwatch(ctx, &realtimeports.WatchRecordRequest{
		OrganizationID: wait.OrganizationID,
		BusinessUnitID: wait.BusinessUnitID,
		RecordID:       wait.WatchID,
		Until:          wait.ExpiresAt,
	}); err != nil {
		s.l.Debug("could not take a record off the change stream; it drops off when it "+
			"expires", zap.String("wait", wait.ID.String()), zap.Error(err))
	}
}

type tenantChanges struct {
	tenant  pagination.TenantInfo
	changes []realtimeports.WatchedRecordChange
}

// NotifyRecordChanges hands changes read off the change stream to the waits
// watching their records: one read of the open waits per tenant in the
// batch, and the names of whoever made them in one more.
func (s *Service) NotifyRecordChanges(
	ctx context.Context,
	changes []realtimeports.TenantRecordChange,
) {
	for _, group := range byTenant(changes) {
		s.notifyTenant(ctx, group)
	}
}

func byTenant(changes []realtimeports.TenantRecordChange) []*tenantChanges {
	groups := make(map[pagination.TenantInfo]*tenantChanges, 1)
	order := make([]*tenantChanges, 0, 1)
	for idx := range changes {
		change := &changes[idx]
		tenant := pagination.TenantInfo{OrgID: change.OrganizationID, BuID: change.BusinessUnitID}
		group, seen := groups[tenant]
		if !seen {
			group = &tenantChanges{tenant: tenant}
			groups[tenant] = group
			order = append(order, group)
		}
		group.changes = append(group.changes, change.Change)
	}

	return order
}

func (s *Service) notifyTenant(ctx context.Context, group *tenantChanges) {
	ctx = dbscope.WithValidTenant(context.WithoutCancel(ctx), group.tenant.DBTenant())
	ids := make([]pulid.ID, 0, len(group.changes))
	for idx := range group.changes {
		id, err := pulid.Parse(group.changes[idx].RecordID)
		if err == nil && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}

	waits, err := s.waits.ListOpenWatching(ctx, &agentwaitports.ListOpenWaitsWatchingRequest{
		TenantInfo: group.tenant,
		Kinds:      streamedKinds,
		WatchIDs:   ids,
	})
	if err != nil {
		s.l.Warn("could not read the waits watching changed records", zap.Error(err))
		return
	}
	if len(waits) == 0 {
		return
	}

	names := s.changerNames(ctx, group, waits)
	for _, wait := range waits {
		for idx := range group.changes {
			change := &group.changes[idx]
			if change.RecordID != wait.WatchID.String() {
				continue
			}
			switch wait.Kind { //nolint:exhaustive // the streamed kinds
			case agentwait.KindRecordChanged:
				s.recordChanged(ctx, wait, change, names)
			case agentwait.KindReportRun:
				s.reportRunChanged(ctx, wait)
			}
		}
	}
}

// changerNames are the names of the people other than each wait's owner who
// made the changes, so the work hears who changed the record.
func (s *Service) changerNames(
	ctx context.Context,
	group *tenantChanges,
	waits []*agentwait.Wait,
) map[string]string {
	owners := make(map[string]struct{}, len(waits))
	for _, wait := range waits {
		owners[wait.UserID.String()] = struct{}{}
	}
	people := make([]pulid.ID, 0, len(group.changes))
	for idx := range group.changes {
		change := &group.changes[idx]
		if change.ActorType != string(permissionports.PrincipalTypeUser) {
			continue
		}
		if _, owner := owners[change.ActorUserID]; owner {
			continue
		}
		if id, err := pulid.Parse(change.ActorUserID); err == nil {
			people = append(people, id)
		}
	}
	if len(people) == 0 {
		return nil
	}

	found, err := s.records.UserNames(ctx, group.tenant, people)
	if err != nil {
		s.l.Debug("could not read who changed a watched record", zap.Error(err))
		return nil
	}
	names := make(map[string]string, len(found))
	for id, name := range found {
		names[id.String()] = name
	}

	return names
}

// recordChanged tells a RecordChanged wait its record changed. A change from
// before the wait was set is what the agent already read.
func (s *Service) recordChanged(
	ctx context.Context,
	wait *agentwait.Wait,
	change *realtimeports.WatchedRecordChange,
	names map[string]string,
) {
	if change.At > 0 && change.At < wait.CreatedAt {
		return
	}
	described := *change
	described.Label = watchedLabel(wait.Condition)
	owner := &permissionports.RequestActor{UserID: wait.UserID}
	detail := fmt.Sprintf("%s Seen at %s.",
		worldwatch.DescribeChange(&described, owner, names), stamp(change.At))

	s.meetWith(ctx, wait, agentwaitcontract.Met{
		Detail: detail,
		At:     change.At,
		Key:    change.RecordID,
	})
}

// watchedLabel names the watched record as the wait was told it: its kind
// and label, or its kind alone.
func watchedLabel(c *agentwait.Condition) string {
	noun := recordkind.Kind(c.RecordKind).Noun()
	switch {
	case c.RecordLabel != "" && noun != "":
		return noun + " " + c.RecordLabel
	case c.RecordLabel != "":
		return c.RecordLabel
	case noun != "":
		return noun + " " + c.RecordID.String()
	default:
		return c.RecordID.String()
	}
}

// conversationTookItIn reports whether the conversation's owner spoke in it
// after the first change a settled wait heard: a person who fixed the record
// and said so has already had their answer, and a second turn would only
// repeat it.
func (s *Service) conversationTookItIn(
	ctx context.Context,
	wait *agentwait.Wait,
	firstMetAt int64,
) bool {
	if firstMetAt <= 0 || !wait.Conversational() || wait.Kind != agentwait.KindRecordChanged {
		return false
	}
	spoke, err := s.activity.PersonSpokeSince(ctx, &agentwaitports.PersonSpokeSinceRequest{
		ThreadID:   wait.ThreadID,
		TenantInfo: tenantOf(wait),
		Since:      firstMetAt,
	})
	if err != nil {
		s.l.Debug("could not read whether the person spoke since the change; picking the "+
			"work up", zap.String("wait", wait.ID.String()), zap.Error(err))
		return false
	}

	return spoke
}
