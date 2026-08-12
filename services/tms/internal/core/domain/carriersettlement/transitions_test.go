package carriersettlement

import "testing"

func TestIsAllowedTransitionMatrix(t *testing.T) {
	allStatuses := []Status{
		StatusDraft, StatusPendingApproval, StatusApproved,
		StatusPosted, StatusPaid, StatusVoided,
	}

	allowed := map[Status]map[Status]bool{
		StatusDraft: {
			StatusPendingApproval: true,
			StatusVoided:          true,
		},
		StatusPendingApproval: {
			StatusDraft:    true,
			StatusApproved: true,
			StatusVoided:   true,
		},
		StatusApproved: {
			StatusPendingApproval: true,
			StatusPosted:          true,
			StatusVoided:          true,
		},
		StatusPosted: {
			StatusPaid:   true,
			StatusVoided: true,
		},
		StatusPaid:   {},
		StatusVoided: {},
	}

	for _, from := range allStatuses {
		for _, to := range allStatuses {
			want := from == to || allowed[from][to]
			if got := IsAllowedTransition(from, to); got != want {
				t.Errorf("IsAllowedTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestIsTerminalStatus(t *testing.T) {
	terminal := map[Status]bool{
		StatusDraft:           false,
		StatusPendingApproval: false,
		StatusApproved:        false,
		StatusPosted:          false,
		StatusPaid:            true,
		StatusVoided:          true,
	}
	for status, want := range terminal {
		if got := IsTerminalStatus(status); got != want {
			t.Errorf("IsTerminalStatus(%s) = %v, want %v", status, got, want)
		}
		if got := status.IsTerminal(); got != want {
			t.Errorf("Status(%s).IsTerminal() = %v, want %v", status, got, want)
		}
	}
}

func TestSyncTotals(t *testing.T) {
	settlement := &CarrierSettlement{
		Lines: []*CarrierSettlementLine{
			{EventType: CostEventTypeLinehaulCost, AmountMinor: 100_000},
			{EventType: CostEventTypeFuelSurcharge, AmountMinor: 12_500},
			{EventType: CostEventTypeAccessorial, AmountMinor: 7_500},
			{EventType: CostEventTypeAdjustment, AmountMinor: -5_000},
		},
	}
	settlement.SyncTotals()

	if settlement.GrossCostMinor != 120_000 {
		t.Errorf("GrossCostMinor = %d, want 120000", settlement.GrossCostMinor)
	}
	if settlement.AdjustmentsMinor != -5_000 {
		t.Errorf("AdjustmentsMinor = %d, want -5000", settlement.AdjustmentsMinor)
	}
	if settlement.NetPayableMinor != 115_000 {
		t.Errorf("NetPayableMinor = %d, want 115000", settlement.NetPayableMinor)
	}
	for idx, line := range settlement.Lines {
		if line.LineNumber != idx+1 {
			t.Errorf("line %d has LineNumber %d", idx, line.LineNumber)
		}
	}
}
