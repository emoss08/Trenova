package ai_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ai"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAIClassificationService is a mock implementation for testing
type MockAIClassificationService struct {
	mock.Mock
}

func (m *MockAIClassificationService) ClassifyLocation(ctx context.Context, req *ai.ClassificationRequest) (*ai.ClassificationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.ClassificationResponse), args.Error(1)
}

func (m *MockAIClassificationService) ClassifyLocationBatch(ctx context.Context, req *ai.BatchClassificationRequest) (*ai.BatchClassificationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ai.BatchClassificationResponse), args.Error(1)
}

func TestClassifyLocation(t *testing.T) {
	ctx := context.Background()
	mockService := new(MockAIClassificationService)

	tests := []struct {
		name     string
		request  *ai.ClassificationRequest
		expected *ai.ClassificationResponse
	}{
		{
			name: "Walmart Distribution Center",
			request: &ai.ClassificationRequest{
				Name:        "Walmart Distribution Center",
				Description: ptrString("Large warehouse facility for retail distribution"),
			},
			expected: &ai.ClassificationResponse{
				Category:   location.CategoryDistributionCenter,
				Confidence: 0.95,
				Reasoning:  "Walmart facility with 'Distribution Center' in name indicates retail distribution operations",
				AlternativeCategories: []ai.AlternativeCategory{
					{Category: location.CategoryWarehouse, Confidence: 0.75},
				},
			},
		},
		{
			name: "Love's Travel Stop",
			request: &ai.ClassificationRequest{
				Name:        "Love's Travel Stop",
				Description: ptrString("Truck stop with fuel and amenities"),
			},
			expected: &ai.ClassificationResponse{
				Category:   location.CategoryTruckStop,
				Confidence: 0.98,
				Reasoning:  "Love's is a well-known truck stop chain, name contains 'Travel Stop'",
				AlternativeCategories: []ai.AlternativeCategory{},
			},
		},
		{
			name: "Port of Los Angeles",
			request: &ai.ClassificationRequest{
				Name: "Port of Los Angeles",
			},
			expected: &ai.ClassificationResponse{
				Category:   location.CategoryPort,
				Confidence: 0.99,
				Reasoning:  "'Port of' prefix clearly indicates a seaport facility",
				AlternativeCategories: []ai.AlternativeCategory{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.On("ClassifyLocation", ctx, tt.request).Return(tt.expected, nil)

			result, err := mockService.ClassifyLocation(ctx, tt.request)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected.Category, result.Category)
			assert.Equal(t, tt.expected.Confidence, result.Confidence)
			assert.NotEmpty(t, result.Reasoning)

			mockService.AssertExpectations(t)
		})
	}
}

func TestClassifyLocationBatch(t *testing.T) {
	ctx := context.Background()
	mockService := new(MockAIClassificationService)

	request := &ai.BatchClassificationRequest{
		Locations: []ai.ClassificationRequest{
			{Name: "FedEx Freight Terminal"},
			{Name: "Pilot Flying J"},
			{Name: "BNSF Rail Yard"},
		},
	}

	expected := &ai.BatchClassificationResponse{
		Results: []ai.ClassificationResponse{
			{
				Category:   location.CategoryTerminal,
				Confidence: 0.90,
				Reasoning:  "FedEx Freight Terminal indicates a freight terminal facility",
			},
			{
				Category:   location.CategoryTruckStop,
				Confidence: 0.95,
				Reasoning:  "Pilot Flying J is a major truck stop chain",
			},
			{
				Category:   location.CategoryRailYard,
				Confidence: 0.93,
				Reasoning:  "BNSF is a railroad company, 'Rail Yard' indicates rail terminal",
			},
		},
	}

	mockService.On("ClassifyLocationBatch", ctx, request).Return(expected, nil)

	result, err := mockService.ClassifyLocationBatch(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Results, 3)

	for i, res := range result.Results {
		assert.Equal(t, expected.Results[i].Category, res.Category)
		assert.NotEmpty(t, res.Reasoning)
	}

	mockService.AssertExpectations(t)
}

// Helper function to create string pointers
func ptrString(s string) *string {
	return &s
}