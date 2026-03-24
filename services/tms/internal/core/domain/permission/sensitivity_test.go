package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldSensitivity_Level(t *testing.T) {
	t.Parallel()

	tests := []struct {
		sensitivity FieldSensitivity
		expected    int
	}{
		{SensitivityPublic, 0},
		{SensitivityInternal, 1},
		{SensitivityRestricted, 2},
		{SensitivityConfidential, 3},
		{FieldSensitivity("unknown"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.sensitivity), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.sensitivity.Level())
		})
	}
}

func TestFieldSensitivity_CanAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		user   FieldSensitivity
		target FieldSensitivity
		can    bool
	}{
		{"public user accessing public", SensitivityPublic, SensitivityPublic, true},
		{"public user accessing internal", SensitivityPublic, SensitivityInternal, false},
		{"public user accessing restricted", SensitivityPublic, SensitivityRestricted, false},
		{"public user accessing confidential", SensitivityPublic, SensitivityConfidential, false},

		{"internal user accessing public", SensitivityInternal, SensitivityPublic, true},
		{"internal user accessing internal", SensitivityInternal, SensitivityInternal, true},
		{"internal user accessing restricted", SensitivityInternal, SensitivityRestricted, false},
		{
			"internal user accessing confidential",
			SensitivityInternal,
			SensitivityConfidential,
			false,
		},

		{"restricted user accessing public", SensitivityRestricted, SensitivityPublic, true},
		{"restricted user accessing internal", SensitivityRestricted, SensitivityInternal, true},
		{
			"restricted user accessing restricted",
			SensitivityRestricted,
			SensitivityRestricted,
			true,
		},
		{
			"restricted user accessing confidential",
			SensitivityRestricted,
			SensitivityConfidential,
			false,
		},

		{"confidential user accessing public", SensitivityConfidential, SensitivityPublic, true},
		{
			"confidential user accessing internal",
			SensitivityConfidential,
			SensitivityInternal,
			true,
		},
		{
			"confidential user accessing restricted",
			SensitivityConfidential,
			SensitivityRestricted,
			true,
		},
		{
			"confidential user accessing confidential",
			SensitivityConfidential,
			SensitivityConfidential,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.can, tt.user.CanAccess(tt.target))
		})
	}
}

func TestFieldSensitivity_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		sensitivity FieldSensitivity
		expected    string
	}{
		{SensitivityPublic, "public"},
		{SensitivityInternal, "internal"},
		{SensitivityRestricted, "restricted"},
		{SensitivityConfidential, "confidential"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.sensitivity.String())
		})
	}
}

func TestFieldSensitivity_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		sensitivity FieldSensitivity
		valid       bool
	}{
		{SensitivityPublic, true},
		{SensitivityInternal, true},
		{SensitivityRestricted, true},
		{SensitivityConfidential, true},
		{FieldSensitivity("unknown"), false},
		{FieldSensitivity(""), false},
		{FieldSensitivity("INTERNAL"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.sensitivity), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.sensitivity.IsValid())
		})
	}
}

func TestDataScope_Level(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scope    DataScope
		expected int
	}{
		{DataScopeOwn, 0},
		{DataScopeOrganization, 1},
		{DataScopeAll, 2},
		{DataScope("unknown"), 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.scope.Level())
		})
	}
}

func TestDataScope_IsMorePermissive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		scope DataScope
		other DataScope
		more  bool
	}{
		{"own vs own", DataScopeOwn, DataScopeOwn, false},
		{"own vs organization", DataScopeOwn, DataScopeOrganization, false},
		{"own vs all", DataScopeOwn, DataScopeAll, false},

		{"organization vs own", DataScopeOrganization, DataScopeOwn, true},
		{"organization vs organization", DataScopeOrganization, DataScopeOrganization, false},
		{"organization vs all", DataScopeOrganization, DataScopeAll, false},

		{"all vs own", DataScopeAll, DataScopeOwn, true},
		{"all vs organization", DataScopeAll, DataScopeOrganization, true},
		{"all vs all", DataScopeAll, DataScopeAll, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.more, tt.scope.IsMorePermissive(tt.other))
		})
	}
}

func TestDataScope_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scope    DataScope
		expected string
	}{
		{DataScopeOwn, "own"},
		{DataScopeOrganization, "organization"},
		{DataScopeAll, "all"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.scope.String())
		})
	}
}

func TestDataScope_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scope DataScope
		valid bool
	}{
		{DataScopeOwn, true},
		{DataScopeOrganization, true},
		{DataScopeAll, true},
		{DataScope("unknown"), false},
		{DataScope(""), false},
		{DataScope("OWN"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, tt.scope.IsValid())
		})
	}
}
