package documentcontrolservice_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/app/documentcontrolservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newService(
	t *testing.T,
	cfg *config.Config,
	repo *tenantportsmock.MockDocumentControlRepository,
) *documentcontrolservice.Service {
	t.Helper()

	return documentcontrolservice.New(documentcontrolservice.Params{
		Logger: zap.NewNop(),
		Repo:   repo,
		Validator: documentcontrolservice.NewValidator(
			documentcontrolservice.ValidatorParams{Config: cfg},
		),
		AuditService: auditportsmock.NewMockAuditService(t),
	})
}

func configuredConverter() *config.Config {
	return &config.Config{DocumentConversion: config.DocumentConversionConfig{
		DoclingURL: "http://docling:5001",
		APIKey:     "key",
	}}
}

/*
An installation without the docling sidecar cannot read a single PDF on the
Docling engine, so choosing it is refused on the field rather than accepted
and left to stop every upload with CONVERTER_UNAVAILABLE.
*/
func TestUpdateRefusesDoclingWithoutAConverter(t *testing.T) {
	t.Parallel()

	for name, cfg := range map[string]*config.Config{
		"no url":     {},
		"no api key": {DocumentConversion: config.DocumentConversionConfig{DoclingURL: "http://docling:5001"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")
			repo := tenantportsmock.NewMockDocumentControlRepository(t)
			repo.EXPECT().
				GetOrCreate(mock.Anything, orgID, buID).
				Return(tenant.NewDefaultDocumentControl(orgID, buID), nil)

			entity := tenant.NewDefaultDocumentControl(orgID, buID)
			entity.ExtractionEngine = tenant.ExtractionEngineDocling

			_, err := newService(t, cfg, repo).Update(t.Context(), entity, pulid.MustNew("usr_"))

			multiErr, ok := errors.AsType[*errortypes.MultiError](err)
			require.True(t, ok, "want a field error, got %v", err)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, "extractionEngine", multiErr.Errors[0].Field)
			assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
		})
	}
}

func TestValidateUpdateAllowsLegacyWithoutAConverterAndDoclingWithOne(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	legacy := tenant.NewDefaultDocumentControl(orgID, buID)
	assert.Nil(t, documentcontrolservice.NewValidator(
		documentcontrolservice.ValidatorParams{Config: &config.Config{}},
	).ValidateUpdate(t.Context(), legacy))

	docling := tenant.NewDefaultDocumentControl(orgID, buID)
	docling.ExtractionEngine = tenant.ExtractionEngineDocling
	assert.Nil(t, documentcontrolservice.NewValidator(
		documentcontrolservice.ValidatorParams{Config: configuredConverter()},
	).ValidateUpdate(t.Context(), docling))
}
