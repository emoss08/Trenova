package detentionsync

import (
	"context"
	"errors"
	"slices"
	"sort"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const syncNow int64 = 400_000

type candidatePolicyRepo struct {
	detentionports.DetentionPolicyRepository

	policies []*detentionpolicy.DetentionPolicy
	requests []*detentionports.GetResolutionCandidatesRequest
}

func (r *candidatePolicyRepo) GetResolutionCandidates(
	_ context.Context,
	req *detentionports.GetResolutionCandidatesRequest,
) ([]*detentionpolicy.DetentionPolicy, error) {
	r.requests = append(r.requests, req)

	out := make([]*detentionpolicy.DetentionPolicy, 0, len(r.policies))
	for _, policy := range r.policies {
		if policy.Status != detentionpolicy.PolicyStatusActive {
			continue
		}
		if !scopedTo(policy.CustomerID, req.CustomerID) {
			continue
		}
		if policy.LocationID != nil && !slices.Contains(req.LocationIDs, *policy.LocationID) {
			continue
		}
		if policy.EffectiveStartDate != nil && *policy.EffectiveStartDate > req.ArrivedTo {
			continue
		}
		if policy.EffectiveEndDate != nil && *policy.EffectiveEndDate < req.ArrivedFrom {
			continue
		}
		out = append(out, policy)
	}

	return orderLikeTheIndex(out), nil
}

// perStopCandidates is the read the sync made for every stop before it read
// once per shipment.
func perStopCandidates(
	policies []*detentionpolicy.DetentionPolicy,
	customerID, locationID pulid.ID,
) []*detentionpolicy.DetentionPolicy {
	out := make([]*detentionpolicy.DetentionPolicy, 0, len(policies))
	for _, policy := range policies {
		if policy.Status != detentionpolicy.PolicyStatusActive {
			continue
		}
		if !scopedTo(policy.CustomerID, customerID) || !scopedTo(policy.LocationID, locationID) {
			continue
		}
		out = append(out, policy)
	}

	return orderLikeTheIndex(out)
}

func scopedTo(scope *pulid.ID, id pulid.ID) bool {
	return scope == nil || *scope == id
}

func orderLikeTheIndex(
	policies []*detentionpolicy.DetentionPolicy,
) []*detentionpolicy.DetentionPolicy {
	sort.SliceStable(policies, func(a, b int) bool {
		if policies[a].Priority != policies[b].Priority {
			return policies[a].Priority > policies[b].Priority
		}
		if policies[a].SpecificityScore != policies[b].SpecificityScore {
			return policies[a].SpecificityScore > policies[b].SpecificityScore
		}
		return policies[a].CreatedAt < policies[b].CreatedAt
	})

	return policies
}

type recordingOccurrenceRepo struct {
	detentionports.DetentionOccurrenceRepository

	stored   []*detention.DetentionOccurrence
	requests []*detentionports.ListOccurrencesByStopsRequest
	upserts  []*detention.DetentionOccurrence
	created  []pulid.ID
}

func (r *recordingOccurrenceRepo) ListByStops(
	_ context.Context,
	req *detentionports.ListOccurrencesByStopsRequest,
) ([]*detention.DetentionOccurrence, error) {
	r.requests = append(r.requests, req)

	out := make([]*detention.DetentionOccurrence, 0, len(r.stored))
	for _, occurrence := range r.stored {
		if slices.Contains(req.StopIDs, occurrence.StopID) {
			out = append(out, occurrence)
		}
	}

	return out, nil
}

func (r *recordingOccurrenceRepo) Upsert(
	_ context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("do_")
		r.created = append(r.created, entity.StopID)
	}
	r.upserts = append(r.upserts, entity)

	return entity, nil
}

func (r *recordingOccurrenceRepo) winnerByStop() map[pulid.ID]pulid.ID {
	out := make(map[pulid.ID]pulid.ID, len(r.upserts))
	for _, occurrence := range r.upserts {
		out[occurrence.StopID] = *occurrence.DetentionPolicyID
	}

	return out
}

