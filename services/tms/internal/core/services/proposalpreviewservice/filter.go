package proposalpreviewservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// access is what one reader may see: their ceiling per resource and whether
// they may read each resource at all.
type readerAccess struct {
	ceilings services.FieldCeilings
	reads    services.ResourceReadAccess
}

func (s *Service) accessFor(viewer *services.PreviewViewer) readerAccess {
	a := readerAccess{}
	actor := viewerActor(viewer)
	if viewer != nil {
		a.ceilings = viewer.Ceilings
		a.reads = viewer.Reads
	}
	if a.ceilings == nil && actor != nil {
		a.ceilings = fieldsensitivity.NewCeilings(
			s.permissions,
			actor.UserID,
			actor.OrganizationID,
		)
	}
	if a.reads == nil {
		a.reads = fieldsensitivity.NewReadAccess(s.permissions, actor)
	}

	return a
}

func (a readerAccess) ceiling(
	ctx context.Context,
	resource permission.Resource,
) permission.FieldSensitivity {
	if a.ceilings == nil {
		return permission.SensitivityInternal
	}

	return a.ceilings.For(ctx, resource)
}

func (a readerAccess) mayRead(ctx context.Context, resource permission.Resource) bool {
	return a.reads != nil && a.reads.MayRead(ctx, resource)
}

// finish filters a draft for its reader and digests what they are shown.
// Tools read without regard to who is asking, so this is the one place a
// reader's access is applied.
func (s *Service) finish(
	ctx context.Context,
	d *draft,
	a readerAccess,
) (*agent.ProposalPreview, error) {
	preview := d.preview
	filter(ctx, preview, a)

	digest, err := preview.ComputeDigest(d.params)
	if err != nil {
		return nil, err
	}
	preview.Digest = digest

	return preview, nil
}

// filter withholds from a preview what its reader may not see, in place:
//
//   - a record of a resource they may not read, whole, with its label;
//   - a value above their ceiling on the record's resource;
//   - the label of a record a value points at, when they may not read that
//     resource;
//   - an amount above their ceiling.
//
// Confidential values never reached the preview. What is withheld is
// counted, so the reader is told how much they did not see.
func filter(ctx context.Context, preview *agent.ProposalPreview, a readerAccess) {
	withheld := 0
	for i := range preview.Changes {
		withheld += filterChange(ctx, &preview.Changes[i], a)
	}

	preview.WithheldCount = withheld
	if withheld > 0 {
		preview.AddWarning(agent.PreviewWarning{
			Code:    agent.PreviewWarningWithheld,
			Args:    []string{strconv.Itoa(withheld)},
			Message: "Part of this change is hidden by your data access.",
		})
	}
}

func filterChange(ctx context.Context, change *agent.RecordChange, a readerAccess) int {
	if change.Withheld {
		return 1
	}
	if change.Resource != "" && !a.mayRead(ctx, change.Resource) {
		withholdChange(change)

		return 1
	}

	withheld := 0
	ceiling := a.ceiling(ctx, change.Resource)
	for i := range change.Fields {
		withheld += filterField(ctx, &change.Fields[i], ceiling, a)
	}

	if money := change.Money; money != nil {
		if money.Withheld {
			withheld++
		} else if !fieldsensitivity.VisibleAt(levelOf(money.Sensitivity), ceiling) {
			withholdMoney(money)
			withheld++
		}
	}

	return withheld
}

func filterField(
	ctx context.Context,
	field *agent.PreviewFieldChange,
	ceiling permission.FieldSensitivity,
	a readerAccess,
) int {
	if field.Withheld {
		return 1
	}
	if !fieldsensitivity.VisibleAt(levelOf(field.Sensitivity), ceiling) {
		withholdField(field)

		return 1
	}

	return filterRef(ctx, field.BeforeRef, a) + filterRef(ctx, field.AfterRef, a)
}

func filterRef(ctx context.Context, ref *agent.PreviewRef, a readerAccess) int {
	if ref == nil {
		return 0
	}
	if ref.Withheld {
		return 1
	}
	if ref.Resource == "" || a.mayRead(ctx, ref.Resource) {
		return 0
	}

	ref.Label = ""
	ref.Record = nil
	ref.Withheld = true

	return 1
}

// levelOf reads a value with no sensitivity recorded as Internal, never as
// Public.
func levelOf(level permission.FieldSensitivity) permission.FieldSensitivity {
	if level == "" {
		return permission.SensitivityInternal
	}

	return level
}

func withholdChange(change *agent.RecordChange) {
	change.Withheld = true
	change.Label = ""
	change.Record = nil
	change.EntityID = ""
	change.Version = nil
	change.Fields = nil
	change.OmittedFields = 0
	change.Message = nil
	change.Money = nil
}

func withholdField(field *agent.PreviewFieldChange) {
	field.Withheld = true
	field.Before = nil
	field.After = nil
	field.BeforeRef = nil
	field.AfterRef = nil
	field.ProposedBefore = nil
	field.Truncated = false
}

func withholdMoney(money *agent.MoneyPreview) {
	money.Withheld = true
	money.Lines = nil
	money.TotalBefore = decimal.NullDecimal{}
	money.TotalAfter = decimal.NullDecimal{}
	money.Delta = decimal.NullDecimal{}
}

// recorded is a decision's preview as a later reader may see it: what the
// decider was shown, filtered again for this reader, with the digest the
// decision recorded.
func (s *Service) recorded(
	ctx context.Context,
	decision *agent.AgentDecision,
	a readerAccess,
) *agent.ProposalPreview {
	clone := new(agent.ProposalPreview)
	if err := jsonutils.Convert(decision.Preview, clone); err != nil {
		s.l.Warn("a recorded preview could not be read",
			zap.String("decision", decision.ID.String()), zap.Error(err))

		return nil
	}

	clone.Recorded = true
	clone.Warnings = withoutCode(clone.Warnings, agent.PreviewWarningWithheld)
	filter(ctx, clone, a)
	clone.Digest = decision.PreviewDigest

	return clone
}

func withoutCode(
	warnings []agent.PreviewWarning,
	code agent.PreviewWarningCode,
) []agent.PreviewWarning {
	kept := make([]agent.PreviewWarning, 0, len(warnings))
	for i := range warnings {
		if warnings[i].Code != code {
			kept = append(kept, warnings[i])
		}
	}

	return kept
}
