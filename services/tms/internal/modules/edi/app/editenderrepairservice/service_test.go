package editenderrepairservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

const isaHead = "ISA*00*          *00*          *ZZ*ACMEFOODS      *ZZ*TRENOVA        " +
	"*261013*0912*^*00401*000000317*0*P*>~GS*SM*ACMEFOODS*TRENOVA*20261013*0912*317*X*004010~"

type fakeRepo struct {
	pages      [][]*ediports.TenderReferenceCandidate
	afterIDs   []pulid.ID
	repairs    []*ediports.RepairTenderReferenceRequest
	unreadable int
	result     ediports.RepairTenderReferenceResult
}

func (r *fakeRepo) ListTenderReferenceCandidates(
	_ context.Context,
	req *ediports.ListTenderReferenceCandidatesRequest,
) ([]*ediports.TenderReferenceCandidate, error) {
	r.afterIDs = append(r.afterIDs, req.AfterID)
	if len(r.pages) == 0 {
		return nil, nil
	}
	page := r.pages[0]
	r.pages = r.pages[1:]
	return page, nil
}

func (r *fakeRepo) CountUnreadableTenders(
	context.Context,
	*ediports.CountUnreadableTendersRequest,
) (int, error) {
	return r.unreadable, nil
}

func (r *fakeRepo) RepairTenderReference(
	_ context.Context,
	req *ediports.RepairTenderReferenceRequest,
) (*ediports.RepairTenderReferenceResult, error) {
	r.repairs = append(r.repairs, req)
	result := r.result
	return &result, nil
}

type observed struct {
	report   *Report
	repaired []Repair
	skipped  []Skip
}

func run(t *testing.T, repo *fakeRepo, req Request) observed {
	t.Helper()

	var seen observed
	req.OnRepair = func(repair Repair) { seen.repaired = append(seen.repaired, repair) }
	req.OnSkip = func(skip Skip) { seen.skipped = append(seen.skipped, skip) }
	report, err := New(Deps{Repo: repo}).Repair(t.Context(), &req)
	require.NoError(t, err)
	seen.report = report
	return seen
}

func candidate(id, raw, head, shipmentID, payment string) *ediports.TenderReferenceCandidate {
	return &ediports.TenderReferenceCandidate{
		MessageID:      pulid.ID(id),
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		RawX12:         raw,
		FileHead:       head,
		ShipmentID:     shipmentID,
		PaymentMethod:  payment,
	}
}

func TestRepairCorrectsATenderReadOneElementEarly(t *testing.T) {
	t.Parallel()

	misread := candidate(
		"edimsg_1",
		"ST*204*0001~B2**ACME**LT-88240**PP~B2A*00~SE*4*0001~",
		isaHead,
		"ACME",
		"LT-88240",
	)
	repo := &fakeRepo{
		pages:  [][]*ediports.TenderReferenceCandidate{{misread}},
		result: ediports.RepairTenderReferenceResult{Transfers: 1, Shipments: 1},
	}

	seen := run(t, repo, Request{})

	require.Equal(t, []*ediports.RepairTenderReferenceRequest{{
		Candidate:     misread,
		ShipmentID:    "LT-88240",
		PaymentMethod: "PP",
	}}, repo.repairs)
	require.Equal(t, []Repair{{
		MessageID:      "edimsg_1",
		OrganizationID: "org_1",
		From:           Reference{ShipmentID: "ACME", PaymentMethod: "LT-88240"},
		To:             Reference{ShipmentID: "LT-88240", PaymentMethod: "PP"},
	}}, seen.repaired)
	require.Equal(t, 1, seen.report.Repaired)
	require.Equal(t, 1, seen.report.TransfersRepaired)
	require.Equal(t, 1, seen.report.ShipmentsRepaired)
}