type acceptingEvidenceRepo struct {
	detentionports.DetentionEvidenceRepository
}

func (acceptingEvidenceRepo) Append(
	_ context.Context,
	req *detentionports.AppendEvidenceRequest,
) (*detentionevidence.DetentionEvidence, error) {
	return req.Entry, nil
}

// hourlyAccessorialRepo answers only the batched read: a per-stop GetByID
// would reach the nil embedded interface and panic.
type hourlyAccessorialRepo struct {
	accessorialchargeports.AccessorialChargeRepository

	requests []accessorialchargeports.GetAccessorialChargesByIDsRequest
}

func (r *hourlyAccessorialRepo) GetByIDs(
	_ context.Context,
	req accessorialchargeports.GetAccessorialChargesByIDsRequest,
) ([]*accessorialcharge.AccessorialCharge, error) {
	r.requests = append(r.requests, req)

	out := make([]*accessorialcharge.AccessorialCharge, 0, len(req.AccessorialChargeIDs))
	for _, id := range req.AccessorialChargeIDs {
		out = append(out, &accessorialcharge.AccessorialCharge{
			ID:       id,
			Code:     "DET",
			Method:   accessorialcharge.MethodPerUnit,
			RateUnit: accessorialcharge.RateUnitHour,
			Amount:   decimal.NewFromInt(75),
		})
	}

	return out, nil
}

type acceptingCommentRepo struct {
	shipmentcommentports.ShipmentCommentRepository
}

func (acceptingCommentRepo) Create(
	_ context.Context,
	entity *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	return entity, nil
}

type utcOrganizations struct {
	tenantports.OrganizationRepository
}

func (utcOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "UTC"}, nil
}

type syncRig struct {
	svc          *Service
	policies     *candidatePolicyRepo
	occurrences  *recordingOccurrenceRepo
	accessorials *hourlyAccessorialRepo
}

func newSyncRig(policies ...*detentionpolicy.DetentionPolicy) *syncRig {
	clock := func() int64 { return syncNow }
	policyRepo := &candidatePolicyRepo{policies: policies}
	occurrenceRepo := &recordingOccurrenceRepo{}
	accessorialRepo := &hourlyAccessorialRepo{}

	return &syncRig{
		svc: &Service{
			l:               zap.NewNop(),
			policyRepo:      policyRepo,
			occurrenceRepo:  occurrenceRepo,
			evidenceRepo:    acceptingEvidenceRepo{},
			accessorialRepo: accessorialRepo,
			commentRepo:     acceptingCommentRepo{},
			watchtower:      silentProjector{},
			publisher:       &recordingPublisher{},
			now:             clock,
			effects: detentioneffects.New(detentioneffects.Params{
				Logger:        zap.NewNop(),
				Organizations: utcOrganizations{},
				Now:           clock,
			}),
		},
		policies:     policyRepo,
		occurrences:  occurrenceRepo,
		accessorials: accessorialRepo,
	}
}

func syncPolicy(
	code string,
	createdAt int64,
	mutate func(*detentionpolicy.DetentionPolicy),
) *detentionpolicy.DetentionPolicy {
	p := &detentionpolicy.DetentionPolicy{
		ID:                      pulid.MustNew("dtp_"),
		Code:                    code,
		Name:                    code,
		Status:                  detentionpolicy.PolicyStatusActive,
		ClockStartBasis:         detentionpolicy.ClockStartBasisArrival,
		LateArrivalRule:         detentionpolicy.LateArrivalRuleNoEffect,
		RoundingMode:            detentionpolicy.RoundingModeUp,
		BillingIncrementMinutes: 15,
		BillingFreeMinutes:      120,
		RateSource:              detentionpolicy.RateSourceAccessorial,
		AccessorialChargeID:     pulid.MustNew("acc_"),
		NotificationRequirement: detentionpolicy.NotificationRequirementNone,
		UnnotifiedBehavior:      detentionpolicy.UnnotifiedBehaviorBill,
		DayBoundaryMode:         detentionpolicy.CapScopePerStop,
		Currency:                "USD",
		CreatedAt:               createdAt,
	}
	if mutate != nil {
		mutate(p)
	}
	p.SpecificityScore = p.ComputeSpecificity()

	return p
}

