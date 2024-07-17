// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package gen_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// MockCodeGeneratable implements CodeGeneratable for testing
type MockCodeGeneratable struct {
	tableName  string
	codePrefix string
	generateFn func(pattern string, counter int) string
}

func (m *MockCodeGeneratable) TableName() string {
	return m.tableName
}

func (m *MockCodeGeneratable) GetCodePrefix(pattern string) string {
	return m.codePrefix
}

func (m *MockCodeGeneratable) GenerateCode(pattern string, counter int) string {
	return m.generateFn(pattern, counter)
}

// TestModel represents the structure of our test table
type TestModel struct {
	bun.BaseModel `bun:"table:test_models,alias:tm"`

	ID             string `bun:"id,pk,type:TEXT"`
	Code           string `bun:"code,notnull"`
	OrganizationID string `bun:"organization_id,notnull,type:TEXT"`
}

func TestCodeGenerator_GenerateUniqueCode(t *testing.T) {
	testDB, cleanup := testutils.SetupTestCase(t)
	defer cleanup()

	ctx := context.Background()
	orgID := uuid.New().String()

	// Create a test table
	_, err := testDB.DB.NewCreateTable().Model((*TestModel)(nil)).Exec(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		setupModel    func() *MockCodeGeneratable
		setupDB       func(*testutils.TestDB)
		expectedCode  string
		expectedError string
	}{
		{
			name: "Successful unique code generation",
			setupModel: func() *MockCodeGeneratable {
				return &MockCodeGeneratable{
					tableName:  "test_models",
					codePrefix: "TST",
					generateFn: func(pattern string, counter int) string {
						return "TST0001000"
					},
				}
			},
			setupDB: func(db *testutils.TestDB) {
				// No setup needed for this test case
			},
			expectedCode: "TST0001000",
		},
		{
			name: "Code exists, then generates unique",
			setupModel: func() *MockCodeGeneratable {
				return &MockCodeGeneratable{
					tableName:  "test_models",
					codePrefix: "TST",
					generateFn: func(pattern string, counter int) string {
						if counter == 1 {
							return "TST0001000"
						}
						return "TST0002000"
					},
				}
			},
			setupDB: func(db *testutils.TestDB) {
				err = db.WithTransaction(func(tx *bun.Tx) error {
					_, err = tx.NewInsert().Model(&TestModel{
						ID:             uuid.New().String(),
						Code:           "TST0001000",
						OrganizationID: orgID,
					}).Exec(ctx)
					return err
				})
				require.NoError(t, err)
			},
			expectedCode: "TST0002000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setupDB(testDB)
			model := tt.setupModel()

			codeChecker := &gen.CodeChecker{DB: testDB.DB}
			cg := gen.NewCodeGenerator(gen.NewCounterManager(), codeChecker)

			// Test
			code, err := cg.GenerateUniqueCode(ctx, model, "TEST", uuid.MustParse(orgID))

			// Assert
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, code)
			}
		})
	}
}
