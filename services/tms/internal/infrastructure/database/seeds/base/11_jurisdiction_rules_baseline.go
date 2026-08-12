package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

// The baseline reuses the federal constants the domain already defines rather
// than restating them. 23 CFR 658.15 fixes width on the National Network at 102
// inches and 23 U.S.C. 127 fixes the Interstate gross at 80,000 lb; a load
// beyond either needs a permit in every jurisdiction, which is enough for the
// engine to do its job.
//
// Height and length are NOT federally fixed for this purpose — states commonly
// set 13'6" to 14'. The domain's federal constants sit at the low end of that
// range, which is the safe direction: a limit set too high lets an illegal load
// dispatch unflagged, while one set too low raises a requirement an operator can
// check against the statute and clear. Over-flagging is recoverable;
// under-flagging is a load stopped at a scale.

const baselineSourceNote = "Federal baseline only: 102in width (23 CFR 658.15) and " +
	"80,000lb Interstate gross (23 U.S.C. 127). Height and length are the conservative " +
	"lower bound of common state limits, not this state's statute. Permit lead time, " +
	"validity, fees, superload thresholds, escort requirements and travel curfews are " +
	"NOT populated. Confirm against the state permit office and mark this row Verified " +
	"before relying on it."

const baselineSourceURL = "https://www.fhwa.dot.gov/policy/otps/truck/"

type JurisdictionRulesBaselineSeed struct {
	seedhelpers.BaseSeed
}

func NewJurisdictionRulesBaselineSeed() *JurisdictionRulesBaselineSeed {
	seed := &JurisdictionRulesBaselineSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"JurisdictionRulesBaseline",
		"1.0.0",
		"Creates an unverified federal-baseline oversize rule per US state",
		[]common.Environment{
			common.EnvProduction,
			common.EnvStaging,
			common.EnvDevelopment,
			common.EnvTest,
		},
	)
	seed.SetDependencies(seedhelpers.SeedUSStates)
	return seed
}

// Run inserts one Unverified rule per state.
//
// Every row is Unverified on purpose. The permit panel already reports
// unverified jurisdictions to the operator, and a requirement derived from one
// carries that provenance, so the baseline makes the engine functional while
// still telling anyone reading it that the numbers are not legal authority.
//
// Fields that genuinely vary by state — lead time, validity, fees, superload
// thresholds, curfews — are left at their schema defaults or NULL rather than
// invented. A fabricated fee produces a confident wrong cost estimate, and a
// fabricated curfew produces a confident wrong departure time.
func (s *JurisdictionRulesBaselineSeed) Run(ctx context.Context, tx bun.Tx) error {
	var existing int
	err := tx.NewSelect().
		Model((*jurisdictionrule.JurisdictionRule)(nil)).
		ColumnExpr("count(*)").
		Scan(ctx, &existing)
	if err != nil {
		return err
	}

	// Never overwrite. Once a carrier has verified or corrected rows, re-running
	// the seed must not quietly reset them to the unverified baseline.
	if existing > 0 {
		return nil
	}

	var states []usstate.UsState
	if err = tx.NewSelect().Model(&states).Scan(ctx); err != nil {
		return err
	}

	if len(states) == 0 {
		return nil
	}

	rules := make([]jurisdictionrule.JurisdictionRule, 0, len(states))
	for i := range states {
		rules = append(rules, jurisdictionrule.JurisdictionRule{
			StateID:           states[i].ID,
			Status:            jurisdictionrule.StatusActive,
			MaxWidthFeet:      jurisdictionrule.FederalMaxWidthFeet,
			MaxHeightFeet:     jurisdictionrule.FederalMaxHeightFeet,
			MaxLengthFeet:     jurisdictionrule.FederalMaxLengthFeet,
			MaxWeightPounds:   jurisdictionrule.FederalMaxWeightPounds,
			SourceNote:        baselineSourceNote,
			SourceURL:         baselineSourceURL,
			VerificationState: jurisdictionrule.VerificationUnverified,
		})
	}

	_, err = tx.NewInsert().Model(&rules).Exec(ctx)

	return err
}