func idPtr(id pulid.ID) *pulid.ID { return &id }

func int64Ptr(v int64) *int64 { return &v }

func arrivedStop(
	locationID pulid.ID,
	stopType shipmentvocab.StopType,
	arrival int64,
) *shipment.Stop {
	departure := arrival + 5*60*60

	return &shipment.Stop{
		ID:              pulid.MustNew("stp_"),
		LocationID:      locationID,
		Status:          shipmentvocab.StopStatusCompleted,
		Type:            stopType,
		ScheduleType:    shipmentvocab.StopScheduleTypeOpen,
		ActualArrival:   int64Ptr(arrival),
		ActualDeparture: int64Ptr(departure),
	}
}

type syncScenario struct {
	tenant     pagination.TenantInfo
	customerID pulid.ID
	shipment   *shipment.Shipment
	policies   []*detentionpolicy.DetentionPolicy
	defaultID  pulid.ID
	billable   []*shipment.Stop
	locations  []pulid.ID
	want       map[pulid.ID]string
}

func newSyncScenario() *syncScenario {
	tenantInfo := testTenant()
	customerID := pulid.MustNew("cus_")
	shipper := pulid.MustNew("loc_")
	consignee := pulid.MustNew("loc_")
	unvisited := pulid.MustNew("loc_")

	orgDefault := syncPolicy("DEFAULT", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.Priority = 50
	})
	customerTerms := syncPolicy("CUSTOMER", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = idPtr(customerID)
	})
	shipperTerms := syncPolicy("SHIPPER", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.LocationID = idPtr(shipper)
	})
	shipperPromo := syncPolicy("SHIPPER-PROMO", 2000, func(p *detentionpolicy.DetentionPolicy) {
		p.LocationID = idPtr(shipper)
		p.Priority = 10
		p.EffectiveEndDate = int64Ptr(150_000)
	})
	otherCustomer := syncPolicy("OTHER-CUSTOMER", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.CustomerID = idPtr(pulid.MustNew("cus_"))
		p.Priority = 90
	})
	unvisitedFacility := syncPolicy("UNVISITED", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.LocationID = idPtr(unvisited)
		p.Priority = 90
	})
	expired := syncPolicy("EXPIRED", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.Priority = 90
		p.EffectiveEndDate = int64Ptr(50_000)
	})
	notYet := syncPolicy("NOT-YET", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.Priority = 90
		p.EffectiveStartDate = int64Ptr(500_000)
	})
	draft := syncPolicy("DRAFT", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.Priority = 90
		p.Status = detentionpolicy.PolicyStatusDraft
	})

	pickup := arrivedStop(shipper, shipmentvocab.StopTypePickup, 100_000)
	delivery := arrivedStop(consignee, shipmentvocab.StopTypeDelivery, 200_000)
	secondPickup := arrivedStop(shipper, shipmentvocab.StopTypePickup, 300_000)

	canceled := arrivedStop(unvisited, shipmentvocab.StopTypeDelivery, 250_000)
	canceled.Status = shipmentvocab.StopStatusCanceled
	notArrived := arrivedStop(unvisited, shipmentvocab.StopTypeDelivery, 0)
	notArrived.ActualArrival = nil

	entity := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CustomerID:     customerID,
		ServiceTypeID:  pulid.MustNew("svt_"),
		ShipmentTypeID: pulid.MustNew("spt_"),
		Moves: []*shipment.ShipmentMove{
			{ID: pulid.MustNew("smv_"), Stops: []*shipment.Stop{pickup, delivery, canceled}},
			nil,
			{ID: pulid.MustNew("smv_"), Stops: []*shipment.Stop{notArrived, secondPickup}},
		},
	}

	return &syncScenario{
		tenant:     tenantInfo,
		customerID: customerID,
		shipment:   entity,
		policies: []*detentionpolicy.DetentionPolicy{
			orgDefault, customerTerms, shipperTerms, shipperPromo, otherCustomer,
			unvisitedFacility, expired, notYet, draft,
		},
		defaultID: orgDefault.ID,
		billable:  []*shipment.Stop{pickup, delivery, secondPickup},
		locations: []pulid.ID{shipper, consignee},
		want: map[pulid.ID]string{
			pickup.ID:       "SHIPPER-PROMO",
			delivery.ID:     "CUSTOMER",
			secondPickup.ID: "SHIPPER",
		},
	}
}

