package documenttemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFinishRunningPart_InlinesAssetsTheWayTheBodyDoes(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	inliner := documenttemplateportsmock.NewMockAssetInliner(t)
	inliner.EXPECT().
		InlineAssets(mock.Anything, &documenttemplateports.AssetInlineRequest{
			HTML:           `<div><img src="https://cdn.example.com/logo.png"></div>`,
			Field:          "headerHtml",
			OrganizationID: orgID,
		}).
		Return(&documenttemplateports.AssetInlineResult{HTML: "<div>Inlined</div>"}, nil).
		Once()
	svc := &Service{inliner: inliner}

	part, diags, err := svc.finishRunningPart(
		t.Context(),
		"headerHtml",
		`<div><img src="https://cdn.example.com/logo.png"></div>`,
		compileOptions{OrganizationID: orgID},
	)

	require.NoError(t, err)
	assert.Equal(t, "<div>Inlined</div>", part)
	assert.Empty(t, diags)
}

func TestFinishRunningPart_LeavesAnEmptyPartAlone(t *testing.T) {
	t.Parallel()

	svc := &Service{inliner: documenttemplateportsmock.NewMockAssetInliner(t)}

	part, diags, err := svc.finishRunningPart(t.Context(), "footerHtml", "", compileOptions{})

	require.NoError(t, err)
	assert.Empty(t, part)
	assert.Empty(t, diags)
}