func TestRepairLeavesATenderThatWasReadRight(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{pages: [][]*ediports.TenderReferenceCandidate{{
		candidate(
			"edimsg_1",
			"ST*204*0001~B2*ABCD*shp_01JABCDEF**PP~SE*3*0001~",
			isaHead,
			"shp_01JABCDEF",
			"PP",
		),
		candidate(
			"edimsg_2",
			"ST*204*0002~B2**ACME**LT-88240**PP~SE*3*0002~",
			isaHead,
			"LT-88240",
			"PP",
		),
	}}}

	seen := run(t, repo, Request{})

	require.Empty(t, repo.repairs)
	require.Empty(t, seen.repaired)
	require.Zero(t, seen.report.Repaired)
	require.Equal(t, 2, seen.report.Unchanged)
}

func TestRepairReadsAPurgedFilesTransactionWithItsOwnSeparators(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{pages: [][]*ediports.TenderReferenceCandidate{{
		candidate(
			"edimsg_1",
			"ST|204|0001!\nB2||ACME||LT-88240||CC!\nSE|3|0001!\n",
			"",
			"ACME",
			"LT-88240",
		),
	}}}

	seen := run(t, repo, Request{})

	require.Len(t, seen.repaired, 1)
	require.Equal(t, Reference{ShipmentID: "LT-88240", PaymentMethod: "CC"}, seen.repaired[0].To)
}

func TestRepairDryRunListsWithoutWriting(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{pages: [][]*ediports.TenderReferenceCandidate{{
		candidate(
			"edimsg_1",
			"ST*204*0001~B2**ACME**LT-88240**PP~SE*3*0001~",
			isaHead,
			"ACME",
			"LT-88240",
		),
	}}}

	seen := run(t, repo, Request{DryRun: true})

	require.Empty(t, repo.repairs)
	require.Len(t, seen.repaired, 1)
	require.Equal(t, 1, seen.report.Repaired)
	require.Zero(t, seen.report.TransfersRepaired)
}

func TestRepairPagesByMessageAndCountsWhatCannotBeReRead(t *testing.T) {
	t.Parallel()

	read := func(id string) *ediports.TenderReferenceCandidate {
		return candidate(id, "ST*204*0001~B2**ACME**LT-1**PP~SE*3*0001~", isaHead, "LT-1", "PP")
	}
	repo := &fakeRepo{
		pages: [][]*ediports.TenderReferenceCandidate{
			{read("edimsg_1"), read("edimsg_2")},
			{read("edimsg_3")},
		},
		unreadable: 4,
	}

	seen := run(t, repo, Request{BatchSize: 2})

	require.Equal(t, []pulid.ID{"", "edimsg_2", "edimsg_3"}, repo.afterIDs)
	require.Equal(t, 3, seen.report.Unchanged)
	require.Equal(t, 4, seen.report.Unreadable)
}

func TestRepairReadsEveryPageWhenTheStoreReturnsFewerThanAsked(t *testing.T) {
	t.Parallel()

	read := func(id string) *ediports.TenderReferenceCandidate {
		return candidate(id, "ST*204*0001~B2**ACME**LT-1**PP~SE*3*0001~", isaHead, "LT-1", "PP")
	}
	repo := &fakeRepo{pages: [][]*ediports.TenderReferenceCandidate{
		{read("edimsg_1"), read("edimsg_2")},
		{read("edimsg_3"), read("edimsg_4")},
		{read("edimsg_5")},
	}}

	seen := run(t, repo, Request{BatchSize: 5000})

	require.Equal(t, 5, seen.report.Unchanged, "a page shorter than the batch is not the last")
	require.Equal(t, []pulid.ID{"", "edimsg_2", "edimsg_4", "edimsg_5"}, repo.afterIDs)
}

func TestRepairSkipsATransactionWithoutB2(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{pages: [][]*ediports.TenderReferenceCandidate{{
		candidate("edimsg_1", "ST*204*0001~B2A*00~SE*3*0001~", isaHead, "ACME", ""),
	}}}

	seen := run(t, repo, Request{})

	require.Empty(t, repo.repairs)
	require.Equal(t, []Skip{{
		MessageID:      "edimsg_1",
		OrganizationID: "org_1",
		Reason:         "the kept transaction has no B2 segment",
	}}, seen.skipped)
	require.Equal(t, 1, seen.report.Skipped)
}