func TestSyncShipment_ReadsCandidatePoliciesOncePerShipment(t *testing.T) {
	t.Parallel()

	scenario := newSyncScenario()
	rig := newSyncRig(scenario.policies...)

	result, err := rig.svc.SyncShipment(t.Context(), scenario.shipment, scenario.defaultID)
	require.NoError(t, err)
	require.Len(t, result.Occurrences, len(scenario.billable))

	require.Len(t, rig.policies.requests, 1, "one candidate read per shipment")
	req := rig.policies.requests[0]
	require.Equal(t, scenario.customerID, req.CustomerID)
	require.Equal(t, scenario.tenant, req.TenantInfo)
	require.Equal(t, scenario.locations, req.LocationIDs,
		"each billable facility once, canceled and unarrived stops left out")
	require.Equal(t, int64(100_000), req.ArrivedFrom)
	require.Equal(t, int64(300_000), req.ArrivedTo)
}

func TestSyncShipment_ResolvesEveryStopAsThePerStopReadDid(t *testing.T) {
	t.Parallel()

	scenario := newSyncScenario()
	rig := newSyncRig(scenario.policies...)

	_, err := rig.svc.SyncShipment(t.Context(), scenario.shipment, scenario.defaultID)
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(scenario.policies))
	for _, policy := range scenario.policies {
		codes[policy.ID] = policy.Code
	}

	winners := rig.occurrences.winnerByStop()
	require.Len(t, winners, len(scenario.billable))

	commodityIDs := shipmentCommodityIDs(scenario.shipment)
	for _, stop := range scenario.billable {
		in := &detentionpolicy.MatchInput{
			CustomerID:     scenario.shipment.CustomerID,
			LocationID:     stop.LocationID,
			ShipmentTypeID: scenario.shipment.ShipmentTypeID,
			ServiceTypeID:  scenario.shipment.ServiceTypeID,
			CommodityIDs:   commodityIDs,
			StopType:       stop.Type,
			ScheduleType:   stop.ScheduleType,
			Timestamp:      *stop.ActualArrival,
		}
		before := detentionengine.ResolvePolicy(
			perStopCandidates(scenario.policies, scenario.customerID, stop.LocationID),
			in,
			scenario.defaultID,
		)
		require.True(t, before.Found())

		require.Equal(t, scenario.want[stop.ID], codes[winners[stop.ID]])
		require.Equal(t, before.Policy.ID, winners[stop.ID],
			"stop %s resolved differently from the per-stop read", scenario.want[stop.ID])
	}
}

