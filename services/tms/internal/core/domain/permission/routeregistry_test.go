package permission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRouteRegistry(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()
	require.NotNil(t, rr)
	assert.True(t, rr.Count() > 0)
}

func TestNewEmptyRouteRegistry(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()
	require.NotNil(t, rr)
	assert.Equal(t, 0, rr.Count())
}

func TestRouteRegistry_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		def       *RouteDefinition
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid exact route",
			def: &RouteDefinition{
				Path:      "/test/path",
				MatchType: RouteMatchExact,
				Requirements: []RouteRequirement{
					{Resource: ResourceUser, Operation: OpRead},
				},
			},
			expectErr: false,
		},
		{
			name: "valid prefix route",
			def: &RouteDefinition{
				Path:      "/api/v1/:id",
				MatchType: RouteMatchPrefix,
				Requirements: []RouteRequirement{
					{Resource: ResourceUser, Operation: OpRead},
				},
			},
			expectErr: false,
		},
		{
			name: "valid pattern route",
			def: &RouteDefinition{
				Path:      "/resources/:id/edit",
				MatchType: RouteMatchPattern,
				Requirements: []RouteRequirement{
					{Resource: ResourceUser, Operation: OpUpdate},
				},
			},
			expectErr: false,
		},
		{
			name: "empty path",
			def: &RouteDefinition{
				Path:      "",
				MatchType: RouteMatchExact,
				Requirements: []RouteRequirement{
					{Resource: ResourceUser, Operation: OpRead},
				},
			},
			expectErr: true,
			errMsg:    "route path is required",
		},
		{
			name: "no requirements",
			def: &RouteDefinition{
				Path:         "/empty/reqs",
				MatchType:    RouteMatchExact,
				Requirements: []RouteRequirement{},
			},
			expectErr: true,
			errMsg:    "at least one route requirement is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := NewEmptyRouteRegistry()
			err := rr.Register(tt.def)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRouteRegistry_Register_DuplicateExact(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	def := &RouteDefinition{
		Path:      "/users",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	}

	err := rr.Register(def)
	require.NoError(t, err)

	err = rr.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRouteRegistry_Match(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/users",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		DisplayName: "Users List",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/users/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		DisplayName: "User Details",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/users/:id/edit",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpUpdate},
		},
		DisplayName: "Edit User",
	})

	tests := []struct {
		name        string
		path        string
		found       bool
		displayName string
	}{
		{"exact match", "/users", true, "Users List"},
		{"pattern match single param", "/users/abc123", true, "User Details"},
		{"pattern match with suffix", "/users/abc123/edit", true, "Edit User"},
		{"no match", "/unknown", false, ""},
		{"partial no match", "/user", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def, ok := rr.Match(tt.path)
			assert.Equal(t, tt.found, ok)
			if tt.found {
				assert.Equal(t, tt.displayName, def.DisplayName)
			}
		})
	}
}

func TestRouteRegistry_Match_PrefixRoutes(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/api/:version/resources",
		MatchType: RouteMatchPrefix,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		DisplayName: "API Resources",
	})

	def, ok := rr.Match("/api/v1/resources")
	assert.True(t, ok)
	assert.Equal(t, "API Resources", def.DisplayName)

	_, ok = rr.Match("/api/v2/other")
	assert.False(t, ok)
}

func TestRouteRegistry_Get(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/exact",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		DisplayName: "Exact Route",
	})

	def, ok := rr.Get("/exact")
	assert.True(t, ok)
	assert.Equal(t, "Exact Route", def.DisplayName)

	_, ok = rr.Get("/nonexistent")
	assert.False(t, ok)
}

func TestRouteRegistry_All(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/exact",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/pattern/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/prefix/:ver/stuff",
		MatchType: RouteMatchPrefix,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})

	all := rr.All()
	assert.Len(t, all, 3)
}

