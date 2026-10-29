package formulatemplatereviewservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
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
	dependents      *formulatemplatetest.FakeDependents
	resolver        *tariff.TemplateResolver
	templates       *formulatemplateservice.Service
	svc             *Service
}

func setupTest(t *testing.T) *testDeps {
	t.Helper()

	kit := formulatemplatetest.NewDeps(t)
	templates := formulatemplateservice.New(formulatemplateservice.Params{
		Logger:       zap.NewNop(),
		DB:           dbtest.NopConnection{},
		Repo:         kit.Repo,
		VersionRepo:  kit.VersionRepo,
		Checker:      kit.Checker,
		Resolver:     kit.Resolver,
		AuditService: kit.AuditSvc,
	})
	dependents := &formulatemplatetest.FakeDependents{}
	return &testDeps{
		repo:            kit.Repo,
		versionRepo:     kit.VersionRepo,
		libraryVersions: kit.LibraryVersions,
		auditSvc:        kit.AuditSvc,
		dependents:      dependents,
		resolver:        kit.Resolver,
		templates:       templates,
		svc: &Service{
			l:           zap.NewNop(),
			db:          dbtest.NopConnection{},
			templates:   templates,
			repo:        kit.Repo,
			versionRepo: kit.VersionRepo,
			reviewRepo:  &stubReviewRepo{},
			checker:     kit.Checker,
			resolver:    kit.Resolver,
			dependents:  dependents,
		},
	}
}