func TestCandidatesForStop_MatchesThePerStopReadsExplanation(t *testing.T) {
	t.Parallel()

	scenario := newSyncScenario()
	rig := newSyncRig(scenario.policies...)

	req, ok := resolutionCandidatesRequest(scenario.shipment, scenario.tenant)
	require.True(t, ok)
	shared, err := rig.policies.GetResolutionCandidates(t.Context(), req)
	require.NoError(t, err)

	for _, stop := range scenario.billable {
		in := &detentionpolicy.MatchInput{
			CustomerID: scenario.customerID,
			LocationID: stop.LocationID,
			StopType:   stop.Type,
			Timestamp:  *stop.ActualArrival,
		}
		before := detentionengine.ResolvePolicy(
			perStopCandidates(scenario.policies, scenario.customerID, stop.LocationID),
			in,
			scenario.defaultID,
		)
		after := detentionengine.ResolvePolicy(
			candidatesForStop(shared, stop.LocationID),
			in,
			scenario.defaultID,
		)

		require.Equal(t, before.Policy.ID, after.Policy.ID)
		require.Equal(t, before.Explanation, after.Explanation)
	}
}

func TestSyncShipment_DefaultPolicyGovernsWhatNothingElseCovers(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenant()
	orgDefault := syncPolicy("DEFAULT", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.Priority = -5
	})
	lowerPriority := syncPolicy("BROAD", 1000, func(p *detentionpolicy.DetentionPolicy) {
		p.Priority = -10
	})
	stop := arrivedStop(pulid.MustNew("loc_"), shipmentvocab.StopTypeDelivery, 100_000)
	entity := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CustomerID:     pulid.MustNew("cus_"),
		Moves:          []*shipment.ShipmentMove{{Stops: []*shipment.Stop{stop}}},
	}

	rig := newSyncRig(orgDefault, lowerPriority)
	_, err := rig.svc.SyncShipment(t.Context(), entity, orgDefault.ID)
	require.NoError(t, err)
	require.Equal(t, lowerPriority.ID, rig.occurrences.winnerByStop()[stop.ID],
		"the default ranks below every other matching policy")

	alone := newSyncRig(orgDefault)
	_, err = alone.svc.SyncShipment(t.Context(), entity, orgDefault.ID)
	require.NoError(t, err)
	require.Equal(t, orgDefault.ID, alone.occurrences.winnerByStop()[stop.ID])
}

func TestSyncShipment_SkipsTheReadWhenNoStopIsBillable(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenant()
	waiting := arrivedStop(pulid.MustNew("loc_"), shipmentvocab.StopTypePickup, 0)
	waiting.ActualArrival = nil
	entity := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		CustomerID:     pulid.MustNew("cus_"),
		Moves:          []*shipment.ShipmentMove{{Stops: []*shipment.Stop{waiting}}},
	}

	rig := newSyncRig(syncPolicy("DEFAULT", 1000, nil))
	result, err := rig.svc.SyncShipment(t.Context(), entity, pulid.Nil)
	require.NoError(t, err)
	require.Empty(t, result.Occurrences)
	require.Empty(t, rig.policies.requests)
	require.Empty(t, rig.occurrences.requests)
	require.Empty(t, rig.accessorials.requests)
}

