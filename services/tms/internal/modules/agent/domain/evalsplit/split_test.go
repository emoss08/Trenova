package evalsplit_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/evalsplit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func items(stratum evalsplit.Stratum, n int) []evalsplit.Item {
	out := make([]evalsplit.Item, 0, n)
	for idx := range n {
		out = append(out, evalsplit.Item{Key: fmt.Sprintf("%s-%d", stratum.Desk, idx), Stratum: stratum})
	}

	return out
}

func count(splits []evalsplit.Split, want evalsplit.Split) int {
	n := 0
	for _, split := range splits {
		if split == want {
			n++
		}
	}

	return n
}

func TestHeldOutQuota_IsAQuarterRoundedWithOneForEveryStratumOfTwo(t *testing.T) {
	t.Parallel()

	for members, want := range map[int]int{0: 0, 1: 0, 2: 1, 3: 1, 5: 1, 6: 2, 9: 2, 10: 3, 32: 8} {
		assert.Equalf(t, want, evalsplit.HeldOutQuota(members), "%d members", members)
	}
}

func TestAssign_HoldsOutEachStratumOnItsOwn(t *testing.T) {
	t.Parallel()

	chat := evalsplit.Stratum{Desk: "Dispatch", Mode: evalsplit.Chat}
	unattended := evalsplit.Stratum{Desk: "Dispatch", Mode: evalsplit.Unattended}
	all := append(items(chat, 8), items(unattended, 2)...)
	all = append(all, evalsplit.Item{Key: "lone", Stratum: evalsplit.Stratum{Desk: "EDI"}})

	splits := evalsplit.Assign(all)

	assert.Equal(t, 2, count(splits[:8], evalsplit.HeldOut))
	assert.Equal(t, 1, count(splits[8:10], evalsplit.HeldOut))
	assert.Equal(t, evalsplit.Development, splits[10], "a stratum of one is never held out")
}

func TestAssign_LeavesAdversarialItemsOutOfTheQuota(t *testing.T) {
	t.Parallel()

	stratum := evalsplit.Stratum{Desk: "Intake", Mode: evalsplit.Unattended}
	all := items(stratum, 3)
	all[0].Adversarial = true
	all[1].Adversarial = true

	splits := evalsplit.Assign(all)

	assert.Equal(t, []evalsplit.Split{
		evalsplit.Adversarial, evalsplit.Adversarial, evalsplit.Development,
	}, splits)
}

func TestAssign_DependsOnlyOnTheKeysNotTheirOrder(t *testing.T) {
	t.Parallel()

	stratum := evalsplit.Stratum{Desk: "Billing", Mode: evalsplit.Chat}
	forward := items(stratum, 12)
	backward := slices.Clone(forward)
	slices.Reverse(backward)

	held := func(all []evalsplit.Item) []string {
		ledger := evalsplit.LedgerOf(all, evalsplit.Assign(all))
		return ledger[stratum]
	}

	assert.Equal(t, held(forward), held(backward))
	assert.Len(t, held(forward), 3)
}

func TestLedger_RoundTripsAndNamesWhatMoved(t *testing.T) {
	t.Parallel()

	stratum := evalsplit.Stratum{Desk: "Billing", Mode: evalsplit.Chat}
	recorded := evalsplit.Ledger{stratum: {"a", "b"}}
	raw, err := recorded.Marshal("# header\n")
	require.NoError(t, err)

	parsed, err := evalsplit.ParseLedger(raw)
	require.NoError(t, err)
	assert.True(t, parsed.Diff(recorded).Empty())

	now := evalsplit.Ledger{stratum: {"b", "c"}}
	diff := now.Diff(parsed)
	assert.Equal(t, []string{"c [Billing (chat)]"}, diff.Joined)
	assert.Equal(t, []string{"a [Billing (chat)]"}, diff.Left)
}

func TestLedger_NamesAKeyThatMovedToAnotherStratum(t *testing.T) {
	t.Parallel()

	billing := evalsplit.Stratum{Desk: "Billing", Mode: evalsplit.Chat}
	exceptions := evalsplit.Stratum{Desk: "Billing exceptions", Mode: evalsplit.Unattended}
	recorded := evalsplit.Ledger{billing: {"a"}}
	now := evalsplit.Ledger{exceptions: {"a"}}

	diff := now.Diff(recorded)
	assert.Equal(t, []string{"a [Billing exceptions (unattended)]"}, diff.Joined)
	assert.Equal(t, []string{"a [Billing (chat)]"}, diff.Left)
}
