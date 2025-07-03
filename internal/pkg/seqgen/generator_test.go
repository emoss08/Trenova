package seqgen_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	sequenceType sequencestore.SequenceType,
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
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
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
					Type:                sequencestore.SequenceTypeProNumber,
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
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				// * Mock sequence store
				year := time.Now().Year()
				month := int(time.Now().Month())
				store.On("GetNextSequence", ctx, mock.MatchedBy(func(req *seqgen.SequenceRequest) bool {
					return req.Type == sequencestore.SequenceTypeProNumber &&
						req.OrganizationID == orgID &&
						req.BusinessUnitID == buID &&
						req.Year == year &&
						req.Month == month &&
						req.Count == 1
				})).
					Return(int64(1234), nil)
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
			},
			wantPrefix:   "S",
			wantContains: []string{"12", "1234"},
		},
		{
			name: "successful consolidation generation",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				format := &seqgen.Format{
					Type:                sequencestore.SequenceTypeConsolidation,
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
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeConsolidation, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequence", ctx, mock.Anything).Return(int64(5678), nil)
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeConsolidation,
				OrganizationID: orgID,
				BusinessUnitID: buID,
			},
			wantPrefix:   "C",
			wantContains: []string{"12", "5678"},
		},
		{
			name: "format provider error",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(nil, errors.New("format not found"))
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
			},
			wantError:     true,
			errorContains: "get format configuration",
		},
		{
			name: "sequence store error",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				format := &seqgen.Format{
					Type:           sequencestore.SequenceTypeProNumber,
					Prefix:         "S",
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequence", ctx, mock.Anything).
					Return(int64(0), errors.New("database error"))
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
			},
			wantError:     true,
			errorContains: "get next sequence",
		},
		{
			name: "invalid format causes generation error",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				// * Invalid format with no prefix
				format := &seqgen.Format{
					Type:           sequencestore.SequenceTypeProNumber,
					Prefix:         "", // Invalid
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequence", ctx, mock.Anything).Return(int64(1234), nil)
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
			},
			wantError:     true,
			errorContains: "generate sequence number",
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
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
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
					Type:           sequencestore.SequenceTypeProNumber,
					Prefix:         "S",
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				sequences := []int64{1001, 1002, 1003, 1004, 1005}
				store.On("GetNextSequenceBatch", ctx, mock.MatchedBy(func(req *seqgen.SequenceRequest) bool {
					return req.Count == 5
				})).
					Return(sequences, nil)
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Count:          5,
			},
			expectedCount: 5,
		},
		{
			name: "zero count returns empty slice",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				// * No mocks needed as it should return early
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Count:          0,
			},
			expectedCount: 0,
		},
		{
			name: "format provider error in batch",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(nil, errors.New("format error"))
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Count:          3,
			},
			wantError:     true,
			errorContains: "get format configuration",
		},
		{
			name: "sequence store error in batch",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				format := &seqgen.Format{
					Type:           sequencestore.SequenceTypeProNumber,
					Prefix:         "S",
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				store.On("GetNextSequenceBatch", ctx, mock.Anything).
					Return([]int64(nil), errors.New("batch error"))
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Count:          3,
			},
			wantError:     true,
			errorContains: "get sequence batch",
		},
		{
			name: "generation error in batch",
			setupMocks: func(store *mockSequenceStore, provider *mockFormatProvider) {
				// * Invalid format
				format := &seqgen.Format{
					Type:           sequencestore.SequenceTypeProNumber,
					Prefix:         "",
					SequenceDigits: 4,
				}
				provider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
					Return(format, nil)

				sequences := []int64{1001, 1002}
				store.On("GetNextSequenceBatch", ctx, mock.Anything).
					Return(sequences, nil)
			},
			request: &seqgen.GenerateRequest{
				Type:           sequencestore.SequenceTypeProNumber,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				Count:          2,
			},
			wantError:     true,
			errorContains: "generate sequence number",
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
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockStore := new(mockSequenceStore)
	mockProvider := new(mockFormatProvider)

	// * Setup format with date components
	format := &seqgen.Format{
		Type:              sequencestore.SequenceTypeProNumber,
		Prefix:            "D",
		IncludeYear:       true,
		YearDigits:        4,
		IncludeMonth:      true,
		IncludeWeekNumber: false,
		IncludeDay:        true,
		SequenceDigits:    4,
	}
	mockProvider.On("GetFormat", ctx, sequencestore.SequenceTypeProNumber, orgID, buID).
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
		Type:           sequencestore.SequenceTypeProNumber,
		OrganizationID: orgID,
		BusinessUnitID: buID,
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
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	// * Create a simple mock that returns predictable values
	mockStore := new(mockSequenceStore)
	mockProvider := new(mockFormatProvider)

	format := &seqgen.Format{
		Type:                sequencestore.SequenceTypeProNumber,
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
		Type:           sequencestore.SequenceTypeProNumber,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.Generate(ctx, request)
	}
}

func BenchmarkGenerator_GenerateBatch(b *testing.B) {
	ctx := context.Background()
	testLogger := logger.NewLogger(&config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "error"},
	})
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	mockStore := new(mockSequenceStore)
	mockProvider := new(mockFormatProvider)

	format := &seqgen.Format{
		Type:           sequencestore.SequenceTypeProNumber,
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
		Type:           sequencestore.SequenceTypeProNumber,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Count:          10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = generator.GenerateBatch(ctx, request)
	}
}