func TestSyncShipment_ReadsOccurrencesAndAccessorialsOncePerShipment(t *testing.T) {
	t.Parallel()

	scenario := newSyncScenario()
	pickup, delivery, secondPickup := scenario.billable[0], scenario.billable[1], scenario.billable[2]

	recalculated := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		OrganizationID: scenario.tenant.OrgID,
		BusinessUnitID: scenario.tenant.BuID,
		StopID:         pickup.ID,
		Status:         detention.OccurrenceStatusAccruing,
		BillableAmount: decimal.NewFromInt(10),
		Version:        4,
		CreatedAt:      12_345,
	}
	billed := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		OrganizationID: scenario.tenant.OrgID,
		BusinessUnitID: scenario.tenant.BuID,
		StopID:         secondPickup.ID,
		Status:         detention.OccurrenceStatusBilled,
		BillableAmount: decimal.NewFromInt(90),
		Version:        2,
	}

	rig := newSyncRig(scenario.policies...)
	rig.occurrences.stored = []*detention.DetentionOccurrence{recalculated, billed}

	result, err := rig.svc.SyncShipment(t.Context(), scenario.shipment, scenario.defaultID)
	require.NoError(t, err)

	require.Len(t, rig.occurrences.requests, 1, "one occurrence read per shipment")
	require.Equal(t, scenario.tenant, rig.occurrences.requests[0].TenantInfo)
	require.Equal(t,
		[]pulid.ID{pickup.ID, delivery.ID, secondPickup.ID},
		rig.occurrences.requests[0].StopIDs,
		"every billable stop, canceled and unarrived stops left out")

	codes := make(map[string]*detentionpolicy.DetentionPolicy, len(scenario.policies))
	for _, policy := range scenario.policies {
		codes[policy.Code] = policy
	}

	require.Len(t, rig.accessorials.requests, 1, "one accessorial read per shipment")
	require.Equal(t, scenario.tenant, rig.accessorials.requests[0].TenantInfo)
	require.Equal(t,
		[]pulid.ID{
			codes[scenario.want[pickup.ID]].AccessorialChargeID,
			codes[scenario.want[delivery.ID]].AccessorialChargeID,
		},
		rig.accessorials.requests[0].AccessorialChargeIDs,
		"the charges of the policies that won, the frozen stop's left out")

	require.Equal(t, 1, result.Skipped)
	require.Len(t, result.Occurrences, 3)
	require.Same(t, billed, result.Occurrences[2], "a frozen occurrence is carried as stored")

	require.Len(t, rig.occurrences.upserts, 2)
	updated := rig.occurrences.upserts[0]
	require.Equal(t, recalculated.ID, updated.ID, "the stored row is updated, not re-read")
	require.Equal(t, recalculated.Version, updated.Version)
	require.Equal(t, recalculated.CreatedAt, updated.CreatedAt)
	require.Equal(t, []pulid.ID{delivery.ID}, rig.occurrences.created)

	for _, occurrence := range rig.occurrences.upserts {
		require.Equal(t, "DET", occurrence.PolicySnapshot.AccessorialCode)
		require.True(t, occurrence.PolicySnapshot.FlatRate.Equal(decimal.NewFromInt(75)))
	}

	total := decimal.Zero
	for _, occurrence := range result.Occurrences {
		total = total.Add(occurrence.BillableAmount)
	}
	require.True(t, total.Equal(result.TotalAmount))
}

// recordingPayAssignmentRepo answers only the batched read: a per-move
// GetEffectiveForWorker would reach the nil embedded interface and panic.
type recordingPayAssignmentRepo struct {
	driverpayports.WorkerPayAssignmentRepository

	byWorker map[pulid.ID]*payassignment.WorkerPayAssignment
	err      error
	requests []driverpayports.GetWorkerPayAssignmentsRequest
}

func (r *recordingPayAssignmentRepo) GetEffectiveForWorkers(
	_ context.Context,
	req driverpayports.GetWorkerPayAssignmentsRequest,
) (map[pulid.ID]*payassignment.WorkerPayAssignment, error) {
	r.requests = append(r.requests, req)
	if r.err != nil {
		return nil, r.err
	}

	out := make(map[pulid.ID]*payassignment.WorkerPayAssignment, len(req.WorkerIDs))
	for _, id := range req.WorkerIDs {
		if assignment, ok := r.byWorker[id]; ok {
			out[id] = assignment
		}
	}

	return out, nil
}

func payComponent(
	kind driverpay.ComponentKind,
	rate int64,
	active bool,
) *driverpay.PayProfileComponent {
	return &driverpay.PayProfileComponent{
		ID:       pulid.MustNew("dppc_"),
		Kind:     kind,
		Method:   driverpay.CalcMethodPerHour,
		Rate:     decimal.NewFromInt(rate),
		IsActive: active,
	}
}

