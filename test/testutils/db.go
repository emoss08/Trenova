package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/migrations"
	"github.com/emoss08/trenova/internal/pkg/utils/fileutils"
	"github.com/emoss08/trenova/test/fixtures"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
)

var BunFixture *dbfixture.Fixture

type TestDBConnection struct {
	db      *bun.DB
	fixture *dbfixture.Fixture
}

func (t *TestDBConnection) DB(ctx context.Context) (*bun.DB, error) {
	return t.db, nil
}

func (t *TestDBConnection) ConnectionInfo() (*db.ConnectionInfo, error) {
	return &db.ConnectionInfo{
		Host:     "localhost",
		Port:     5432,
		Database: "trenova",
		Username: "postgres",
		Password: "postgres",
	}, nil
}

func (t *TestDBConnection) Close() error {
	return nil
}

func (t *TestDBConnection) Fixture(ctx context.Context) (*dbfixture.Fixture, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	projectRoot, err := fileutils.FindProjectRoot(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	fixturesPath := filepath.Join(projectRoot, "test", "fixtures")

	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("fixtures directory does not exist: %s", fixturesPath)
	}

	if err = fileutils.EnsureDirExists(fixturesPath); err != nil {
		return nil, fmt.Errorf("failed to ensure fixtures directory exists: %w", err)
	}

	helpers := fixtures.NewFixtureHelpers()
	fixture := dbfixture.New(t.db, dbfixture.WithTemplateFuncs(helpers.GetTemplateFuncs()))

	BunFixture = fixture

	if err := fixture.Load(ctx, os.DirFS(fixturesPath), "fixtures.yml"); err != nil {
		return nil, fmt.Errorf("failed to load fixtures: %w", err)
	}

	return fixture, nil
}

func NewTestDBConnection(db *bun.DB, fixture *dbfixture.Fixture) *TestDBConnection {
	return &TestDBConnection{db: db, fixture: fixture}
}

var (
	testDB     *TestDatabase
	testDBConn *TestDBConnection
	once       sync.Once
)

func GetTestDB() *TestDBConnection {
	once.Do(func() {
		testDB = NewTestDatabase(&testing.T{}, migrations.Migrations)
		testDBConn = NewTestDBConnection(testDB.DB, testDB.Fixture)
	})

	return testDBConn
}

func GetTestFixture() *dbfixture.Fixture {
	once.Do(func() {
		testDB = NewTestDatabase(&testing.T{}, migrations.Migrations)
		testDBConn = NewTestDBConnection(testDB.DB, testDB.Fixture)
	})

	return BunFixture
}

func FixtureMustRow(name string) any {
	fixture := GetTestFixture()
	return fixture.MustRow(name)
}

func CleanupTestDB() {
	if testDB != nil {
		testDB.Cleanup()
		testDB = nil
		testDBConn = nil
		once = sync.Once{}
	}
}
