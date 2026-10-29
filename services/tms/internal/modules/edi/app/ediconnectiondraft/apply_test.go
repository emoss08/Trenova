package ediconnectiondraft

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionfield"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectionguide"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func approver(req *ediconnectiondraftports.ConnectionDraftRequest) *permissionports.RequestActor {
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

	h := newHarness(t)
	req := acmeRequest(t)
	applied, err := h.service.ApplyConnectionDraft(
		t.Context(),
		&ediconnectiondraftports.ApplyConnectionDraftRequest{
			Draft:                  req,
			ExpectedDocumentDigest: ediconnectionfield.DocumentDigest(req.Document.Pages),
			Actor:                  approver(req),
		},
	)
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
	assert.NotContains(
		t,
		transport.Config,
		"isaReceiverId",
		"the interchange lives in the Lading project",
	)
	assert.Empty(t, transport.Secrets, "no secret comes from the document")

	require.Len(t, h.projects.imports, 1)
	imported := h.projects.imports[0]
	assert.Equal(t, partner.ID, imported.PartnerID)
	assert.Equal(t, "Drafted from acme-retail-edi-guide.pdf", imported.Notes)
	partnerFile := ""
	for _, file := range imported.Files {
		if strings.HasPrefix(strings.TrimSpace(file.Source), "partner ") {
			partnerFile = file.Source
		}
	}
	assert.Contains(t, partnerFile, "007912345", "the partner's interchange ID is in the project")
	assert.Contains(t, partnerFile, "TRNV")
	assert.Contains(
		t,
		partnerFile,
		"partner acmeretail",
		"the starter is named for the new partner",
	)
	guides := map[string]string{}
	for _, file := range imported.Files {
		if strings.HasPrefix(file.Path, ediconnectionguide.GuideDirectory) {
			guides[file.Path] = file.Source
		}
	}
	require.NotEmpty(t, guides)
	for path, source := range guides {
		assert.Contains(t, source, `"Acme Retail Group `, "%s names the new partner", path)
	}
	var project *ediconnectiondraftports.DraftWrite
	for index := range applied.Writes {
		if applied.Writes[index].Record == projectRecord {
			project = &applied.Writes[index]
		}
	}
	require.NotNil(t, project, "the Lading project draft is a write the person sees")
	assert.Equal(t, writeCreated, project.Action)
	assert.Equal(t, "Draft 1", project.Label)

	require.Len(
		t,
		h.setup.controls,
		3,
		"990, 214 and 210 each start their interchange numbers at 1000",
	)
	for _, control := range h.setup.controls {
		assert.Equal(t, int64(1000), control.NextValue)
		assert.Equal(t, edidocument.ControlNumberKindInterchange, control.Kind)
	}
	assert.Empty(t, h.setup.mappings)
	assert.NotEmpty(t, applied.Writes)
}

func TestApply_UpdatesOnlyWhatChangedOnAKnownPartner(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	existing := existingAcme(t, req)
	h.repo.partners = []*ediconnectiondraftports.EDIPartnerSetup{existing}

	applied, err := h.service.ApplyConnectionDraft(
		t.Context(),
		&ediconnectiondraftports.ApplyConnectionDraftRequest{
			Draft:                  req,
			ExpectedDocumentDigest: ediconnectionfield.DocumentDigest(req.Document.Pages),
			Actor:                  approver(req),
		},
	)
	require.NoError(t, err)

	assert.False(t, applied.Created)
	require.Len(t, h.setup.partners, 1)
	assert.Equal(t, existing.Partner.ID, h.setup.partners[0].ID)
	assert.Equal(
		t,
		int64(4),
		h.setup.partners[0].Version,
		"the update is pinned to the version read",
	)
	assert.Equal(t, "Dana Whitfield", h.setup.partners[0].ContactName)
	assert.Equal(t, domaintypes.StatusActive, h.setup.partners[0].Status)
	assert.True(t, h.setup.partners[0].EnabledForInbound, "the 204 turns inbound on")
	assert.Empty(t, h.setup.transports, "the AS2 profile already matches")

	require.Len(t, h.projects.imports, 1, "the guide's requirements go into a new draft")
	assert.Equal(t, existing.Partner.ID, h.projects.imports[0].PartnerID)
	for _, control := range h.setup.controls {
		assert.NotEqual(
			t,
			documentTypeID(edidocument.TransactionSet214, edidocument.DocumentDirectionOutbound),
			control.DocumentTypeID,
			"the 214's numbers are never moved back",
		)
	}
}

func TestApply_RefusesWhatThePersonDidNotApprove(t *testing.T) {
	t.Parallel()

	req := acmeRequest(t)
	digest := ediconnectionfield.DocumentDigest(req.Document.Pages)
	cases := map[string]struct {
		request  func() *ediconnectiondraftports.ApplyConnectionDraftRequest
		contains string
	}{
		"no approver": {
			request: func() *ediconnectiondraftports.ApplyConnectionDraftRequest {
				return &ediconnectiondraftports.ApplyConnectionDraftRequest{
					Draft:                  req,
					ExpectedDocumentDigest: digest,
				}
			},
			contains: "must approve",
		},
		"an approver from another organization": {
			request: func() *ediconnectiondraftports.ApplyConnectionDraftRequest {
				actor := approver(req)
				actor.OrganizationID = pulid.MustNew("org_")
				return &ediconnectiondraftports.ApplyConnectionDraftRequest{
					Draft:                  req,
					ExpectedDocumentDigest: digest,
					Actor:                  actor,
				}
			},
			contains: "another organization",
		},
		"a document read differently": {
			request: func() *ediconnectiondraftports.ApplyConnectionDraftRequest {
				return &ediconnectiondraftports.ApplyConnectionDraftRequest{
					Draft: req, ExpectedDocumentDigest: "sha256:00", Actor: approver(req),
				}
			},
			contains: "read differently",
		},
		"a draft with a decision left": {
			request: func() *ediconnectiondraftports.ApplyConnectionDraftRequest {
				held := acmeRequest(t)
				held.Document = req.Document
				held.Values[1].Confidence = 0.4
				return &ediconnectiondraftports.ApplyConnectionDraftRequest{
					Draft: held, ExpectedDocumentDigest: digest, Actor: approver(held),
				}
			},
			contains: "Decide partner name",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			_, err := h.service.ApplyConnectionDraft(t.Context(), tc.request())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.contains)
			assert.Empty(t, h.setup.partners, "nothing is written")
			assert.Empty(t, h.projects.imports)
		})
	}
}

func TestApply_StopsAtTheFirstFailedWrite(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.setup.failOn = "transport"
	req := acmeRequest(t)
	_, err := h.service.ApplyConnectionDraft(
		t.Context(),
		&ediconnectiondraftports.ApplyConnectionDraftRequest{
			Draft:                  req,
			ExpectedDocumentDigest: ediconnectionfield.DocumentDigest(req.Document.Pages),
			Actor:                  approver(req),
		},
	)
	require.Error(t, err)
	assert.Empty(t, h.projects.imports, "the writes stop at the failed one")
}
