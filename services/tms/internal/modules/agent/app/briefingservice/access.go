package briefingservice

import (
	"context"
	"fmt"
	"net/url"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/stringutils"
)

// watchtowerKindsParam is the query parameter a watchtower line's path names
// the kind of item it counts in.
const watchtowerKindsParam = "kinds"

// sectionResources are what a reader must be able to read to see a section:
// the records its figures count. A section with no entry is never shown to a
// gated reader, so a section added without one fails closed.
var sectionResources = map[briefing.SectionKey]permission.Resource{
	briefing.SectionAttention:  permission.ResourceWatchtower,
	briefing.SectionExceptions: permission.ResourceWatchtower,
	briefing.SectionToday:      permission.ResourceShipment,
	briefing.SectionCoverage:   permission.ResourceShipmentMove,
	briefing.SectionDecisions:  permission.ResourceAgentProposal,
	briefing.SectionCompliance: permission.ResourceWorkerCredential,
	briefing.SectionBilling:    permission.ResourceBillingQueue,
	briefing.SectionCash:       permission.ResourceCustomerPayment,
}

// gatedResources are every resource a page's sections and watchtower lines
// can need, asked about in one batch per read.
var gatedResources = func() []permission.Resource {
	out := make([]permission.Resource, 0, len(sectionResources)+len(watchtower.AllSourceKinds()))
	for _, resource := range sectionResources {
		out = append(out, resource)
	}
	for _, kind := range watchtower.AllSourceKinds() {
		if resource := kind.ReadResource(); resource != "" {
			out = append(out, resource)
		}
	}
	slices.Sort(out)

	return slices.Compact(out)
}()

// readAccess is which of gatedResources a reader may read. An ungated one
// reads all, for a caller the service is not asked to gate.
type readAccess struct {
	gated   bool
	allowed map[permission.Resource]bool
}

func (s *Service) readAccess(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (readAccess, error) {
	if s.permissions == nil || actor == nil {
		return readAccess{}, nil
	}

	checks := make([]permissionports.ResourceOperationCheck, 0, len(gatedResources))
	for _, resource := range gatedResources {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  resource.String(),
			Operation: permission.OpRead,
		})
	}
	result, err := s.permissions.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Checks:         checks,
	})
	if err != nil {
		return readAccess{}, fmt.Errorf(
			"check which briefing sections the reader may read: %w",
			err,
		)
	}

	access := readAccess{
		gated:   true,
		allowed: make(map[permission.Resource]bool, len(gatedResources)),
	}
	for idx, resource := range gatedResources {
		access.allowed[resource] = idx < len(result.Results) && result.Results[idx].Allowed
	}

	return access, nil
}

func (a readAccess) mayRead(resource permission.Resource) bool {
	return !a.gated || (resource != "" && a.allowed[resource])
}

// withhold cuts a page to what its reader may read. A section counting
// records they may not read is left out; a watchtower section keeps the
// lines of the kinds they may read, without the wording that counted the
// rest. While anything is withheld, the headline and the model's sentences
// go too, because they were written about the whole page and may cite what
// was cut; the sections left read in their computed wording.
func (a readAccess) withhold(page *briefing.Briefing) {
	if page == nil {
		return
	}

	withheld := make([]briefing.WithheldSection, 0)
	kept := make([]briefing.Section, 0, len(page.Sections))
	for _, section := range page.Sections {
		title := stringutils.FirstNonEmpty(section.Title, section.Key.String())
		if !a.mayRead(sectionResources[section.Key]) {
			withheld = append(withheld, briefing.WithheldSection{Key: section.Key, Title: title})
			continue
		}
		if perKind(section.Key) {
			items := slices.DeleteFunc(slices.Clone(section.Items), func(item briefing.Item) bool {
				return !a.mayRead(watchtowerResource(item.Path))
			})
			if len(items) < len(section.Items) {
				section.Items = items
				section.Summary = ""
				section.Body = ""
				withheld = append(withheld, briefing.WithheldSection{
					Key: section.Key, Title: title, Partial: true,
				})
			}
		}
		kept = append(kept, section)
	}

	if len(withheld) > 0 {
		page.Headline = ""
		page.Narrated = false
		for idx := range kept {
			kept[idx].Body = ""
		}
	}
	page.Sections = kept
	page.WithheldSections = withheld
}

func (a readAccess) withholdAll(pages []*briefing.Briefing) {
	for _, page := range pages {
		a.withhold(page)
	}
}

// perKind reports whether a section's lines each count one kind of
// watchtower item, read under that kind's own resource.
func perKind(key briefing.SectionKey) bool {
	return key == briefing.SectionAttention || key == briefing.SectionExceptions
}

// watchtowerResource is the resource the kind a watchtower line counts is
// read under; empty, and so never readable, when the line names no kind.
func watchtowerResource(path string) permission.Resource {
	parsed, err := url.Parse(path)
	if err != nil {
		return ""
	}
	kind := watchtower.SourceKind(parsed.Query().Get(watchtowerKindsParam))
	if !kind.IsValid() {
		return ""
	}

	return kind.ReadResource()
}
