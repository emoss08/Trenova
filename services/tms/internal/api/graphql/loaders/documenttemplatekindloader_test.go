package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentTemplateKindBatchFunc_MapsKindsAndDefaultsMissingToEmpty(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("dtpl_")
	secondID := pulid.MustNew("dtpl_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	getter := &stubDocumentTemplateKindsGetter{
		kinds: func(_ context.Context, tenantInfo pagination.TenantInfo, ids []pulid.ID) (map[pulid.ID]documenttemplate.Kind, error) {
			assert.Equal(t, tenant, tenantInfo)
			assert.Equal(t, []pulid.ID{firstID, secondID}, ids)
			return map[pulid.ID]documenttemplate.Kind{firstID: "invoice"}, nil
		},
	}
	factory := &DocumentTemplateKindByTemplateIDLoaderFactory{kinds: getter}

	values, errs := factory.batchFunc(tenant)(t.Context(), []string{
		firstID.String(),
		"bad",
		secondID.String(),
		firstID.String(),
	})

	require.Len(t, values, 4)
	require.NoError(t, errs[0])
	assert.Equal(t, documenttemplate.Kind("invoice"), values[0])
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Empty(t, values[2])
	require.NoError(t, errs[3])
	assert.Equal(t, documenttemplate.Kind("invoice"), values[3])
}

func TestDocumentTemplateKindBatchFunc_RepositoryErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("dtpl_")
	repoErr := errors.New("repository failed")
	getter := &stubDocumentTemplateKindsGetter{
		kinds: func(context.Context, pagination.TenantInfo, []pulid.ID) (map[pulid.ID]documenttemplate.Kind, error) {
			return nil, repoErr
		},
	}
	factory := &DocumentTemplateKindByTemplateIDLoaderFactory{kinds: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{
		"bad",
		templateID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.NotErrorIs(t, errs[0], repoErr)
	require.ErrorIs(t, errs[1], repoErr)
}

func TestDocumentTemplateKindBatchFunc_NoValidKeysSkipsFetch(t *testing.T) {
	t.Parallel()

	getter := &stubDocumentTemplateKindsGetter{
		kinds: func(context.Context, pagination.TenantInfo, []pulid.ID) (map[pulid.ID]documenttemplate.Kind, error) {
			t.Fatal("fetch must not run without valid keys")
			return nil, nil
		},
	}
	factory := &DocumentTemplateKindByTemplateIDLoaderFactory{kinds: getter}

	values, errs := factory.batchFunc(pagination.TenantInfo{})(t.Context(), []string{"bad"})

	require.Len(t, values, 1)
	require.Error(t, errs[0])
}

type stubDocumentTemplateKindsGetter struct {
	kinds func(context.Context, pagination.TenantInfo, []pulid.ID) (map[pulid.ID]documenttemplate.Kind, error)
}

func (s *stubDocumentTemplateKindsGetter) KindsByTemplateIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	templateIDs []pulid.ID,
) (map[pulid.ID]documenttemplate.Kind, error) {
	return s.kinds(ctx, tenantInfo, templateIDs)
}
