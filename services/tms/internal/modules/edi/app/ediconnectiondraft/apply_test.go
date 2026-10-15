package ediconnectiondraft

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func approver(req *ediports.ConnectionDraftRequest) *permissionports.RequestActor {
	user := pulid.MustNew("usr_")
	return &permissionports.RequestActor{
		PrincipalID:    user,
		UserID:         user,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
}

func TestApply_WritesANewConnectionInactive(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	applied, err := h.service.ApplyConnectionDraft(t.Context(), &ediports.ApplyConnectionDraftRequest{
		Draft:                  req,
		ExpectedDocumentDigest: DocumentDigest(req.Document.Pages),
		Actor:                  approver(req),
	})
	require.NoError(t, err)

	assert.True(t, applied.Created)
	require.Len(t, h.setup.partners, 1)
	partner := h.setup.partners[0]
	assert.Equal(t, applied.PartnerID, partner.ID)
	assert.Equal(t, edi.PartnerKindExternal, partner.Kind)
	assert.Equal(t, domaintypes.StatusInactive, partner.Status)
	assert.Equal(t, "ACMERETAIL", partner.Code)
	assert.True(t, partner.EnabledForInbound)
	assert.True(t, partner.EnabledForOutbound)

	require.Len(t, h.setup.transports, 1)
	transport := h.setup.transports[0]
	assert.Equal(t, partner.ID, transport.EDIPartnerID)
	assert.Equal(t, string(domaintypes.StatusInactive), transport.Status)
	assert.Equal(t, "Acme Retail Group AS2", transport.Name)
	assert.Equal(t, "ACMERETAILAS2", transport.Config["partnerAS2Id"])
	assert.Equal(t, "007912345", transport.Config["isaReceiverId"])
	assert.Empty(t, transport.Secrets, "no secret comes from the document")

	require.Len(t, h.setup.profiles, 4)
	for _, profile := range h.setup.profiles {
		assert.Equal(t, partner.ID, profile.EDIPartnerID)
		assert.Equal(t, edi.DocumentStatusInactive, profile.Status)
		assert.True(t, profile.TemplateID.IsNotNil(), "every profile names its template")
	}
	assert.ElementsMatch(t, []edi.DocumentDirection{
		edi.DocumentDirectionInbound,
		edi.DocumentDirectionOutbound,
		edi.DocumentDirectionOutbound,
		edi.DocumentDirectionOutbound,
	}, h.templates.ensured)

	inbound := h.setup.profiles[0]
	assert.Equal(t, "007912345", inbound.Envelope.InterchangeSenderID, "the partner sends the 204")
	assert.Equal(t, "TRNV", inbound.Envelope.InterchangeReceiverID)

	require.Len(t, h.setup.controls, 3, "990, 214 and 210 each start their interchange numbers at 1000")
	for _, control := range h.setup.controls {
		assert.Equal(t, int64(1000), control.NextValue)
		assert.Equal(t, edi.ControlNumberKindInterchange, control.Kind)
	}
	assert.Empty(t, h.setup.mappings)
	assert.NotEmpty(t, applied.Writes)
}

func TestApply_UpdatesOnlyWhatChangedOnAKnownPartner(t *testing.T) {
	t.Parallel()

	h := newHarness()
	req := acmeRequest(t)
	existing := existingAcme(req)
	h.repo.partners = []*ediports.EDIPartnerSetup{existing}

	applied, err := h.service.ApplyConnectionDraft(t.Context(), &ediports.ApplyConnectionDraftRequest{
		Draft:                  req,
		ExpectedDocumentDigest: DocumentDigest(req.Document.Pages),
		Actor:                  approver(req),
	})
	require.NoError(t, err)

	assert.False(t, applied.Created)
	require.Len(t, h.setup.partners, 1)
	assert.Equal(t, existing.Partner.ID, h.setup.partners[0].ID)
	assert.Equal(t, int64(4), h.setup.partners[0].Version, "the update is pinned to the version read")
	assert.Equal(t, "Dana Whitfield", h.setup.partners[0].ContactName)
	assert.Equal(t, domaintypes.StatusActive, h.setup.partners[0].Status)
	assert.True(t, h.setup.partners[0].EnabledForInbound, "the 204 turns inbound on")
	assert.Empty(t, h.setup.transports, "the AS2 profile already matches")

	for _, profile := range h.setup.profiles {
		assert.NotEqual(t, existing.DocumentProfiles[0].ID, profile.ProfileID,
			"the 214 already matches the spec and is left alone")
	}
	assert.Len(t, h.setup.profiles, 3)
	for _, control := range h.setup.controls {
		assert.NotEqual(t, documentTypeID(edi.TransactionSet214, edi.DocumentDirectionOutbound),
			control.DocumentTypeID, "the 214's numbers are never moved back")
	}
}

func TestApply_RefusesWhatThePersonDidNotApprove(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	digest := DocumentDigest(req.Document.Pages)
	cases := map[string]struct {
		request  func() *ediports.ApplyConnectionDraftRequest
		contains string
	}{
		"no approver": {
			request: func() *ediports.ApplyConnectionDraftRequest {
				return &ediports.ApplyConnectionDraftRequest{Draft: req, ExpectedDocumentDigest: digest}
			},
			contains: "must approve",
		},
		"an approver from another organization": {
			request: func() *ediports.ApplyConnectionDraftRequest {
				actor := approver(req)
				actor.OrganizationID = pulid.MustNew("org_")
				return &ediports.ApplyConnectionDraftRequest{Draft: req, ExpectedDocumentDigest: digest, Actor: actor}
			},
			contains: "another organization",
		},
		"a document read differently": {
			request: func() *ediports.ApplyConnectionDraftRequest {
				return &ediports.ApplyConnectionDraftRequest{
					Draft: req, ExpectedDocumentDigest: "sha256:00", Actor: approver(req),
				}
			},
			contains: "read differently",
		},
		"a draft with a decision left": {
			request: func() *ediports.ApplyConnectionDraftRequest {
				held := acmeRequest(t)
				held.Document = req.Document
				held.Values[1].Confidence = 0.4
				return &ediports.ApplyConnectionDraftRequest{
					Draft: held, ExpectedDocumentDigest: digest, Actor: approver(held),
				}
			},
			contains: "Decide partner name",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHarness()
			_, err := h.service.ApplyConnectionDraft(t.Context(), tc.request())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.contains)
			assert.Empty(t, h.setup.partners, "nothing is written")
			assert.Empty(t, h.setup.profiles)
		})
	}
}

func TestApply_StopsAtTheFirstFailedWrite(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.setup.failOn = "transport"
	req := acmeRequest(t)
	_, err := h.service.ApplyConnectionDraft(t.Context(), &ediports.ApplyConnectionDraftRequest{
		Draft:                  req,
		ExpectedDocumentDigest: DocumentDigest(req.Document.Pages),
		Actor:                  approver(req),
	})
	require.Error(t, err)
	assert.Empty(t, h.setup.profiles, "the transaction rolls back from the failed write")
}
