// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package testutils

import (
	"context"
	"database/sql"
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

// ReadDB returns the test database for read operations
// In tests, we use the same database for both read and write
func (t *TestDBConnection) ReadDB(ctx context.Context) (*bun.DB, error) {
	return t.db, nil
}

// WriteDB returns the test database for write operations
// In tests, we use the same database for both read and write
func (t *TestDBConnection) WriteDB(ctx context.Context) (*bun.DB, error) {
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

func (t *TestDBConnection) SQLDB(ctx context.Context) (*sql.DB, error) {
	return t.db.DB, nil
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
