// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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

func (m *MockCodeGeneratable) GetCodePrefix(_ string) string {
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
	server, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	orgID := uuid.New().String()

	// Create a test table
	_, err := server.DB.NewCreateTable().Model((*TestModel)(nil)).Exec(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		setupModel    func() *MockCodeGeneratable
		setupDB       func(db *bun.DB)
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
			setupDB: func(db *bun.DB) {
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
			setupDB: func(db *bun.DB) {
				err = db.RunInTx(ctx, nil, func(_ context.Context, tx bun.Tx) error {
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
			tt.setupDB(server.DB)
			model := tt.setupModel()

			codeChecker := &gen.CodeChecker{DB: server.DB}
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