func TestRouteRegistry_GetByCategory(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		Category: "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/roles",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceRole, Operation: OpRead},
		},
		Category: "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/users/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		Category: "Administration",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/billing/invoices",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceInvoice, Operation: OpRead},
		},
		Category: "Billing",
	})

	adminRoutes := rr.GetByCategory("Administration")
	assert.Len(t, adminRoutes, 3)

	billingRoutes := rr.GetByCategory("Billing")
	assert.Len(t, billingRoutes, 1)

	unknownRoutes := rr.GetByCategory("Unknown")
	assert.Empty(t, unknownRoutes)
}

func TestRouteRegistry_Count(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()
	assert.Equal(t, 0, rr.Count())

	_ = rr.Register(&RouteDefinition{
		Path:      "/a",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})
	assert.Equal(t, 1, rr.Count())

	_ = rr.Register(&RouteDefinition{
		Path:      "/b/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})
	assert.Equal(t, 2, rr.Count())

	_ = rr.Register(&RouteDefinition{
		Path:      "/c/:ver/x",
		MatchType: RouteMatchPrefix,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})
	assert.Equal(t, 3, rr.Count())
}

func TestRouteRegistry_ComputeAccess(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/users",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/users/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpCreate},
		},
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/invoices",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceInvoice, Operation: OpRead},
		},
	})

	t.Run("user with read permission", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceUser): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.True(t, access["/users"])
		assert.False(t, access["/users/new"])
		assert.False(t, access["/invoices"])
	})

	t.Run("user with read and create", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceUser): ClientOpRead | ClientOpCreate,
		}

		access := rr.ComputeAccess(permissions)
		assert.True(t, access["/users"])
		assert.True(t, access["/users/new"])
		assert.False(t, access["/invoices"])
	})

	t.Run("user with all permissions", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceUser):    ClientOpRead | ClientOpCreate,
			string(ResourceInvoice): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.True(t, access["/users"])
		assert.True(t, access["/users/new"])
		assert.True(t, access["/invoices"])
	})

	t.Run("empty permissions", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{}

		access := rr.ComputeAccess(permissions)
		assert.False(t, access["/users"])
		assert.False(t, access["/users/new"])
		assert.False(t, access["/invoices"])
	})
}

func TestRouteRegistry_ComputeAccess_WithOrCombinator(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/dashboard",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceReport, Operation: OpRead, Combinator: "or"},
			{Resource: ResourceDashboard, Operation: OpRead, Combinator: "or"},
		},
	})

	t.Run("has first resource", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceReport): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.True(t, access["/dashboard"])
	})

	t.Run("has second resource", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceDashboard): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.True(t, access["/dashboard"])
	})

	t.Run("has neither resource", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceUser): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.False(t, access["/dashboard"])
	})
}

func TestRouteRegistry_ComputeAccess_WithAndCombinator(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/admin/special",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
			{Resource: ResourceRole, Operation: OpRead},
		},
	})

	t.Run("has both permissions", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceUser): ClientOpRead,
			string(ResourceRole): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.True(t, access["/admin/special"])
	})

	t.Run("has only first", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceUser): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.False(t, access["/admin/special"])
	})

	t.Run("has only second", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceRole): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.False(t, access["/admin/special"])
	})
}

func TestRouteRegistry_ComputeAccess_PatternAndPrefixRoutes(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/items/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceCommodity, Operation: OpRead},
		},
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/api/:ver/items",
		MatchType: RouteMatchPrefix,
		Requirements: []RouteRequirement{
			{Resource: ResourceCommodity, Operation: OpRead},
		},
	})

	t.Run("has permission", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{
			string(ResourceCommodity): ClientOpRead,
		}

		access := rr.ComputeAccess(permissions)
		assert.True(t, access["/items/:id"])
		assert.True(t, access["/api/:ver/items"])
	})

	t.Run("no permission", func(t *testing.T) {
		t.Parallel()

		permissions := map[string]uint32{}
		access := rr.ComputeAccess(permissions)
		assert.False(t, access["/items/:id"])
		assert.False(t, access["/api/:ver/items"])
	})
}

