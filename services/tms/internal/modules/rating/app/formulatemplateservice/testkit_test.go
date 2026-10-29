package formulatemplateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatetest"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulalibraryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"go.uber.org/zap"
)

type testDeps struct {
	repo            *formulatemplateportsmock.MockFormulaTemplateRepository
	versionRepo     *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	libraryVersions *formulalibraryportsmock.MockFormulaLibraryVersionRepository
	auditSvc        *auditportsmock.MockAuditService
	resolver        *tariff.TemplateResolver
	svc             *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()

	kit := formulatemplatetest.NewDeps(t)
	return &testDeps{
		repo:            kit.Repo,
		versionRepo:     kit.VersionRepo,
		libraryVersions: kit.LibraryVersions,
		auditSvc:        kit.AuditSvc,
		resolver:        kit.Resolver,
		svc: &Service{
			l:            zap.NewNop(),
			db:           dbtest.NopConnection{},
			repo:         kit.Repo,
			versionRepo:  kit.VersionRepo,
			checker:      kit.Checker,
			resolver:     kit.Resolver,
			auditService: kit.AuditSvc,
		},
	}
}
