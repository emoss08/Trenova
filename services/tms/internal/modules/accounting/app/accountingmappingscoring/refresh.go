package accountingmappingscoring

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingscore"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingmapping"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
)

const (
	maxModelTargets        = 100
	MaxRefreshErrorLength  = 1000
	maxReferencePagesGuard = 500
)

var errReferenceUnsupported = errors.New("this accounting system cannot share its reference data")

func tallyRow(
	result *accountingmappingports.AccountingRescoreResult,
	row *accountingmapping.AccountingMapping,
) {
	if row.State == accountingmapping.MappingStateProposed {
		result.Proposed++
	}
	if row.NeedsModelReview() && len(result.NeedsModel) < maxModelTargets {
		result.NeedsModel = append(result.NeedsModel, row.ID)
	}
}

type referenceSet struct {
	byKind  map[accountingprovider.ReferenceKind][]*accountingmapping.AccountingReferenceObject
	indexes map[accountingprovider.ReferenceKind]*accountingmappingscore.TokenIndex
}

type rescoreOutcome int

const (
	rescoreUnchanged rescoreOutcome = iota
	rescoreChanged
	rescoreRelabeled
)

func (r *referenceSet) rescore(
	row *accountingmapping.AccountingMapping,
	t *accountingmappingscore.Target,
) rescoreOutcome {
	labelChanged := row.TargetLabel != t.Label
	row.TargetLabel = t.Label
	proposalChanged := false
	if row.Rescorable() {
		kind := t.Kind
		proposal := accountingmappingscore.Score(t, r.byKind[kind], r.indexes[kind])
		if !row.KeepsModelPick(proposal) {
			proposalChanged = row.ApplyProposal(proposal)
		}
	}
	switch {
	case proposalChanged || (labelChanged && row.State != accountingmapping.MappingStateConfirmed):
		return rescoreChanged
	case labelChanged:
		return rescoreRelabeled
	default:
		return rescoreUnchanged
	}
}
