package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stretchr/testify/suite"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/infrastructure/database/postgres"
	"github.com/trenova-app/transport/internal/infrastructure/database/postgres/migrations"
	postgresRepos "github.com/trenova-app/transport/internal/infrastructure/database/postgres/repositories"
	"github.com/trenova-app/transport/internal/pkg/config"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/registry"
	"github.com/trenova-app/transport/internal/pkg/utils/fileutils"
	"github.com/trenova-app/transport/internal/pkg/validator/compliancevalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/rptmetavalidator"
	"github.com/trenova-app/transport/internal/pkg/validator/workervalidator"
	"github.com/trenova-app/transport/test/fixtures"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
	"go.uber.org/fx"
)

type TestModule struct {
	fx.In

	Logger     *logger.Logger
	Validators ValidatorSet
}

type ValidatorSet struct {
	fx.In

	WorkerPTOValidator     *workervalidator.WorkerPTOValidator
	WorkerProfileValidator *workervalidator.WorkerProfileValidator
	WorkerValidator        *workervalidator.Validator
	ComplianceValidator    *compliancevalidator.Validator
	VariableValidator      *rptmetavalidator.VariableValidator
	RptMetaValidator       *rptmetavalidator.MetadataValidator
}

// RepositorySet groups all repositories
type RepositorySet struct {
	fx.In

	WorkerRepo    repositories.WorkerRepository
	HazmatExpRepo repositories.HazmatExpirationRepository
}

// BaseSuite provides common test suite functionality
type BaseSuite struct {
	suite.Suite
	Ctx          context.Context
	DB           *bun.DB
	Fixture      *dbfixture.Fixture
	Logger       *logger.Logger
	Config       *config.Config
	Validators   ValidatorSet
	Repositories RepositorySet
	shutdowner   fx.Shutdowner
	migrator     *migrate.Migrator
}

// SetupSuite prepares the test environment
func (s *BaseSuite) SetupSuite() {
	var err error

	// Get test database connection
	s.DB, err = GetTestDB()
	s.Require().NoError(err)

	s.Ctx = context.Background()
	s.Config = NewTestConfig()
	s.Config.DB.Database = testDBName

	// Initialize migrator
	s.migrator = migrate.NewMigrator(s.DB, migrations.Migrations)

	testConfigManager := NewTestConfigManager(s.Config)

	var module TestModule
	app := fx.New(
		// Infrastructure
		fx.Provide(
			func() *config.Manager { return testConfigManager },
			func() *config.Config { return s.Config },
			func() *bun.DB { return s.DB },
			postgres.NewConnection,
			NewTestLogger,
		),
		// Repositories
		fx.Provide(
			postgresRepos.NewWorkerRepository,
			postgresRepos.NewHazmatExpirationRepository,
		),
		fx.Provide(
			workervalidator.NewWorkerPTOValidator,
			workervalidator.NewWorkerProfileValidator,
			workervalidator.NewValidator,
			compliancevalidator.NewValidator,
			rptmetavalidator.NewVariableValidator,
			rptmetavalidator.NewMetadataValidator,
		),
		// Populate with basic modules
		fx.Populate(&module),
		fx.Populate(&s.Validators),
		fx.Populate(&s.Repositories),
		fx.Populate(&s.shutdowner),
	)

	err = app.Start(s.Ctx)
	s.Require().NoError(err)

	// Register models
	s.DB.RegisterModel(registry.RegisterEntities()...)

	// Add debug hook in test environment
	s.DB.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(false),
		bundebug.FromEnv("TEST_DEBUG"),
	))
}

// BeforeTest runs before each test
func (s *BaseSuite) LoadTestDB() {
	var err error

	// Reset database like your reset command
	_, err = s.DB.ExecContext(s.Ctx, `
        DROP SCHEMA public CASCADE;
        CREATE SCHEMA public;
    `)
	s.Require().NoError(err)

	// Initialize migrations like your init command
	err = s.migrator.Init(s.Ctx)
	s.Require().NoError(err)

	// Lock migrations
	err = s.migrator.Lock(s.Ctx)
	s.Require().NoError(err)

	// Run migrations
	_, err = s.migrator.Migrate(s.Ctx)
	s.Require().NoError(err)

	// Unlock migrations
	err = s.migrator.Unlock(s.Ctx)
	s.Require().NoError(err)

	// Register models again after reset
	s.DB.RegisterModel(registry.RegisterEntities()...)

	// Initialize fixture helpers
	helpers := fixtures.NewFixtureHelpers()

	// Setup fixtures
	s.Fixture = dbfixture.New(s.DB,
		dbfixture.WithTemplateFuncs(helpers.GetTemplateFuncs()),
	)

	// Load fixtures
	err = s.loadFixtures()
	s.Require().NoError(err)

	// Load additional fixtures
	err = fixtures.LoadFixtures(s.Ctx, s.Fixture, s.DB)
	s.Require().NoError(err)

	s.T().Logf("Validators initialized: %+v", s.Validators)
	s.T().Logf("Repositories initialized: %+v", s.Repositories)
}

// TearDownSuite cleans up after all tests
func (s *BaseSuite) TearDownSuite() {
	if s.shutdowner != nil {
		s.Require().NoError(s.shutdowner.Shutdown())
	}
}

// ResetTestDB resets the database state
func ResetTestDB(ctx context.Context, db *bun.DB) error {
	// Drop all tables in the current schema
	_, err := db.ExecContext(ctx, `
        DROP SCHEMA IF EXISTS public CASCADE;
        CREATE SCHEMA public;
        GRANT ALL ON SCHEMA public TO postgres;
        GRANT ALL ON SCHEMA public TO public;
    `)
	if err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	return nil
}

// loadFixtures loads all fixture files
func (s *BaseSuite) loadFixtures() error {
	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root
	projectRoot, err := fileutils.FindProjectRoot(currentDir)
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	fixturesPath := filepath.Join(projectRoot, "test", "fixtures")
	fmt.Printf("Loading fixtures from: %s\n", fixturesPath)

	if err = s.Fixture.Load(s.Ctx, os.DirFS(fixturesPath), "fixtures.yml"); err != nil {
		return fmt.Errorf("failed to load fixtures: %w", err)
	}

	return nil
}

// GetFixture retrieves a specific fixture by name
func (s *BaseSuite) GetFixture(name string) any {
	return s.Fixture.MustRow(name)
}
