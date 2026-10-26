package ediconnectiondraft

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlan_AHeldOptionalValueIsListedButDoesNotBlock(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	for index := range req.Values {
		if req.Values[index].Field == fieldPartnerContactPhone {
			req.Values[index].Confidence = 0.4
		}
	}

	draft, err := newHarness(t).service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	phone := valueOf(t, draft.Partner.Fields, fieldPartnerContactPhone)
	assert.Equal(t, ediports.ValueStatusHeld, phone.Status)
	assert.Empty(t, phone.Value, "a held value is never written")
	require.Len(t, draft.Decisions, 1, "the person still sees it under To decide")
	assert.Equal(t, fieldPartnerContactPhone, draft.Decisions[0].Field)
	assert.Empty(t, draft.Blockers)
	assert.True(t, draft.Ready)
}

func TestPlan_NeverMatchesAPartnerOnAValueItHolds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{existingAcme(t, req)}
	req.Values = append(req.Values,
		value(fieldPartnerCode, "ACMERETAILAS2", 2, "Acme AS2 Identifier: ACMERETAILAS2"))

	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	code := valueOf(t, draft.Partner.Fields, fieldPartnerCode)
	require.Equal(t, ediports.ValueStatusHeld, code.Status)
	assert.NotContains(t, draft.Partner.Reasons, "Same partner code",
		"a code the draft refuses to use is not evidence of who the partner is")
}

func TestPlan_MatchesAPartnerOnTheValueThePersonPicked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{existingAcme(t, req)}
	req.Values = append(req.Values,
		value(fieldPartnerCode, "ACMERETAILAS2", 2, "Acme AS2 Identifier: ACMERETAILAS2"))
	req.Resolutions = []ediports.DraftResolutionInput{{
		Field: fieldPartnerCode, Value: "ACMERETAIL", Source: ediports.ResolutionSourceCandidate,
	}}

	draft, err := h.service.PlanConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, ediports.ValueStatusAccepted, valueOf(t, draft.Partner.Fields, fieldPartnerCode).Status)
	assert.Contains(t, draft.Partner.Reasons, "Same partner code")
}

// acmeWithOwnGuide is Acme's project with a 204 guide someone wrote by hand in the studio, at a
// path a draft never writes.
func acmeWithOwnGuide(t *testing.T, req *ediports.ConnectionDraftRequest, usage string) *ediports.EDIPartnerSetup {
	t.Helper()
	existing := existingAcme(t, req)
	version := existing.LatestVersion
	files := make([]lading.File, 0, len(version.Files)+1)
	for _, file := range version.Files {
		if strings.HasPrefix(strings.TrimSpace(file.Source), "partner ") {
			file.Source = strings.Replace(file.Source, "usage test", "usage "+usage, 1)
		}
		files = append(files, file)
	}
	files = append(files, lading.File{
		Path:   "guides/acme-tender.lading",
		Source: "guide acmeTender from x12.004010.204 \"Acme's own 204\"\n\nB2 required\n",
	})
	samples, err := versionSamples(version)
	require.NoError(t, err)
	encoded, err := lading.Encode(
		&lading.Project{Files: files, Samples: samples},
		&lading.Context{Org: lading.Org{Name: "Trenova Logistics", SCAC: "TRNV", Zone: "America/Chicago"}},
	)
	require.NoError(t, err)
	check, err := lading.Default().Check(t.Context(), encoded)
	require.NoError(t, err)
	require.Zero(t, check.Errors)
	version.Files = files
	version.Manifest = *check
	return existing
}

func TestPlan_KeepsAGuideWrittenInTheStudio(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{acmeWithOwnGuide(t, req, valueTest)}

	plan, err := h.service.PreviewConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	assert.Contains(t, plan.Draft.Kept, ediports.DraftKept{
		Record: "Guide acmeTender",
		Label:  keptGuideLabel,
	})
	require.NotNil(t, plan.Project)
	require.NotNil(t, plan.Project.Before)
	assert.Equal(t, "guide acmeTender from x12.004010.204 \"Acme's own 204\"\n\nB2 required\n",
		projectFile(t, plan.Project.Files, "guides/acme-tender.lading"),
		"a hand-written guide is never rewritten")
	for _, file := range plan.Project.Files {
		assert.NotEqual(t, "guides/204.lading", file.Path,
			"the 204's requirements are left for the person to add to their own guide")
	}
	assert.Contains(t, projectFile(t, plan.Project.Files, "guides/214.lading"), "/// p.5:")
}

func TestPlan_WritesTheGuidesInterchangeIntoThePartnerFile(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{acmeWithOwnGuide(t, req, valueProduction)}

	plan, err := h.service.PreviewConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	usage := valueOf(t, plan.Draft.Interchange, fieldUsageIndicator)
	assert.Equal(t, "T", usage.Value)
	assert.Equal(t, "P", usage.Current)
	assert.Equal(t, ediports.ChangeKindChange, usage.Change)

	partner := ""
	for _, file := range plan.Project.Files {
		if strings.HasPrefix(strings.TrimSpace(file.Source), "partner ") {
			partner = file.Source
		}
	}
	assert.Contains(t, partner, "usage test", "the guide's ISA15 reaches the project")
	assert.Contains(t, partner, "\"007912345\"")
	assert.Contains(t, partner, "setting autoDecline", "the rest of the partner file is kept")
}
