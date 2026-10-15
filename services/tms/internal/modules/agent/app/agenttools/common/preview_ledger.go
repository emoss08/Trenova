package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ManualJournalPlan struct {
	Before   *manualjournal.Request      `json:"-"`
	After    *manualjournal.Request      `json:"-"`
	Journal  *kernelports.JournalPreview `json:"-"`
	Accounts map[pulid.ID]string         `json:"-"`
}

type ReversalPlan struct {
	Change   *journalreversalports.JournalReversalChange `json:"-"`
	Accounts map[pulid.ID]string                         `json:"-"`
}

func JournalLinesOf(entity *manualjournal.Request) []kernelports.JournalLinePreview {
	if entity == nil {
		return nil
	}
	lines := make([]kernelports.JournalLinePreview, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, kernelports.JournalLinePreview{
			GLAccountID: line.GLAccountID,
			Description: line.Description,
			DebitMinor:  line.DebitAmount,
			CreditMinor: line.CreditAmount,
		})
	}

	return lines
}

func AccountLabels(
	ctx context.Context,
	accounts LedgerAccounts,
	tenant pagination.TenantInfo,
	sets ...[]kernelports.JournalLinePreview,
) (map[pulid.ID]string, error) {
	seen := make(map[pulid.ID]struct{})
	ids := make([]pulid.ID, 0)
	for _, lines := range sets {
		for _, line := range lines {
			if line.GLAccountID.IsNil() {
				continue
			}
			if _, dup := seen[line.GLAccountID]; dup {
				continue
			}
			seen[line.GLAccountID] = struct{}{}
			ids = append(ids, line.GLAccountID)
		}
	}

	labels := make(map[pulid.ID]string, len(ids))
	if accounts == nil || len(ids) == 0 {
		return labels, nil
	}

	found, err := accounts.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
		TenantInfo:   tenant,
		GLAccountIDs: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("read the accounts the journal posts to: %w", err)
	}
	for _, account := range found {
		if account != nil {
			labels[account.ID] = strings.TrimSpace(account.AccountCode + " " + account.Name)
		}
	}

	return labels, nil
}

func PlanManualJournalDraft(
	ctx context.Context,
	accounts LedgerAccounts,
	tenant pagination.TenantInfo,
	planned *manualjournal.Request,
) (*ManualJournalPlan, error) {
	labels, err := AccountLabels(ctx, accounts, tenant, JournalLinesOf(planned))
	if err != nil {
		return nil, err
	}

	return &ManualJournalPlan{After: planned, Accounts: labels}, nil
}

func PlanManualJournalChange(
	ctx context.Context,
	accounts LedgerAccounts,
	tenant pagination.TenantInfo,
	change *manualjournalports.ManualJournalChange,
) (*ManualJournalPlan, error) {
	sets := [][]kernelports.JournalLinePreview{
		JournalLinesOf(change.Before),
		JournalLinesOf(change.After),
	}
	if change.Journal != nil {
		sets = append(sets, change.Journal.Lines)
	}
	labels, err := AccountLabels(ctx, accounts, tenant, sets...)
	if err != nil {
		return nil, err
	}

	return &ManualJournalPlan{
		Before:   change.Before,
		After:    change.After,
		Journal:  change.Journal,
		Accounts: labels,
	}, nil
}

func PlanReversal(
	ctx context.Context,
	accounts LedgerAccounts,
	tenant pagination.TenantInfo,
	change *journalreversalports.JournalReversalChange,
) (*ReversalPlan, error) {
	plan := &ReversalPlan{Change: change, Accounts: map[pulid.ID]string{}}
	if change.Journal == nil {
		return plan, nil
	}
	labels, err := AccountLabels(ctx, accounts, tenant, change.Journal.Lines)
	if err != nil {
		return nil, err
	}
	plan.Accounts = labels

	return plan, nil
}