func payAssignment(
	workerID pulid.ID,
	overrides []payassignment.RateOverride,
	components ...*driverpay.PayProfileComponent,
) *payassignment.WorkerPayAssignment {
	return &payassignment.WorkerPayAssignment{
		ID:            pulid.MustNew("wpa_"),
		WorkerID:      workerID,
		RateOverrides: overrides,
		PayProfile:    &driverpay.PayProfile{ID: pulid.MustNew("dpp_"), Components: components},
	}
}

func assignTo(move *shipment.ShipmentMove, workerID pulid.ID) {
	move.Assignment = &shipment.Assignment{PrimaryWorkerID: idPtr(workerID)}
}

type driverPayScenario struct {
	*syncScenario

	overridden pulid.ID
	standard   pulid.ID
	rates      map[pulid.ID]decimal.Decimal
}

// newDriverPayScenario puts a driver on each move of the sync scenario: the
// first move's driver has an override on the detention component, the
// second's is paid the profile rate. Two more moves have no billable stop:
// one's driver must not be read, the other's is the first driver again and
// is read once.
func newDriverPayScenario() (*driverPayScenario, *recordingPayAssignmentRepo) {
	base := newSyncScenario()
	overridden := pulid.MustNew("wrk_")
	standard := pulid.MustNew("wrk_")
	idle := pulid.MustNew("wrk_")

	moves := base.shipment.Moves
	assignTo(moves[0], overridden)
	assignTo(moves[2], standard)

	canceled := arrivedStop(pulid.MustNew("loc_"), shipmentvocab.StopTypePickup, 150_000)
	canceled.Status = shipmentvocab.StopStatusCanceled
	idleMove := &shipment.ShipmentMove{
		ID:    pulid.MustNew("smv_"),
		Stops: []*shipment.Stop{canceled},
	}
	assignTo(idleMove, idle)
	idleRepeat := &shipment.ShipmentMove{
		ID:    pulid.MustNew("smv_"),
		Stops: []*shipment.Stop{canceled},
	}
	assignTo(idleRepeat, overridden)
	base.shipment.Moves = append(moves, idleMove, idleRepeat)

	detentionComponent := payComponent(driverpay.ComponentKindDetention, 30, true)
	repo := &recordingPayAssignmentRepo{
		byWorker: map[pulid.ID]*payassignment.WorkerPayAssignment{
			overridden: payAssignment(
				overridden,
				[]payassignment.RateOverride{
					{ComponentID: detentionComponent.ID, Rate: decimal.NewFromInt(40)},
				},
				payComponent(driverpay.ComponentKindDetention, 99, false),
				payComponent(driverpay.ComponentKindLinehaul, 1, true),
				detentionComponent,
			),
			standard: payAssignment(
				standard,
				nil,
				payComponent(driverpay.ComponentKindDetention, 25, true),
			),
			idle: payAssignment(
				idle,
				nil,
				payComponent(driverpay.ComponentKindDetention, 70, true),
			),
		},
	}

	pickup, delivery, secondPickup := base.billable[0], base.billable[1], base.billable[2]

	return &driverPayScenario{
		syncScenario: base,
		overridden:   overridden,
		standard:     standard,
		rates: map[pulid.ID]decimal.Decimal{
			pickup.ID:       decimal.NewFromInt(40),
			delivery.ID:     decimal.NewFromInt(40),
			secondPickup.ID: decimal.NewFromInt(25),
		},
	}, repo
}

func expectedDriverPay(rate decimal.Decimal, minutes int32) decimal.Decimal {
	hours := decimal.NewFromInt(int64(minutes)).Div(decimal.NewFromInt(60)).Round(4)

	return rate.Mul(hours).Round(2)
}

