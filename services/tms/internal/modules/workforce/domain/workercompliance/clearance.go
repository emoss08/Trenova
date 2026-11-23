package workercompliance

import (
	"sort"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpage"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const ClearanceItemCap = 60

type ClearanceLane string

const (
	ClearanceLaneLicense           = ClearanceLane("License")
	ClearanceLaneMedical           = ClearanceLane("Medical")
	ClearanceLaneReviewsAndQueries = ClearanceLane("ReviewsAndQueries")
	ClearanceLaneTraining          = ClearanceLane("Training")
)

func (l ClearanceLane) String() string { return string(l) }

type ClearanceSource string

const (
	ClearanceSourceCredential         = ClearanceSource("Credential")
	ClearanceSourceTraining           = ClearanceSource("Training")
	ClearanceSourceClearinghouseQuery = ClearanceSource("ClearinghouseQuery")
	ClearanceSourcePerformanceReview  = ClearanceSource("PerformanceReview")
)

func (s ClearanceSource) String() string { return string(s) }

type ClearanceEffect string

const (
	ClearanceEffectStopsAssignment = ClearanceEffect("StopsAssignment")
	ClearanceEffectDue             = ClearanceEffect("Due")
)

func (e ClearanceEffect) String() string { return string(e) }

type ClearanceRule string

const (
	ClearanceRuleRequiredCredential = ClearanceRule("RequiredCredential")
	ClearanceRuleDispatchCompliance = ClearanceRule("DispatchCompliance")
)

func (r ClearanceRule) String() string { return string(r) }

type ClearanceBlock string

const (
	ClearanceBlockNotEmployed               = ClearanceBlock("NotEmployed")
	ClearanceBlockDispatchHold              = ClearanceBlock("DispatchHold")
	ClearanceBlockRequiredCredentialLapsed  = ClearanceBlock("RequiredCredentialLapsed")
	ClearanceBlockRequiredCredentialMissing = ClearanceBlock("RequiredCredentialMissing")
	ClearanceBlockDispatchCompliance        = ClearanceBlock("DispatchCompliance")
)

func (b ClearanceBlock) String() string { return string(b) }

type ClearanceItem struct {
	Key         string
	Lane        ClearanceLane
	Source      ClearanceSource
	Name        string
	DueAt       int64
	DaysUntil   int64
	Effect      ClearanceEffect
	Rule        *ClearanceRule
	HazmatOnly  bool
	Regulation  *string
	Required    bool
	Destination workerpage.Destination
}

type Clearance struct {
	Cleared         bool
	ClearedThrough  *int64
	NotClearedSince *int64
	Blocks          []ClearanceBlock
	FirstStop       *ClearanceItem
	Items           []*ClearanceItem
	ItemsTruncated  bool
}

const (
	clearanceKeyCredential    = "credential:"
	clearanceKeyTraining      = "training:"
	clearanceKeyClearinghouse = "clearinghouse"
	clearanceKeyReview        = "review"

	clearinghouseQueryName = "Clearinghouse query"
	performanceReviewName  = "Performance review"
)

type ClearanceInput struct {
	Worker      *worker.Worker
	Credentials *workercredential.WorkerCredentialSummary
	Training    *workertraining.WorkerTrainingSummary
	DrugAlcohol *workerdot.DrugAlcoholStanding
	ReviewDueAt *int64
	Dispatch    *DispatchVerdicts
	Now         int64
}

type ruling struct {
	stops      bool
	rule       *ClearanceRule
	hazmatOnly bool
	regulation *string
}

func (r ruling) effect() ClearanceEffect {
	if r.stops {
		return ClearanceEffectStopsAssignment
	}
	return ClearanceEffectDue
}

func (r ruling) stopsAll() bool { return r.stops && !r.hazmatOnly }

func ruleOf(rule ClearanceRule) *ClearanceRule { return &rule }

func dispatchRuling(stop DispatchStop) ruling {
	out := ruling{
		stops:      true,
		rule:       ruleOf(ClearanceRuleDispatchCompliance),
		hazmatOnly: stop.HazmatOnly,
	}
	if stop.Regulation != "" {
		regulation := stop.Regulation
		out.regulation = &regulation
	}
	return out
}

func credentialRuling(
	item *workercredential.CredentialSummaryItem,
	dispatch *DispatchVerdicts,
) ruling {
	if item.BlocksWhenLapsed() {
		return ruling{stops: true, rule: ruleOf(ClearanceRuleRequiredCredential)}
	}
	if probe, ok := credentialProbe(item); ok {
		if stop, found := dispatch.StopAt(probe); found {
			return dispatchRuling(stop)
		}
	}
	return ruling{}
}

func clearinghouseRuling(due int64, dispatch *DispatchVerdicts) ruling {
	stop, found := dispatch.StopAt(DispatchProbe{
		Requirement: DispatchRequirementClearinghouseQuery,
		At:          due,
	})
	if !found {
		return ruling{}
	}
	return dispatchRuling(stop)
}

func clearinghouseDue(standing *workerdot.DrugAlcoholStanding) *int64 {
	if standing == nil || standing.NextClearinghouseQueryDue == nil ||
		*standing.NextClearinghouseQueryDue <= 0 {
		return nil
	}
	return standing.NextClearinghouseQueryDue
}

func credentialLane(category workercredential.CredentialCategory) ClearanceLane {
	switch category {
	case workercredential.CredentialCategoryMedical:
		return ClearanceLaneMedical
	case workercredential.CredentialCategoryBackground:
		return ClearanceLaneReviewsAndQueries
	case workercredential.CredentialCategoryLicense,
		workercredential.CredentialCategoryEndorsement,
		workercredential.CredentialCategorySecurity,
		workercredential.CredentialCategoryCertification,
		workercredential.CredentialCategoryOther:
		return ClearanceLaneLicense
	default:
		return ClearanceLaneLicense
	}
}

func credentialDestination(item *workercredential.CredentialSummaryItem) workerpage.Destination {
	dest := workerpage.AtView(workerpage.TabQualification, workerpage.ViewCredentials)
	if item.Credential != nil {
		dest = dest.WithRecord(workerpage.RecordCredential, item.Credential.ID)
	}
	return dest
}

func trainingDestination(item *workertraining.TrainingSummaryItem) workerpage.Destination {
	dest := workerpage.AtView(workerpage.TabDevelopment, workerpage.ViewTraining)
	if item.Record != nil {
		dest = dest.WithRecord(workerpage.RecordTraining, item.Record.ID)
	}
	return dest
}

func trainingDueAt(item *workertraining.TrainingSummaryItem) *int64 {
	if item == nil || item.Record == nil {
		return nil
	}
	if item.Record.IsOpen() {
		if item.Record.DueAt != nil && *item.Record.DueAt > 0 {
			return item.Record.DueAt
		}
		return nil
	}
	if item.Record.ExpiresAt != nil && *item.Record.ExpiresAt > 0 {
		return item.Record.ExpiresAt
	}
	return nil
}

func BuildClearance(in ClearanceInput) *Clearance {
	if in.Worker == nil || in.Credentials == nil {
		return nil
	}

	clearance := &Clearance{
		Blocks: clearanceBlocks(in),
		Items:  clearanceItems(in),
	}
	clearance.Cleared = len(clearance.Blocks) == 0

	for _, item := range clearance.Items {
		if item.Effect != ClearanceEffectStopsAssignment || item.HazmatOnly {
			continue
		}
		if clearance.FirstStop == nil {
			clearance.FirstStop = item
		}
		if item.DueAt < in.Now {
			if !clearance.Cleared && clearance.NotClearedSince == nil {
				since := item.DueAt
				clearance.NotClearedSince = &since
			}
			continue
		}
		if clearance.Cleared && clearance.ClearedThrough == nil {
			through := item.DueAt
			clearance.ClearedThrough = &through
		}
	}

	if len(clearance.Items) > ClearanceItemCap {
		clearance.Items = clearance.Items[:ClearanceItemCap]
		clearance.ItemsTruncated = true
	}
	return clearance
}

func clearanceBlocks(in ClearanceInput) []ClearanceBlock {
	blocks := make([]ClearanceBlock, 0, 4)
	wrk := in.Worker
	if wrk.Status != domaintypes.StatusActive {
		blocks = append(blocks, ClearanceBlockNotEmployed)
	}
	if !wrk.CanBeAssigned || !wrk.AvailableForDispatch || wrk.AssignmentBlocked != "" ||
		in.Dispatch.RefusedOutOfSight() {
		blocks = append(blocks, ClearanceBlockDispatchHold)
	}
	lapsed, missing := false, false
	for _, item := range in.Credentials.Items {
		if item == nil || !item.BlocksWhenLapsed() {
			continue
		}
		switch item.Health {
		case workercredential.CredentialHealthExpired:
			lapsed = true
		case workercredential.CredentialHealthMissing:
			missing = true
		case workercredential.CredentialHealthValid,
			workercredential.CredentialHealthExpiringSoon:
		}
	}
	if lapsed {
		blocks = append(blocks, ClearanceBlockRequiredCredentialLapsed)
	}
	if missing {
		blocks = append(blocks, ClearanceBlockRequiredCredentialMissing)
	}
	if in.Dispatch.Refused() {
		blocks = append(blocks, ClearanceBlockDispatchCompliance)
	}
	return blocks
}

func clearanceItems(in ClearanceInput) []*ClearanceItem {
	items := make([]*ClearanceItem, 0, len(in.Credentials.Items)+8)
	for _, entry := range in.Credentials.Items {
		if item := credentialClearanceItem(entry, in); item != nil {
			items = append(items, item)
		}
	}
	if in.Training != nil {
		for _, entry := range in.Training.Items {
			if item := trainingClearanceItem(entry, in.Now); item != nil {
				items = append(items, item)
			}
		}
	}
	if due := clearinghouseDue(in.DrugAlcohol); due != nil {
		r := clearinghouseRuling(*due, in.Dispatch)
		items = append(items, &ClearanceItem{
			Key:        clearanceKeyClearinghouse,
			Lane:       ClearanceLaneReviewsAndQueries,
			Source:     ClearanceSourceClearinghouseQuery,
			Name:       clearinghouseQueryName,
			DueAt:      *due,
			DaysUntil:  workercredential.DaysUntil(*due, in.Now),
			Effect:     r.effect(),
			Rule:       r.rule,
			HazmatOnly: r.hazmatOnly,
			Regulation: r.regulation,
			Required:   true,
			Destination: workerpage.AtView(
				workerpage.TabQualification,
				workerpage.ViewDrugAndAlcohol,
			),
		})
	}
	if in.ReviewDueAt != nil && *in.ReviewDueAt > 0 {
		due := *in.ReviewDueAt
		items = append(items, &ClearanceItem{
			Key:         clearanceKeyReview,
			Lane:        ClearanceLaneReviewsAndQueries,
			Source:      ClearanceSourcePerformanceReview,
			Name:        performanceReviewName,
			DueAt:       due,
			DaysUntil:   workercredential.DaysUntil(due, in.Now),
			Effect:      ClearanceEffectDue,
			Destination: workerpage.AtView(workerpage.TabDevelopment, workerpage.ViewReviews),
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].DueAt != items[j].DueAt {
			return items[i].DueAt < items[j].DueAt
		}
		return items[i].Key < items[j].Key
	})
	return items
}

func credentialClearanceItem(
	entry *workercredential.CredentialSummaryItem,
	in ClearanceInput,
) *ClearanceItem {
	if entry == nil || entry.CredentialType == nil || entry.Credential == nil ||
		entry.Credential.ExpiresAt == nil || *entry.Credential.ExpiresAt <= 0 {
		return nil
	}
	due := *entry.Credential.ExpiresAt
	r := credentialRuling(entry, in.Dispatch)
	return &ClearanceItem{
		Key:         clearanceKeyCredential + entry.CredentialType.ID.String(),
		Lane:        credentialLane(entry.CredentialType.Category),
		Source:      ClearanceSourceCredential,
		Name:        entry.CredentialType.Name,
		DueAt:       due,
		DaysUntil:   workercredential.DaysUntil(due, in.Now),
		Effect:      r.effect(),
		Rule:        r.rule,
		HazmatOnly:  r.hazmatOnly,
		Regulation:  r.regulation,
		Required:    entry.Required,
		Destination: credentialDestination(entry),
	}
}

func trainingClearanceItem(entry *workertraining.TrainingSummaryItem, now int64) *ClearanceItem {
	due := trainingDueAt(entry)
	if due == nil || entry.Course == nil {
		return nil
	}
	return &ClearanceItem{
		Key:         clearanceKeyTraining + entry.Course.ID.String(),
		Lane:        ClearanceLaneTraining,
		Source:      ClearanceSourceTraining,
		Name:        entry.Course.Name,
		DueAt:       *due,
		DaysUntil:   workercredential.DaysUntil(*due, now),
		Effect:      ClearanceEffectDue,
		Required:    entry.Required,
		Destination: trainingDestination(entry),
	}
}
