package deskattentionservice

import (
	"cmp"
	"context"
	"slices"

	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/modules/agent/app/deskrecords"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desklens"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultGroups = 50
	maxGroups     = 100
	window        = 200
)

var queueStatuses = []watchtower.Status{
	watchtower.StatusOpen,
	watchtower.StatusAcknowledged,
	watchtower.StatusInProgress,
}

type Params struct {
	fx.In

	Watchtower watchtowerports.WatchtowerService
	Labels     *deskrecords.Labeler
	Owners     caseownerports.Service
}

type Service struct {
	watchtower watchtowerports.WatchtowerService
	labels     *deskrecords.Labeler
	owners     caseownerports.Service
}

func New(p Params) deskattentionports.DeskAttentionService {
	return &Service{watchtower: p.Watchtower, labels: p.Labels, owners: p.Owners}
}

func (s *Service) Attention(
	ctx context.Context,
	req *deskattentionports.AttentionRequest,
	actor *permissionports.RequestActor,
) (*deskattentionports.Attention, error) {
	lens := req.Lens
	if lens == "" {
		lens = desklens.Everything
	}
	if !lens.IsValid() {
		return nil, errortypes.NewValidationError("lens", errortypes.ErrInvalid, "Unknown lens")
	}
	first := req.First
	if first <= 0 {
		first = defaultGroups
	}
	first = min(first, maxGroups)

	counts, err := s.watchtower.CountsFor(ctx, req.TenantInfo, actor, req.Mine)
	if err != nil {
		return nil, err
	}
	sight, err := s.owners.Sight(ctx, actor)
	if err != nil {
		return nil, err
	}
	out := &deskattentionports.Attention{
		Groups:       []*deskattentionports.Group{},
		Counts:       lensCounts(counts),
		Kinds:        kindCounts(counts, req.Kinds),
		SeenAt:       counts.SeenAt,
		SeesEveryone: sight.All,
	}

	var items []*watchtower.Item
	if len(req.RecordIDs) > 0 {
		page, lErr := s.list(ctx, req, actor, narrowing{about: req.RecordIDs})
		if lErr != nil {
			return nil, lErr
		}
		items = page.Items
	} else {
		kinds := lens.Narrow(req.Kinds)
		if len(req.Kinds) > 0 && len(kinds) == 0 {
			return out, nil
		}
		page, lErr := s.list(ctx, req, actor, narrowing{
			kinds:      kinds,
			severities: req.Severities,
			working:    req.Working,
			search:     req.Search,
			unseen:     req.Unseen,
		})
		if lErr != nil {
			return nil, lErr
		}
		var keys []pulid.ID
		keys, out.HasMore = firstGroups(page.Items, first)
		out.HasMore = out.HasMore || page.HasNextPage
		items = keep(page.Items, keys)

		about := aboutIDs(items)
		if len(about) > 0 {
			filled, fErr := s.list(ctx, req, actor, narrowing{about: about})
			if fErr != nil {
				return nil, fErr
			}
			items = merge(items, filled.Items)
		}
	}

	out.Groups = group(items)
	records := make([]*deskattentionports.Record, 0, len(out.Groups))
	for _, g := range out.Groups {
		records = append(records, g.Record)
	}
	if err = s.labels.Label(ctx, req.TenantInfo, actor, records); err != nil {
		return nil, err
	}
	if err = s.own(ctx, req, actor, out.Groups); err != nil {
		return nil, err
	}

	return out, nil
}

// own says whose case each row's record is, for every row in one read, so the
// queue can show a case someone has and send the reader to it.
func (s *Service) own(
	ctx context.Context,
	req *deskattentionports.AttentionRequest,
	actor *permissionports.RequestActor,
	groups []*deskattentionports.Group,
) error {
	ids := make([]pulid.ID, 0, len(groups))
	for _, g := range groups {
		if g.Record != nil {
			ids = append(ids, g.Record.ID)
		}
	}
	standings, err := s.owners.Standings(ctx, caseownerports.RecordsRequest{
		TenantInfo: req.TenantInfo,
		RecordIDs:  ids,
	}, actor)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if g.Record == nil {
			continue
		}
		if standing, ok := standings[g.Record.ID]; ok {
			ownership := standing.Ownership()
			g.Ownership = &ownership
		}
	}

	return nil
}

type narrowing struct {
	kinds      []watchtower.SourceKind
	about      []pulid.ID
	severities []watchtower.Severity
	working    bool
	search     string
	unseen     bool
}

