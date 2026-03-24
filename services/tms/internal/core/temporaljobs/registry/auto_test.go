package registry

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type TestActivities struct{}

func (a *TestActivities) ProcessDataActivity(ctx context.Context, input string) (string, error) {
	return "processed: " + input, nil
}

func (a *TestActivities) FetchRecordsActivity(ctx context.Context) ([]string, error) {
	return []string{"record1", "record2"}, nil
}

func (a *TestActivities) SendNotificationActivity(ctx context.Context, msg string) error {
	return nil
}

func (a *TestActivities) HandleRequestActivity(ctx context.Context) error {
	return nil
}

func (a *TestActivities) SomeOtherMethod(ctx context.Context) error {
	return nil
}

func newTestLogger() *zap.Logger {
	return zap.NewNop()
}

type mockWorker struct {
	registeredActivities []any
}

func (m *mockWorker) RegisterActivity(a any) {
	m.registeredActivities = append(m.registeredActivities, a)
}

func (m *mockWorker) RegisterActivityWithOptions(a any, options any) {
	m.registeredActivities = append(m.registeredActivities, a)
}

func (m *mockWorker) RegisterWorkflow(w any) {}

func (m *mockWorker) RegisterWorkflowWithOptions(w any, options any) {}

func TestIsActivityMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		expected bool
	}{
		{"Activity suffix", "ProcessDataActivity", true},
		{"Do prefix", "DoSomething", true},
		{"Process prefix", "ProcessData", true},
		{"Execute prefix", "ExecuteTask", true},
		{"Handle prefix", "HandleRequest", true},
		{"Fetch prefix", "FetchRecords", true},
		{"Send prefix", "SendNotification", true},
		{"Create prefix", "CreateRecord", true},
		{"Update prefix", "UpdateRecord", true},
		{"Delete prefix", "DeleteRecord", true},
		{"Get prefix", "GetRecord", true},
		{"No match", "SomeMethod", false},
		{"No match 2", "AnotherMethod", false},
	}

	type testStruct struct{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := reflect.Method{
				Name: tt.method,
			}
			result := isActivityMethod(&method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsActivityMethod_UnexportedMethod(t *testing.T) {
	method := reflect.Method{
		Name: "privateMethod",
	}
	result := isActivityMethod(&method)
	assert.False(t, result)
}

func TestRegisterActivitiesFromStruct_CountsCorrectly(t *testing.T) {
	worker := &mockWorker{}
	activities := &TestActivities{}
	logger := newTestLogger()

	count, err := RegisterActivitiesFromStruct(worker, activities, logger)

	require.NoError(t, err)
	assert.Equal(t, 4, count)
	assert.Len(t, worker.registeredActivities, 4)
}

func TestRegisterActivitiesFromStruct_NilActivities(t *testing.T) {
	worker := &mockWorker{}
	logger := newTestLogger()

	count, err := RegisterActivitiesFromStruct(worker, nil, logger)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

type MinimalActivities struct{}

func (a *MinimalActivities) DoSomething(ctx context.Context) error {
	return nil
}

func (a *MinimalActivities) ExecuteTask(ctx context.Context) error {
	return nil
}

func (a *MinimalActivities) CreateRecord(ctx context.Context) error {
	return nil
}

func (a *MinimalActivities) UpdateRecord(ctx context.Context) error {
	return nil
}

func (a *MinimalActivities) DeleteRecord(ctx context.Context) error {
	return nil
}

func (a *MinimalActivities) GetRecord(ctx context.Context) error {
	return nil
}

func TestRegisterActivitiesFromStruct_AllPrefixes(t *testing.T) {
	worker := &mockWorker{}
	activities := &MinimalActivities{}
	logger := newTestLogger()

	count, err := RegisterActivitiesFromStruct(worker, activities, logger)

	require.NoError(t, err)
	assert.Equal(t, 6, count)
}

type NoActivities struct{}

func (a *NoActivities) SomeMethod(ctx context.Context) error {
	return nil
}

func (a *NoActivities) AnotherMethod(ctx context.Context) error {
	return nil
}

func TestRegisterActivitiesFromStruct_NoMatchingMethods(t *testing.T) {
	worker := &mockWorker{}
	activities := &NoActivities{}
	logger := newTestLogger()

	count, err := RegisterActivitiesFromStruct(worker, activities, logger)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
