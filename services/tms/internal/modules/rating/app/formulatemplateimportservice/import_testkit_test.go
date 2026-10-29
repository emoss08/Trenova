package formulatemplateimportservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatetest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/formulatemplateportsmock"
	"go.uber.org/zap"
)

type testDeps struct {
	repo        *formulatemplateportsmock.MockFormulaTemplateRepository
	versionRepo *formulatemplateportsmock.MockFormulaTemplateVersionRepository
	auditSvc    *auditportsmock.MockAuditService
	svc         *Service
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
	return &testDeps{
		repo:        kit.Repo,
		versionRepo: kit.VersionRepo,
		auditSvc:    kit.AuditSvc,
		svc: &Service{
			l:           zap.NewNop(),
			db:          dbtest.NopConnection{},
			templates:   templates,
			repo:        kit.Repo,
			versionRepo: kit.VersionRepo,
		},
	}
}
