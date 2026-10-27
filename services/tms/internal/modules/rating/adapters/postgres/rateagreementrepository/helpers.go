package rateagreementrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
)

const busyMessage = "Rate agreement is busy. Retry the request."

var (
	agreementErrors = dberror.Mapping{Entity: "RateAgreement"}
	ruleErrors      = dberror.Mapping{Entity: "RateAgreementRule"}
	versionErrors   = dberror.Mapping{Entity: "RateAgreementVersion"}
)

func mapAgreementError(err error, id string) error {
	return dberror.MapRetryableTransactionError(agreementErrors.Map(err, id), busyMessage)
}

type ruleWindowColumns struct {
	from keel.Int64Col
	to   keel.Int64Col
}

func ruleWindow() ruleWindowColumns {
	r := rateagreementdb.RateAgreementRules
	return ruleWindowColumns{from: r.EffectiveFrom, to: r.EffectiveTo}
}

func openAt(to keel.Int64Col, moment int64) keel.Pred {
	return keel.Or(to.IsNull(), to.Gt(moment))
}

func effectiveAt(cols ruleWindowColumns, moment int64) keel.Pred {
	return keel.And(cols.from.Lte(moment), openAt(cols.to, moment))
}

// ruleWindowAt narrows a rule set to the moment asked for. A zero moment
// loads every rule, which is what an editor showing the history of a lane
// needs; a rating only ever wants the live ones.
func ruleWindowAt(cols ruleWindowColumns, asOf int64) keel.Pred {
	if asOf == 0 {
		return nil
	}

	return effectiveAt(cols, asOf)
}

func liveLanes(r *rateagreementdb.RateAgreementRulesTable, now int64) keel.Pred {
	return keel.And(r.Status.Eq(rateagreement.RuleStatusActive), openAt(r.EffectiveTo, now))
}

func stampAccessorials(entity *rateagreement.RateAgreement, resetIDs bool) {
	for _, accessorial := range entity.Accessorials {
		if accessorial == nil {
			continue
		}

		if resetIDs {
			accessorial.ID = pulid.Nil
		}

		accessorial.RateAgreementID = entity.ID
		accessorial.OrganizationID = entity.OrganizationID
		accessorial.BusinessUnitID = entity.BusinessUnitID
	}
}
