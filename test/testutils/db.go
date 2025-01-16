package testutils

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const testDBName = "bun_test"

var (
	globalOnce sync.Once
	testDB     *bun.DB
)

// initTestDatabase ensures we have a connection to the test database
func initTestDatabase() error {
	var initErr error
	globalOnce.Do(func() {
		// Connect to test database
		dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", testDBName)
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		// Configure connection pool
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(4)
		db.SetConnMaxLifetime(time.Hour)

		// Test connection
		if err := db.Ping(); err != nil {
			initErr = fmt.Errorf("failed to connect to test database: %w", err)
			return
		}

		testDB = db
	})
	return initErr
}

// GetTestDB returns a connection to the test database
func GetTestDB() (*bun.DB, error) {
	if err := initTestDatabase(); err != nil {
		return nil, err
	}
	return testDB, nil
}

// CleanupTestDB closes the test database connection
func CleanupTestDB() {
	if testDB != nil {
		testDB.Close()
	}
}