func TestSyncShipment_ReadsDriverPayOncePerShipment(t *testing.T) {
	t.Parallel()

	scenario, payRepo := newDriverPayScenario()
	rig := newSyncRig(scenario.policies...)
	rig.svc.payAssignmentRepo = payRepo

	_, err := rig.svc.SyncShipment(t.Context(), scenario.shipment, scenario.defaultID)
	require.NoError(t, err)

	require.Len(t, payRepo.requests, 1, "one driver pay read per shipment")
	req := payRepo.requests[0]
	require.Equal(t, scenario.tenant, req.TenantInfo)
	require.Equal(t, syncNow, req.AsOf)
	require.Equal(t, []pulid.ID{scenario.overridden, scenario.standard}, req.WorkerIDs,
		"each driver of a move with a billable stop once, idle moves' drivers left out")

	require.Len(t, rig.occurrences.upserts, len(scenario.billable))
	for _, occurrence := range rig.occurrences.upserts {
		rate := scenario.rates[occurrence.StopID]
		require.Positive(t, occurrence.DriverPayMinutes)
		require.True(t, occurrence.DriverPayAmount.Valid)
		require.True(t,
			expectedDriverPay(rate, occurrence.DriverPayMinutes).
				Equal(occurrence.DriverPayAmount.Decimal),
			"stop %s is paid at %s, the override before the profile rate",
			occurrence.StopID, rate)
	}
}

func TestSyncShipment_DriverPayReadFailureLeavesPayOut(t *testing.T) {
	t.Parallel()

	scenario, payRepo := newDriverPayScenario()
	payRepo.err = errors.New("pay read failed")
	rig := newSyncRig(scenario.policies...)
	rig.svc.payAssignmentRepo = payRepo

	result, err := rig.svc.SyncShipment(t.Context(), scenario.shipment, scenario.defaultID)
	require.NoError(t, err)
	require.Len(t, result.Occurrences, len(scenario.billable))
	require.Len(t, payRepo.requests, 1)
	for _, occurrence := range rig.occurrences.upserts {
		require.True(t, occurrence.DriverPayAmount.Decimal.IsZero())
	}
}

func TestSyncShipment_SkipsTheDriverPayReadWithoutDrivers(t *testing.T) {
	t.Parallel()

	scenario := newSyncScenario()
	payRepo := &recordingPayAssignmentRepo{}
	rig := newSyncRig(scenario.policies...)
	rig.svc.payAssignmentRepo = payRepo

	_, err := rig.svc.SyncShipment(t.Context(), scenario.shipment, scenario.defaultID)
	require.NoError(t, err)
	require.Empty(t, payRepo.requests)
}

func TestDetentionPayRate(t *testing.T) {
	t.Parallel()

	component := payComponent(driverpay.ComponentKindDetention, 30, true)
	tests := []struct {
		name       string
		assignment *payassignment.WorkerPayAssignment
		want       decimal.NullDecimal
	}{
		{name: "no assignment"},
		{
			name:       "no profile loaded",
			assignment: &payassignment.WorkerPayAssignment{},
		},
		{
			name: "no active detention component",
			assignment: payAssignment(pulid.MustNew("wrk_"), nil,
				payComponent(driverpay.ComponentKindDetention, 99, false),
				payComponent(driverpay.ComponentKindLayover, 50, true),
			),
		},
		{
			name:       "profile rate",
			assignment: payAssignment(pulid.MustNew("wrk_"), nil, component),
			want:       decimal.NewNullDecimal(decimal.NewFromInt(30)),
		},
		{
			name: "override of the component",
			assignment: payAssignment(pulid.MustNew("wrk_"),
				[]payassignment.RateOverride{
					{ComponentID: pulid.MustNew("dppc_"), Rate: decimal.NewFromInt(5)},
					{ComponentID: component.ID, Rate: decimal.NewFromInt(45)},
				},
				component,
			),
			want: decimal.NewNullDecimal(decimal.NewFromInt(45)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rate, ok := detentionPayRate(tt.assignment)
			require.Equal(t, tt.want.Valid, ok)
			if ok {
				require.True(t, tt.want.Decimal.Equal(rate))
			}
		})
	}
}
