package ediconnectiondraft

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/shared/pulid"
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

	draft, err := newHarness().service.PlanConnectionDraft(t.Context(), req)
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

	h := newHarness()
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{existingAcme(req)}
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

	h := newHarness()
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{existingAcme(req)}
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

func acmeWithInbound204(req *ediports.ConnectionDraftRequest, usage string) *ediports.EDIPartnerSetup {
	existing := existingAcme(req)
	outbound := existing.DocumentProfiles[0]
	template := pulid.MustNew("editpl_")
	existing.DocumentProfiles = append(existing.DocumentProfiles, &edi.EDIPartnerDocumentProfile{
		ID: pulid.MustNew("edidp_"), EDIPartnerID: existing.Partner.ID,
		DocumentTypeID: documentTypeID(edi.TransactionSet204, edi.DocumentDirectionInbound),
		TemplateID:     template, Name: "Acme 204", Status: edi.DocumentStatusActive,
		Direction: edi.DocumentDirectionInbound, TransactionSet: edi.TransactionSet204,
		X12VersionOverride: "004010", FunctionalGroupID: "SM",
		Envelope: edi.X12EnvelopeSettings{
			InterchangeSenderQualifier:   outbound.Envelope.InterchangeReceiverQualifier,
			InterchangeSenderID:          outbound.Envelope.InterchangeReceiverID,
			InterchangeReceiverQualifier: outbound.Envelope.InterchangeSenderQualifier,
			InterchangeReceiverID:        outbound.Envelope.InterchangeSenderID,
			ApplicationSenderCode:        outbound.Envelope.ApplicationReceiverCode,
			ApplicationReceiverCode:      outbound.Envelope.ApplicationSenderCode,
			InterchangeUsageIndicator:    usage,
			ElementSeparator:             "*",
			SegmentTerminator:            "~",
			ComponentSeparator:           ">",
			RepetitionSeparator:          "|",
		},
		ValidationMode: edi.ValidationModeStrict, Version: 3,
		Template: &edi.EDITemplate{ID: template, Name: "Acme 204 template"},
	})
	return existing
}

func plannedProfileFor(
	t *testing.T,
	plan *ediports.ConnectionDraftPlan,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) ediports.PlannedDocumentProfile {
	t.Helper()
	for _, profile := range plan.DocumentProfiles {
		if profile.After.TransactionSet == set && profile.After.Direction == direction {
			return profile
		}
	}
	require.Failf(t, "missing planned profile", "%s %s", set, direction)
	return ediports.PlannedDocumentProfile{}
}

func withoutValue(values []ediports.DraftValueInput, field string) []ediports.DraftValueInput {
	kept := make([]ediports.DraftValueInput, 0, len(values))
	for _, input := range values {
		if input.Field != field {
			kept = append(kept, input)
		}
	}
	return kept
}

func TestPlan_KeepsEachProfilesOwnEnvelopeWhereTheGuideIsSilent(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	existing := acmeWithInbound204(req, "P")
	h.repo.partners = []*ediports.EDIPartnerSetup{existing}
	req.Values = withoutValue(req.Values, fieldUsageIndicator)

	plan, err := h.service.PreviewConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	usage := valueOf(t, plan.Draft.Interchange, fieldUsageIndicator)
	require.Equal(t, ediports.SourceKindSample, usage.Source.Kind,
		"with the text silent, the guide's own sample still says T")

	inbound := plannedProfileFor(t, plan, edi.TransactionSet204, edi.DocumentDirectionInbound)
	require.NotNil(t, inbound.Before)
	expected := inbound.Before.Envelope
	expected.InterchangeUsageIndicator = "T"
	assert.Equal(t, expected, inbound.After.Envelope,
		"a setting the guide never names stays as each profile has it")
	outbound := plannedProfileFor(t, plan, edi.TransactionSet214, edi.DocumentDirectionOutbound)
	assert.Equal(t, "^", outbound.After.Envelope.RepetitionSeparator)
}

func TestPlan_WritesWhatTheGuideSaysToEveryProfile(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	h.repo.partners = []*ediports.EDIPartnerSetup{acmeWithInbound204(req, "P")}

	plan, err := h.service.PreviewConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	inbound := plannedProfileFor(t, plan, edi.TransactionSet204, edi.DocumentDirectionInbound)
	assert.Equal(t, "T", inbound.After.Envelope.InterchangeUsageIndicator,
		"the guide's ISA15 reaches the profile that differs from it")
	assert.Equal(t, "|", inbound.After.Envelope.RepetitionSeparator,
		"a separator the guide never names is left as it is")
	assert.Equal(t, ediports.ChangeKindChange, inbound.Change)
	assert.Equal(t, "TRNV", inbound.After.Envelope.InterchangeReceiverID)
	assert.Equal(t, "007912345", inbound.After.Envelope.InterchangeSenderID)
}
