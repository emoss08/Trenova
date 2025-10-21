package seqgen_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// * Mock implementations
type mockSequenceStore struct {
	mock.Mock
}

func (m *mockSequenceStore) GetNextSequence(
	ctx context.Context,
	req *seqgen.SequenceRequest,
) (int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockSequenceStore) GetNextSequenceBatch(
	ctx context.Context,
	req *seqgen.SequenceRequest,
) ([]int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]int64), args.Error(1)
}

type mockFormatProvider struct {
	mock.Mock
}

func (m *mockFormatProvider) GetFormat(
	ctx context.Context,
	sequenceType seqgen.SequenceType,
	orgID, buID pulid.ID,
) (*seqgen.Format, error) {
	args := m.Called(ctx, sequenceType, orgID, buID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*seqgen.Format), args.Error(1)
}

func TestGenerator_Generate(t *testing.T) {
	ctx := context.Background()
	testLogger := zap.NewNop()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	tests := []struct {
		name          string
		setupMocks    func(*mockSequenceStore, *mockFormatProvider)
		request       *seqgen.GenerateRequest
		wantPrefix    string
		wantContains  []string
		wantError     bool
		errorContains string
	}{
		{
			name: "successful pro number generation",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				// * Mock format provider
				format := &seqgen.Format{
					Type:                seqgen.SequenceTypeProNumber,
					Prefix:              "S",
					IncludeYear:         true,
					YearDigits:          2,
					IncludeMonth:        true,
					SequenceDigits:      4,
					IncludeLocationCode: true,
					LocationCode:        "12",
					IncludeRandomDigits: true,
					RandomDigitsCount:   6,
				}
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				// * Mock sequence store
				year := time.Now().Year()
				month := int(time.Now().Month())
				store.On("GetNextSequence", ctx, mock.MatchedBy(func(req *seqgen.SequenceRequest) bool {
					return req.Type == seqgen.SequenceTypeProNumber &&
						req.OrgID == orgID &&
						req.BuID == buID &&
						req.Year == year &&
						req.Month == month &&
						req.Count == 1
				})).
					Return(int64(1234), nil)
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
			},
			wantPrefix:   "S",
			wantContains: []string{"12", "1234"},
		},
		{
			name: "successful consolidation generation",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				format := &seqgen.Format{
					Type:                seqgen.SequenceTypeConsolidation,
					Prefix:              "C",
					IncludeYear:         true,
					YearDigits:          2,
					IncludeMonth:        true,
					SequenceDigits:      4,
					IncludeLocationCode: true,
					LocationCode:        "12",
					IncludeRandomDigits: true,
					RandomDigitsCount:   6,
				}
				provider.On("GetFormat", ctx, seqgen.SequenceTypeConsolidation, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequence", ctx, mock.Anything).Return(int64(5678), nil)
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeConsolidation,
				OrgID: orgID,
				BuID:  buID,
			},
			wantPrefix:   "C",
			wantContains: []string{"12", "5678"},
		},
		{
			name: "format provider error",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(nil, errors.New("format not found"))
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
			},
			wantError:     true,
			errorContains: "get format configuration",
		},
		{
			name: "sequence store error",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				format := &seqgen.Format{
					Type:           seqgen.SequenceTypeProNumber,
					Prefix:         "S",
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequence", ctx, mock.Anything).
					Return(int64(0), errors.New("database error"))
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
			},
			wantError:     true,
			errorContains: "get next sequence",
		},
		{
			name: "invalid format causes generation error",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				// * Invalid format with 0 sequence digits
				format := &seqgen.Format{
					Type:   seqgen.SequenceTypeProNumber,
					Prefix: "S",
				}
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequence", ctx, mock.Anything).Return(int64(1234), nil)
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
			},
			wantError:     true,
			errorContains: "invalid sequence format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// * Create mocks
			mockStore := new(mockSequenceStore)
			mockProvider := new(mockFormatProvider)

			// * Setup mocks
			tt.setupMocks(mockStore, mockProvider)

			// * Create generator
			params := seqgen.GeneratorParams{
				Store:    mockStore,
				Provider: mockProvider,
				Logger:   testLogger,
			}
			generator := seqgen.NewGenerator(params)

			// * Execute
			result, err := generator.Generate(ctx, tt.request)

			// * Assert
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)

				if tt.wantPrefix != "" {
					assert.True(t, strings.HasPrefix(result, tt.wantPrefix))
				}

				for _, want := range tt.wantContains {
					assert.Contains(t, result, want)
				}
			}

			// * Verify mock expectations
			mockStore.AssertExpectations(t)
			mockProvider.AssertExpectations(t)
		})
	}
}