func TestRouteRegistry_ComputeAccess_EmptyRequirements(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	rr.mu.Lock()
	rr.routes["/open"] = &RouteDefinition{
		Path:         "/open",
		MatchType:    RouteMatchExact,
		Requirements: []RouteRequirement{},
	}
	rr.mu.Unlock()

	permissions := map[string]uint32{}
	access := rr.ComputeAccess(permissions)
	assert.True(t, access["/open"])
}

func TestRouteRegistry_RegisterAll_RoutesExist(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()

	expectedRoutes := []string{
		"/admin/users",
		"/admin/roles",
		"/equipment/tractors",
		"/equipment/trailers",
		"/workers",
		"/shipments",
		"/billing/invoices",
		"/customers",
		"/locations",
		"/commodities",
		"/accounting/gl-accounts",
		"/reports",
		"/dashboard",
	}

	for _, path := range expectedRoutes {
		def, ok := rr.Match(path)
		assert.True(t, ok, "expected route %s to be registered", path)
		if ok {
			assert.NotEmpty(t, def.DisplayName, "route %s should have a display name", path)
			assert.NotEmpty(t, def.Category, "route %s should have a category", path)
		}
	}
}

func TestRouteRegistry_PatternMatchWithParams(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()

	tests := []struct {
		name  string
		path  string
		found bool
	}{
		{"user detail with uuid", "/admin/users/550e8400-e29b-41d4-a716-446655440000", true},
		{"user edit", "/admin/users/abc123/edit", true},
		{"tractor detail", "/equipment/tractors/xyz789", true},
		{"shipment detail", "/shipments/ship-001", true},
		{"unknown nested path", "/unknown/abc/def", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, ok := rr.Match(tt.path)
			assert.Equal(t, tt.found, ok)
		})
	}
}

func TestRouteRegistry_Match_PrioritizesExactOverPattern(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/users/new",
		MatchType: RouteMatchExact,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpCreate},
		},
		DisplayName: "Create User",
	})

	_ = rr.Register(&RouteDefinition{
		Path:      "/users/:id",
		MatchType: RouteMatchPattern,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		DisplayName: "User Details",
	})

	def, ok := rr.Match("/users/new")
	require.True(t, ok)
	assert.Equal(t, "Create User", def.DisplayName)
}

func TestRouteRegistry_GetByCategory_PrefixRoutes(t *testing.T) {
	t.Parallel()

	rr := NewEmptyRouteRegistry()

	_ = rr.Register(&RouteDefinition{
		Path:      "/api/:ver/admin",
		MatchType: RouteMatchPrefix,
		Requirements: []RouteRequirement{
			{Resource: ResourceUser, Operation: OpRead},
		},
		Category: "Admin",
	})

	routes := rr.GetByCategory("Admin")
	assert.Len(t, routes, 1)
}

func TestRouteRegistry_HasHazmatSegregationRuleRoute(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()

	def, ok := rr.Get("/organization/hazmat-segregation-rules")
	require.True(t, ok)
	assert.Equal(t, "Hazmat Segregation Rules", def.DisplayName)
	require.Len(t, def.Requirements, 1)
	assert.Equal(t, ResourceHazmatSegregationRule, def.Requirements[0].Resource)
	assert.Equal(t, OpRead, def.Requirements[0].Operation)
}

func TestRouteRegistry_HasShipmentControlsRoute(t *testing.T) {
	t.Parallel()

	rr := NewRouteRegistry()

	def, ok := rr.Get("/admin/shipment-controls")
	require.True(t, ok)
	assert.Equal(t, "Shipment Controls", def.DisplayName)
	require.Len(t, def.Requirements, 1)
	assert.Equal(t, ResourceShipmentControl, def.Requirements[0].Resource)
	assert.Equal(t, OpRead, def.Requirements[0].Operation)
}
