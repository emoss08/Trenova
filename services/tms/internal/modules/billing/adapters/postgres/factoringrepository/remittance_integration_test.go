//go:build integration

package factoringrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRemittanceStagesMatchesAndKeepsItsLines(t *testing.T) {
	e := setupFactoringEnv(t)
	p := Params{DB: postgres.NewTestConnection(e.db), Logger: zap.NewNop()}
	submissions := NewSubmissionRepository(p)
	remittances := NewRemittanceRepository(p)
	now := time.Now().Unix()

	factor := e.createFactor(t, "FREM")
	agreement, err := e.agreements.Create(e.ctx, e.agreement(factor.ID, now-3600, nil))
	require.NoError(t, err)
	first := e.postInvoice(t, 100000, &factor.ID)
	second := e.postInvoice(t, 50000, &factor.ID)
	sub, err := submissions.Create(e.ctx, e.submission(agreement, "FS-R1", "key-r1", first, second))
	require.NoError(t, err)
	require.NoError(t, sub.MarkSent(now-86400, "", 1))
	sub, err = submissions.Update(e.ctx, sub)
	require.NoError(t, err)
	firstItem, secondItem := sub.Items[0], sub.Items[1]

	created, err := remittances.Create(e.ctx, &factoringremittance.Remittance{
		OrganizationID:     e.tenantInfo.OrgID,
		BusinessUnitID:     e.tenantInfo.BuID,
		FactoringCompanyID: factor.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, factoringremittance.RemittanceStatusPending, created.Status)

	exact := &factoringremittance.RemittanceLine{
		RowNumber:     2,
		Kind:          factoringremittance.RemittanceLineFunding,
		InvoiceNumber: " " + firstItem.InvoiceNumber + " ",
		NormalizedInvoice: factoringremittance.NormalizeRemittanceReference(
			firstItem.InvoiceNumber,
		),
		AdvanceMinor: new(firstItem.ExpectedAdvanceMinor),
		OccurredOn:   new(now),
	}
	byAmount := &factoringremittance.RemittanceLine{
		RowNumber:     3,
		Kind:          factoringremittance.RemittanceLineFunding,
		InvoiceNumber: "UNREADABLE",
		AdvanceMinor:  new(secondItem.ExpectedAdvanceMinor),
	}
	require.NoError(t, created.MarkParsed(&factoringremittance.RemittanceStaging{
		DocumentID:  pulid.MustNew("doc_"),
		FileName:    "remit.csv",
		Format:      factoringremittance.RemittanceFormatCsv,
		DefaultKind: factoringremittance.RemittanceLineFunding,
		At:          now,
	}, []*factoringremittance.RemittanceLine{exact, byAmount}))

	candidates, err := remittances.ListCandidates(
		e.ctx,
		&factoringremittanceports.ListRemittanceCandidatesRequest{
			TenantInfo:         e.tenantInfo,
			FactoringCompanyID: factor.ID,
			Invoices:           []string{exact.NormalizedInvoice, "UNREADABLE"},
			Amounts:            []int64{*byAmount.AdvanceMinor},
		},
	)
	require.NoError(t, err)
	ids := make([]pulid.ID, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Item.ID)
		assert.Equal(t, "FS-R1", candidate.SubmissionNumber)
		require.NotNil(t, candidate.SentAt)
		assert.True(t, candidate.Recourse)
	}
	assert.ElementsMatch(t, []pulid.ID{firstItem.ID, secondItem.ID}, ids,
		"one found by its normalized invoice number, the other by its expected advance")

	none, err := remittances.ListCandidates(
		e.ctx,
		&factoringremittanceports.ListRemittanceCandidatesRequest{
			TenantInfo:         e.tenantInfo,
			FactoringCompanyID: pulid.MustNew("fac_"),
			Invoices:           []string{exact.NormalizedInvoice},
		},
	)
	require.NoError(t, err)
	assert.Empty(t, none, "another factor's items are never candidates")

	factoringremittance.MatchRemittance(created.Lines, candidates)
	created.Recount()
	require.NoError(t, remittances.ReplaceLines(e.ctx, created))
	created, err = remittances.Update(e.ctx, created)
	require.NoError(t, err)

	loaded, err := remittances.Get(e.ctx, factoringremittanceports.GetRemittanceRequest{
		TenantInfo:   e.tenantInfo,
		ID:           created.ID,
		IncludeLines: true,
	})
	require.NoError(t, err)
	assert.Equal(t, factoringremittance.RemittanceStatusParsed, loaded.Status)
	assert.Equal(t, 1, loaded.MatchedCount)
	assert.Equal(t, 1, loaded.ExceptionCount)
	require.Len(t, loaded.Lines, 2)
	assert.Equal(t, factoringremittance.RemittanceLineMatched, loaded.Lines[0].MatchStatus)
	assert.Equal(t, firstItem.ID, loaded.Lines[0].SubmissionItemID)
	assert.Equal(t, factoringremittance.RemittanceLineSuggested, loaded.Lines[1].MatchStatus)
	require.NotEmpty(t, loaded.Lines[1].Suggestions)
	assert.Equal(t, secondItem.ID, loaded.Lines[1].Suggestions[0].ItemID)
	assert.Equal(
		t,
		factoringremittance.MatchReasonAmountOnly,
		loaded.Lines[1].Suggestions[0].Reason,
	)

	pending := loaded.Lines[1]
	pending.MatchTo(
		secondItem,
		factoringremittance.RemittanceMatchedByPerson,
		40,
		factoringremittance.MatchReasonChosenByPerson,
	)
	require.NoError(
		t,
		remittances.UpdateLines(e.ctx, []*factoringremittance.RemittanceLine{pending}),
	)

	page, err := remittances.ListLineConnection(
		e.ctx,
		&factoringremittanceports.ListRemittanceLineConnectionRequest{
			TenantInfo:   e.tenantInfo,
			Cursor:       pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
			RemittanceID: created.ID,
			MatchStatuses: []factoringremittance.RemittanceMatchStatus{
				factoringremittance.RemittanceLineMatched,
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 2, *page.TotalCount, "the person's match is stored")

	open, err := remittances.ListOpen(
		e.ctx,
		&factoringremittanceports.ListOpenRemittancesRequest{TenantInfo: e.tenantInfo},
	)
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, created.ID, open[0].RemittanceID)
	assert.Equal(t, 1, open[0].ExceptionCount)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	_, err = remittances.Get(
		dbscope.WithTenant(e.ctx, other.DBTenant()),
		factoringremittanceports.GetRemittanceRequest{TenantInfo: other, ID: created.ID},
	)
	require.Error(t, err)
}

func TestRemittanceProfileIsOnePerFactor(t *testing.T) {
	e := setupFactoringEnv(t)
	remittances := NewRemittanceRepository(
		Params{DB: postgres.NewTestConnection(e.db), Logger: zap.NewNop()},
	)
	factor := e.createFactor(t, "FRMP")

	_, err := remittances.GetProfile(e.ctx, factoringremittanceports.GetRemittanceProfileRequest{
		TenantInfo:         e.tenantInfo,
		FactoringCompanyID: factor.ID,
	})
	require.Error(t, err, "a factor has no saved layout until a person maps one")

	save := func(mapping map[string]int, signature string) {
		require.NoError(t, remittances.SaveProfile(e.ctx, &factoringremittance.RemittanceProfile{
			OrganizationID:     e.tenantInfo.OrgID,
			BusinessUnitID:     e.tenantInfo.BuID,
			FactoringCompanyID: factor.ID,
			Mapping:            mapping,
			HeaderSignature:    signature,
		}))
	}
	save(map[string]int{"invoiceNumber": 0, "cash": 3}, "sig-1")
	save(map[string]int{"invoiceNumber": 1, "advance": 4}, "sig-2")

	profile, err := remittances.GetProfile(
		e.ctx,
		factoringremittanceports.GetRemittanceProfileRequest{
			TenantInfo:         e.tenantInfo,
			FactoringCompanyID: factor.ID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"invoiceNumber": 1, "advance": 4}, profile.Mapping)
	assert.Equal(t, "sig-2", profile.HeaderSignature)
	assert.Equal(t, int64(1), profile.Version)
}

func TestFactorIsFoundBySenderAddress(t *testing.T) {
	e := setupFactoringEnv(t)
	exact := e.createFactor(t, "FMAIL")
	e.createFactor(t, "FOTHR")

	found, err := e.factors.FindByEmail(
		e.ctx,
		factoringcompanyports.FindFactoringCompaniesByEmailRequest{
			TenantInfo: e.tenantInfo,
			Address:    " REMIT@FMAIL.example.com ",
		},
	)
	require.NoError(t, err)
	require.Len(t, found.Exact, 1)
	assert.Equal(t, exact.ID, found.Exact[0].ID)
	assert.Empty(t, found.ByDomain)

	byDomain, err := e.factors.FindByEmail(
		e.ctx,
		factoringcompanyports.FindFactoringCompaniesByEmailRequest{
			TenantInfo: e.tenantInfo,
			Address:    "noreply@fmail.example.com",
		},
	)
	require.NoError(t, err)
	assert.Empty(t, byDomain.Exact)
	require.Len(t, byDomain.ByDomain, 1, "a factor's other addresses share its domain")
	assert.Equal(t, exact.ID, byDomain.ByDomain[0].ID)

	stranger := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	strangerCtx := dbscope.WithTenant(e.ctx, stranger.DBTenant())
	none, err := e.factors.FindByEmail(
		strangerCtx,
		factoringcompanyports.FindFactoringCompaniesByEmailRequest{
			TenantInfo: stranger,
			Address:    "remit@fmail.example.com",
		},
	)
	require.NoError(t, err)
	assert.Empty(t, none.Exact, "another tenant's factor is never found")
}