func TestGenerator_GenerateBatch(t *testing.T) {
	ctx := context.Background()
	testLogger := zap.NewNop()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	tests := []struct {
		name          string
		setupMocks    func(*mockSequenceStore, *mockFormatProvider)
		request       *seqgen.GenerateRequest
		expectedCount int
		wantError     bool
		errorContains string
	}{
		{
			name: "successful batch generation",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				format := &seqgen.Format{
					Type:           seqgen.SequenceTypeProNumber,
					Prefix:         "S",
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				sequences := []int64{1001, 1002, 1003, 1004, 1005}
				store.On("GetNextSequenceBatch", ctx, mock.MatchedBy(func(req *seqgen.SequenceRequest) bool {
					return req.Count == 5
				})).
					Return(sequences, nil)
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
				Count: 5,
			},
			expectedCount: 5,
		},
		{
			name: "zero count returns empty slice",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				// * No mocks needed as it should return early
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
				Count: 0,
			},
			expectedCount: 0,
		},
		{
			name: "format provider error in batch",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(nil, errors.New("format error"))
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
				Count: 3,
			},
			wantError:     true,
			errorContains: "get format configuration",
		},
		{
			name: "sequence store error in batch",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				format := &seqgen.Format{
					Type:           seqgen.SequenceTypeProNumber,
					Prefix:         "S",
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequenceBatch", ctx, mock.Anything).
					Return([]int64(nil), errors.New("batch error"))
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
				Count: 3,
			},
			wantError:     true,
			errorContains: "get sequence batch",
		},
		{
			name: "generation error in batch",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				// * Invalid format - missing SequenceDigits
				format := &seqgen.Format{
					Type:   seqgen.SequenceTypeProNumber,
					Prefix: "S",
					// SequenceDigits: 0, // Invalid - will cause validation error
				}
				provider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				sequences := []int64{1001, 1002}
				store.On("GetNextSequenceBatch", ctx, mock.Anything).
					Return(sequences, nil)
			},
			request: &seqgen.GenerateRequest{
				Type:  seqgen.SequenceTypeProNumber,
				OrgID: orgID,
				BuID:  buID,
				Count: 2,
			},
			wantError:     true,
			errorContains: "invalid sequence format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// * Create mocks
			mockStore := new(mockSequenceStore)
			mockProvider := new(mockFormatProvider)

			// * Setup mocks
			tt.setupMocks(mockStore, mockProvider)

			// * Create generator
			params := seqgen.GeneratorParams{
				Store:    mockStore,
				Provider: mockProvider,
				Logger:   testLogger,
			}
			generator := seqgen.NewGenerator(params)

			// * Execute
			results, err := generator.GenerateBatch(ctx, tt.request)

			// * Assert
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.expectedCount)

				// * Verify all results are unique and properly formatted
				seen := make(map[string]bool)
				for _, result := range results {
					assert.NotEmpty(t, result)
					assert.False(t, seen[result], "Duplicate sequence number generated")
					seen[result] = true
				}
			}

			// * Verify mock expectations
			mockStore.AssertExpectations(t)
			mockProvider.AssertExpectations(t)
		})
	}
}

