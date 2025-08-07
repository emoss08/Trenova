/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package testutils

import (
	"context"
	"fmt"
	"sync"

	"github.com/uptrace/bun/dbfixture"
)

type TestSetup struct {
	DB      *TestDBConnection
	Fixture *dbfixture.Fixture
	cleanup func()
}

var (
	testSetup *TestSetup
	setupOnce sync.Once
)

// NewTestSetup creates a new test setup with database and fixtures
func NewTestSetup(ctx context.Context) (*TestSetup, error) {
	var setupErr error

	setupOnce.Do(func() {
		db := GetTestDB()
		if db == nil {
			setupErr = fmt.Errorf("failed to initialize test database")
			return
		}

		fixture, err := db.Fixture(ctx)
		if err != nil {
			setupErr = fmt.Errorf("failed to create fixture: %w", err)
			return
		}

		testSetup = &TestSetup{
			DB:      db,
			Fixture: fixture,
			cleanup: CleanupTestDB,
		}
	})

	if setupErr != nil {
		return nil, setupErr
	}

	return testSetup, nil
}

// Cleanup cleans up the test resources
func (ts *TestSetup) Cleanup() {
	if ts.cleanup != nil {
		ts.cleanup()
	}
}
