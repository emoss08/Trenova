package formulatemplatestudioservice

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
	rater           *formulatemplatetest.FakeRater
	records         *formulatemplatetest.FakeRecords
	resolver        *tariff.TemplateResolver
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
	rater := &formulatemplatetest.FakeRater{}
	records := &formulatemplatetest.FakeRecords{}
	return &testDeps{
		repo:            kit.Repo,
		versionRepo:     kit.VersionRepo,
		libraryVersions: kit.LibraryVersions,
		auditSvc:        kit.AuditSvc,
		rater:           rater,
		records:         records,
		resolver:        kit.Resolver,
		svc: &Service{
			l:           zap.NewNop(),
			templates:   templates,
			versionRepo: kit.VersionRepo,
			libraries:   formulatemplatetest.NoApprovedLibraries{},
			checker:     kit.Checker,
			resolver:    kit.Resolver,
			rater:       rater,
			records:     records,
		},
	}
}
