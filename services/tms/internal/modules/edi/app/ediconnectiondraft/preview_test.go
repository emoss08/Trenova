package ediconnectiondraft

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreview_ShowsEachRecordAsItIsAndAsItWouldBe(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := acmeRequest(t)
	existing := existingAcme(t, req)
	h.repo.partners = []*ediconnectiondraftports.EDIPartnerSetup{existing}

	plan, err := h.service.PreviewConnectionDraft(t.Context(), req)
	require.NoError(t, err)

	assert.True(t, plan.Draft.Ready)
	assert.Equal(t, ediconnectiondraftports.ChangeKindChange, plan.Partner.Change)
	require.NotNil(t, plan.Partner.Before)
	assert.Equal(t, "Lee Park", plan.Partner.Before.ContactName)
	assert.Equal(t, "Dana Whitfield", plan.Partner.After.ContactName)
	assert.Equal(t, existing.Partner.Version, plan.Partner.After.Version, "the write is pinned")

	require.NotNil(t, plan.Transport)
	assert.Equal(t, ediconnectiondraftports.ChangeKindSame, plan.Transport.Change)
	assert.NotContains(t, plan.Transport.After.Config, "privateKey", "secrets never enter a plan")

	require.NotNil(t, plan.Project)
	assert.Equal(t, ediconnectiondraftports.ChangeKindChange, plan.Project.Change)
	assert.Same(t, existing.Project, plan.Project.Project)
	assert.Same(t, existing.LatestVersion, plan.Project.Before)
	assert.NotEmpty(t, plan.Project.Files)
	for _, sample := range plan.Project.Samples {
		assert.True(t, strings.HasPrefix(sample.Path, "samples/spec-page-"),
			"a draft adds the guide's own examples; the project keeps its samples")
	}

	require.Len(
		t,
		plan.ControlNumbers,
		2,
		"the 990 and 210 start at 1000; the 214 already passed it",
	)
	for _, control := range plan.ControlNumbers {
		assert.Equal(t, ediconnectiondraftports.ChangeKindAdd, control.Change)
		assert.Equal(t, int64(1000), control.After.NextValue)
	}
}