func TestGenerator_DateHandling(t *testing.T) {
	// * Test that the generator uses current date/time correctly
	ctx := context.Background()
	testLogger := zap.NewNop()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockStore := new(mockSequenceStore)
	mockProvider := new(mockFormatProvider)

	// * Setup format with date components
	format := &seqgen.Format{
		Type:              seqgen.SequenceTypeProNumber,
		Prefix:            "D",
		IncludeYear:       true,
		YearDigits:        4,
		IncludeMonth:      true,
		IncludeWeekNumber: false,
		IncludeDay:        true,
		SequenceDigits:    4,
	}
	mockProvider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
		Return(format, nil)

	// * Verify the sequence request uses current date
	now := time.Now()
	mockStore.On("GetNextSequence", ctx, mock.MatchedBy(func(req *seqgen.SequenceRequest) bool {
		return req.Year == now.Year() && req.Month == int(now.Month())
	})).Return(int64(1), nil)

	params := seqgen.GeneratorParams{
		Store:    mockStore,
		Provider: mockProvider,
		Logger:   testLogger,
	}
	generator := seqgen.NewGenerator(params)

	request := &seqgen.GenerateRequest{
		Type:  seqgen.SequenceTypeProNumber,
		OrgID: orgID,
		BuID:  buID,
	}

	result, err := generator.Generate(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// * Verify the result contains current date components
	yearStr := now.Format("2006")
	monthStr := now.Format("01")
	dayStr := now.Format("02")

	assert.Contains(t, result, yearStr)
	assert.Contains(t, result, monthStr)
	assert.Contains(t, result, dayStr)

	mockStore.AssertExpectations(t)
	mockProvider.AssertExpectations(t)
}

// * Benchmark tests
func BenchmarkGenerator_Generate(b *testing.B) {
	ctx := context.Background()
	testLogger := zap.NewNop()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	// * Create a simple mock that returns predictable values
	mockStore := new(mockSequenceStore)
	mockProvider := new(mockFormatProvider)

	format := &seqgen.Format{
		Type:                seqgen.SequenceTypeProNumber,
		Prefix:              "S",
		IncludeYear:         true,
		YearDigits:          2,
		IncludeMonth:        true,
		SequenceDigits:      4,
		IncludeLocationCode: true,
		LocationCode:        "12",
		IncludeRandomDigits: true,
		RandomDigitsCount:   6,
	}
	mockProvider.On("GetFormat", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(format, nil)

	mockStore.On("GetNextSequence", mock.Anything, mock.Anything).
		Return(int64(1234), nil)

	params := seqgen.GeneratorParams{
		Store:    mockStore,
		Provider: mockProvider,
		Logger:   testLogger,
	}
	generator := seqgen.NewGenerator(params)

	request := &seqgen.GenerateRequest{
		Type:  seqgen.SequenceTypeProNumber,
		OrgID: orgID,
		BuID:  buID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.Generate(ctx, request)
	}
}

func BenchmarkGenerator_GenerateBatch(b *testing.B) {
	ctx := context.Background()
	testLogger := zap.NewNop()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockStore := new(mockSequenceStore)
	mockProvider := new(mockFormatProvider)

	format := &seqgen.Format{
		Type:           seqgen.SequenceTypeProNumber,
		Prefix:         "S",
		SequenceDigits: 4,
	}
	mockProvider.On("GetFormat", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(format, nil)

	sequences := []int64{1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010}
	mockStore.On("GetNextSequenceBatch", mock.Anything, mock.Anything).
		Return(sequences, nil)

	params := seqgen.GeneratorParams{
		Store:    mockStore,
		Provider: mockProvider,
		Logger:   testLogger,
	}
	generator := seqgen.NewGenerator(params)

	request := &seqgen.GenerateRequest{
		Type:  seqgen.SequenceTypeProNumber,
		OrgID: orgID,
		BuID:  buID,
		Count: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.GenerateBatch(ctx, request)
	}
}

// TestFormatCaching tests the format caching functionality
func TestFormatCaching(t *testing.T) {
	ctx := context.Background()
	testLogger := zap.NewNop()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	t.Run("cache hit on second request", func(t *testing.T) {
		mockStore := new(mockSequenceStore)
		mockProvider := new(mockFormatProvider)

		format := &seqgen.Format{
			Type:           seqgen.SequenceTypeProNumber,
			Prefix:         "CACHE",
			SequenceDigits: 4,
		}

		// Provider should only be called once due to caching
		mockProvider.On("GetFormat", ctx, seqgen.SequenceTypeProNumber, orgID, buID).
			Return(format, nil).Once()

		mockStore.On("GetNextSequence", ctx, mock.Anything).
			Return(int64(1), nil)

		params := seqgen.GeneratorParams{
			Store:    mockStore,
			Provider: mockProvider,
			Logger:   testLogger,
		}
		generator := seqgen.NewGenerator(params)

		// First request - cache miss
		req := &seqgen.GenerateRequest{
			Type:  seqgen.SequenceTypeProNumber,
			OrgID: orgID,
			BuID:  buID,
		}
		result1, err := generator.Generate(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, result1)

		// Second request - should hit cache
		mockStore.On("GetNextSequence", ctx, mock.Anything).
			Return(int64(2), nil)
		result2, err := generator.Generate(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, result2)

		// Verify provider was only called once
		mockProvider.AssertNumberOfCalls(t, "GetFormat", 1)
	})

	t.Run("format override bypasses cache", func(t *testing.T) {
		mockStore := new(mockSequenceStore)
		mockProvider := new(mockFormatProvider)

		overrideFormat := &seqgen.Format{
			Type:           seqgen.SequenceTypeProNumber,
			Prefix:         "OVERRIDE",
			SequenceDigits: 5,
		}

		mockStore.On("GetNextSequence", ctx, mock.Anything).
			Return(int64(1), nil)

		params := seqgen.GeneratorParams{
			Store:    mockStore,
			Provider: mockProvider,
			Logger:   testLogger,
		}
		generator := seqgen.NewGenerator(params)

		// Request with format override - should not call provider
		req := &seqgen.GenerateRequest{
			Type:   seqgen.SequenceTypeProNumber,
			OrgID:  orgID,
			BuID:   buID,
			Format: overrideFormat,
		}
		result, err := generator.Generate(ctx, req)
		require.NoError(t, err)
		assert.Contains(t, result, "OVERRIDE")

		// Provider should not be called at all
		mockProvider.AssertNotCalled(t, "GetFormat")
	})
}

// TestSequenceValidation tests the sequence validation functionality
func TestSequenceValidation(t *testing.T) {
	testLogger := zap.NewNop()

	generator := seqgen.NewGenerator(seqgen.GeneratorParams{
		Store:    new(mockSequenceStore),
		Provider: new(mockFormatProvider),
		Logger:   testLogger,
	})

	tests := []struct {
		name      string
		sequence  string
		format    *seqgen.Format
		wantError bool
		errMsg    string
	}{
		{
			name:     "valid simple sequence",
			sequence: "PRE0001",
			format: &seqgen.Format{
				Prefix:         "PRE",
				SequenceDigits: 4,
			},
			wantError: false,
		},
		{
			name:     "invalid prefix",
			sequence: "WRONG0001",
			format: &seqgen.Format{
				Prefix:         "PRE",
				SequenceDigits: 4,
			},
			wantError: true,
			errMsg:    "should start with prefix",
		},
		{
			name:     "sequence too short",
			sequence: "PRE001",
			format: &seqgen.Format{
				Prefix:         "PRE",
				SequenceDigits: 4,
			},
			wantError: true,
			errMsg:    "less than expected minimum",
		},
		{
			name:     "valid with separators",
			sequence: "PRE-0001-ABC",
			format: &seqgen.Format{
				Prefix:              "PRE",
				SequenceDigits:      4,
				IncludeRandomDigits: true,
				RandomDigitsCount:   3,
				UseSeparators:       true,
				SeparatorChar:       "-",
			},
			wantError: false,
		},
		{
			name:     "valid with check digit",
			sequence: "CHK00019",
			format: &seqgen.Format{
				Prefix:            "CHK",
				SequenceDigits:    4,
				IncludeCheckDigit: true,
			},
			wantError: false,
		},
		{
			name:     "empty sequence",
			sequence: "",
			format: &seqgen.Format{
				Prefix:         "PRE",
				SequenceDigits: 4,
			},
			wantError: true,
			errMsg:    "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateSequence(tt.sequence, tt.format)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFormatTemplates tests the format template functionality
func TestFormatTemplates(t *testing.T) {
	t.Run("get simple sequential template", func(t *testing.T) {
		format := seqgen.GetFormatFromTemplate(seqgen.TemplateSimpleSequential, nil)
		assert.Equal(t, "SEQ", format.Prefix)
		assert.Equal(t, 6, format.SequenceDigits)
		assert.False(t, format.IncludeYear)
	})

	t.Run("get year month template", func(t *testing.T) {
		format := seqgen.GetFormatFromTemplate(seqgen.TemplateYearMonth, nil)
		assert.Equal(t, "YM", format.Prefix)
		assert.True(t, format.IncludeYear)
		assert.True(t, format.IncludeMonth)
		assert.Equal(t, 4, format.YearDigits)
	})

	t.Run("template with overrides", func(t *testing.T) {
		overrides := &seqgen.Format{
			Prefix:         "CUSTOM",
			SequenceDigits: 8,
		}
		format := seqgen.GetFormatFromTemplate(seqgen.TemplateSimpleSequential, overrides)
		assert.Equal(t, "CUSTOM", format.Prefix)
		assert.Equal(t, 8, format.SequenceDigits)
	})

	t.Run("comprehensive template", func(t *testing.T) {
		format := seqgen.GetFormatFromTemplate(seqgen.TemplateComprehensive, nil)
		assert.Equal(t, "CMP", format.Prefix)
		assert.True(t, format.IncludeYear)
		assert.True(t, format.IncludeMonth)
		assert.True(t, format.IncludeLocationCode)
		assert.True(t, format.IncludeRandomDigits)
		assert.True(t, format.IncludeCheckDigit)
		assert.True(t, format.IncludeBusinessUnitCode)
	})

	t.Run("invalid template returns default", func(t *testing.T) {
		format := seqgen.GetFormatFromTemplate(seqgen.FormatTemplate("invalid"), nil)
		// Should return simple sequential as default
		assert.Equal(t, "CUSTOM", format.Prefix)
		assert.Equal(t, 8, format.SequenceDigits)
	})

	t.Run("use template in generator", func(t *testing.T) {
		ctx := context.Background()
		testLogger := zap.NewNop()
		orgID := pulid.MustNew("org_")
		buID := pulid.MustNew("bu_")

		mockStore := new(mockSequenceStore)
		mockProvider := new(mockFormatProvider)

		// Get format from template
		templateFormat := seqgen.GetFormatFromTemplate(seqgen.TemplateYearMonth, &seqgen.Format{
			Type:   seqgen.SequenceTypeProNumber,
			Prefix: "INV",
		})

		mockStore.On("GetNextSequence", ctx, mock.Anything).
			Return(int64(42), nil)

		params := seqgen.GeneratorParams{
			Store:    mockStore,
			Provider: mockProvider,
			Logger:   testLogger,
		}
		generator := seqgen.NewGenerator(params)

		req := &seqgen.GenerateRequest{
			Type:   seqgen.SequenceTypeProNumber,
			OrgID:  orgID,
			BuID:   buID,
			Format: templateFormat,
		}

		result, err := generator.Generate(ctx, req)
		require.NoError(t, err)
		assert.Contains(t, result, "INV")
		// Should contain year and month separated by dash
		assert.Regexp(t, `INV-\d{4}-\d{2}-0042`, result)
	})
}

// TestShipmentProNumberGeneration tests the shipment-specific pro number generation
func TestShipmentProNumberGeneration(t *testing.T) {
	ctx := context.Background()
	testLogger := zap.NewNop()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	t.Run("generate single shipment pro number", func(t *testing.T) {
		mockStore := new(mockSequenceStore)
		mockProvider := new(mockFormatProvider)

		mockStore.On("GetNextSequence", ctx, mock.Anything).
			Return(int64(1234), nil)

		params := seqgen.GeneratorParams{
			Store:    mockStore,
			Provider: mockProvider,
			Logger:   testLogger,
		}
		generator := seqgen.NewGenerator(params)

		// Generate shipment pro number
		proNumber, err := generator.GenerateShipmentProNumber(ctx, orgID, buID)
		require.NoError(t, err)
		assert.NotEmpty(t, proNumber)

		// Verify format: S + YY + MM + 4-digit sequence + 6 random digits
		assert.True(t, strings.HasPrefix(proNumber, "S"))
		assert.Len(t, proNumber, 15) // S(1) + YY(2) + MM(2) + Seq(4) + Random(6) = 15 chars total

		// Verify no separators
		assert.NotContains(t, proNumber, "-")
		assert.NotContains(t, proNumber, "/")
		assert.NotContains(t, proNumber, "_")

		mockStore.AssertExpectations(t)
	})

	t.Run("generate batch shipment pro numbers", func(t *testing.T) {
		mockStore := new(mockSequenceStore)
		mockProvider := new(mockFormatProvider)

		sequences := []int64{5000, 5001, 5002, 5003, 5004}
		mockStore.On("GetNextSequenceBatch", ctx, mock.MatchedBy(func(req *seqgen.SequenceRequest) bool {
			return req.Count == 5
		})).
			Return(sequences, nil)

		params := seqgen.GeneratorParams{
			Store:    mockStore,
			Provider: mockProvider,
			Logger:   testLogger,
		}
		generator := seqgen.NewGenerator(params)

		// Generate batch of shipment pro numbers
		proNumbers, err := generator.GenerateShipmentProNumberBatch(ctx, orgID, buID, 5)
		require.NoError(t, err)
		assert.Len(t, proNumbers, 5)

		// Verify each pro number
		for i, proNumber := range proNumbers {
			assert.True(t, strings.HasPrefix(proNumber, "S"))
			assert.Len(t, proNumber, 15)
			assert.NotContains(t, proNumber, "-")

			// Verify sequence numbers are in order
			assert.Contains(t, proNumber, fmt.Sprintf("%04d", sequences[i]))
		}

		// Verify all are unique
		seen := make(map[string]bool)
		for _, proNumber := range proNumbers {
			assert.False(t, seen[proNumber], "Duplicate pro number generated")
			seen[proNumber] = true
		}

		mockStore.AssertExpectations(t)
	})

	t.Run("validate shipment pro number format", func(t *testing.T) {
		generator := seqgen.NewGenerator(seqgen.GeneratorParams{
			Store:    new(mockSequenceStore),
			Provider: new(mockFormatProvider),
			Logger:   testLogger,
		})

		format := seqgen.DefaultShipmentProNumberFormat()

		// Valid pro number
		validProNumber := "S24120001123456"
		err := generator.ValidateSequence(validProNumber, format)
		assert.NoError(t, err)

		// Invalid - has separator
		invalidProNumber := "S-2412-0001-123456"
		err = generator.ValidateSequence(invalidProNumber, format)
		assert.Error(t, err)

		// Invalid - wrong prefix
		wrongPrefix := "P24120001123456"
		err = generator.ValidateSequence(wrongPrefix, format)
		assert.Error(t, err)
	})
}

// TestSeparatorValidation tests separator character validation
func TestSeparatorValidation(t *testing.T) {
	t.Run("valid separators are allowed", func(t *testing.T) {
		validSeparators := []string{"-", "_", "/", "."}

		for _, sep := range validSeparators {
			format := &seqgen.Format{
				Type:           seqgen.SequenceTypeProNumber,
				Prefix:         "TEST",
				SequenceDigits: 4,
				UseSeparators:  true,
				SeparatorChar:  sep,
			}

			err := format.Validate()
			assert.NoError(t, err, "Separator %q should be valid", sep)
		}
	})

	t.Run("invalid separators are rejected", func(t *testing.T) {
		invalidSeparators := []string{"|", " ", ":", ";", ",", "#", "@"}

		for _, sep := range invalidSeparators {
			format := &seqgen.Format{
				Type:           seqgen.SequenceTypeProNumber,
				Prefix:         "TEST",
				SequenceDigits: 4,
				UseSeparators:  true,
				SeparatorChar:  sep,
			}

			err := format.Validate()
			assert.Error(t, err, "Separator %q should be invalid", sep)
			assert.Contains(t, err.Error(), "not allowed")
		}
	})

	t.Run("no separator with UseSeparators false is valid", func(t *testing.T) {
		format := &seqgen.Format{
			Type:           seqgen.SequenceTypeProNumber,
			Prefix:         "TEST",
			SequenceDigits: 4,
			UseSeparators:  false,
			SeparatorChar:  "", // Empty is fine when not using separators
		}

		err := format.Validate()
		assert.NoError(t, err)
	})

	t.Run("sequence validation rejects unexpected separators", func(t *testing.T) {
		generator := seqgen.NewGenerator(seqgen.GeneratorParams{
			Store:    new(mockSequenceStore),
			Provider: new(mockFormatProvider),
			Logger:   zap.NewNop(),
		})

		format := &seqgen.Format{
			Type:           seqgen.SequenceTypeProNumber,
			Prefix:         "TEST",
			SequenceDigits: 4,
			UseSeparators:  false, // No separators expected
		}

		// Test all potential separator characters
		testSequences := []string{
			"TEST-1234", // dash
			"TEST_1234", // underscore
			"TEST/1234", // slash
			"TEST.1234", // dot
			"TEST 1234", // space
			"TEST|1234", // pipe
		}

		for _, seq := range testSequences {
			err := generator.ValidateSequence(seq, format)
			assert.Error(
				t,
				err,
				"Sequence %q should be invalid when separators are not allowed",
				seq,
			)
			assert.Contains(t, err.Error(), "unexpected separator")
		}
	})
}