func (s *Service) list(
	ctx context.Context,
	req *deskattentionports.AttentionRequest,
	actor *permissionports.RequestActor,
	by narrowing,
) (*watchtowerports.WatchtowerPage, error) {
	return s.watchtower.List(ctx, watchtowerports.ListWatchtowerItemsRequest{
		AgentWorking:   by.working,
		TenantInfo:     req.TenantInfo,
		Kinds:          by.kinds,
		Severities:     by.severities,
		UnresolvedOnly: true,
		Statuses:       queueStatuses,
		AboutIDs:       by.about,
		OrderByDue:     true,
		First:          window,
		Mine:           req.Mine,
		Search:         by.search,
		UnseenOnly:     by.unseen,
	}, actor)
}

func kindCounts(
	counts *watchtowerports.WatchtowerCounts,
	asked []watchtower.SourceKind,
) []deskattentionports.KindCount {
	open := make(map[watchtower.SourceKind]int, len(counts.ByKind)+len(asked))
	for _, kind := range asked {
		if kind.IsValid() {
			open[kind] = 0
		}
	}
	for kind, n := range counts.ByKind {
		if n > 0 {
			open[kind] = n
		}
	}
	out := make([]deskattentionports.KindCount, 0, len(open))
	for kind, n := range open {
		out = append(out, deskattentionports.KindCount{
			Kind:  kind,
			Label: kind.Label(),
			Lens:  desklens.Of(kind),
			Count: n,
		})
	}
	slices.SortFunc(out, func(a, b deskattentionports.KindCount) int {
		if c := cmp.Compare(b.Count, a.Count); c != 0 {
			return c
		}
		return cmp.Compare(a.Label, b.Label)
	})

	return out
}

func lensCounts(counts *watchtowerports.WatchtowerCounts) deskattentionports.LensCounts {
	out := deskattentionports.LensCounts{}
	for kind, n := range counts.ByKind {
		out.Everything += n
		switch desklens.Of(kind) {
		case desklens.Decisions:
			out.Decisions += n
		case desklens.Triage:
			out.Triage += n
		default:
			out.Watchtower += n
		}
	}

	return out
}

func firstGroups(items []*watchtower.Item, first int) ([]pulid.ID, bool) {
	keys := make([]pulid.ID, 0, first)
	seen := make(map[pulid.ID]struct{}, first)
	for _, item := range items {
		key := item.GroupKey()
		if _, ok := seen[key]; ok {
			continue
		}
		if len(keys) == first {
			return keys, true
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	return keys, false
}

func keep(items []*watchtower.Item, keys []pulid.ID) []*watchtower.Item {
	wanted := make(map[pulid.ID]struct{}, len(keys))
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	out := make([]*watchtower.Item, 0, len(items))
	for _, item := range items {
		if _, ok := wanted[item.GroupKey()]; ok {
			out = append(out, item)
		}
	}

	return out
}

func aboutIDs(items []*watchtower.Item) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	seen := make(map[pulid.ID]struct{}, len(items))
	for _, item := range items {
		if item.AboutID.IsNil() {
			continue
		}
		if _, ok := seen[item.AboutID]; ok {
			continue
		}
		seen[item.AboutID] = struct{}{}
		ids = append(ids, item.AboutID)
	}

	return ids
}

func merge(items, more []*watchtower.Item) []*watchtower.Item {
	have := make(map[pulid.ID]struct{}, len(items))
	for _, item := range items {
		have[item.ID] = struct{}{}
	}
	for _, item := range more {
		if _, ok := have[item.ID]; ok {
			continue
		}
		have[item.ID] = struct{}{}
		items = append(items, item)
	}

	return items
}

func group(items []*watchtower.Item) []*deskattentionports.Group {
	ordered := slices.Clone(items)
	slices.SortStableFunc(ordered, func(a, b *watchtower.Item) int {
		if c := cmp.Compare(a.DueSortKey(), b.DueSortKey()); c != 0 {
			return c
		}
		return cmp.Compare(a.ID.String(), b.ID.String())
	})

	groups := make([]*deskattentionports.Group, 0, len(ordered))
	byKey := make(map[pulid.ID]*deskattentionports.Group, len(ordered))
	for _, item := range ordered {
		key := item.GroupKey()
		g, ok := byKey[key]
		if !ok {
			g = &deskattentionports.Group{Key: key, Severity: item.Severity}
			if item.AboutID.IsNotNil() && item.AboutKind.Resource() != "" {
				g.Record = &deskattentionports.Record{Kind: item.AboutKind, ID: item.AboutID}
			}
			byKey[key] = g
			groups = append(groups, g)
		}
		g.Items = append(g.Items, item)
		if item.Severity.Rank() > g.Severity.Rank() {
			g.Severity = item.Severity
		}
		if item.DueAt != nil && (g.DueAt == nil || *item.DueAt < *g.DueAt) {
			due := *item.DueAt
			g.DueAt = &due
			g.DueLabel = item.DueLabel
		}
		if !item.Seen {
			g.Unseen = true
		}
	}

	return groups
}
