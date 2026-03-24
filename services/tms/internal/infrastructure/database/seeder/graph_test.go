package seeder

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	seedermocks "github.com/emoss08/trenova/shared/testutil/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraph_NewGraph(t *testing.T) {
	t.Parallel()

	g := NewGraph()

	assert.NotNil(t, g)
	assert.NotNil(t, g.nodes)
	assert.NotNil(t, g.edges)
	assert.Equal(t, 0, g.Size())
}

func TestGraph_AddNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		seedNames []string
		wantSize  int
	}{
		{
			name:      "single node",
			seedNames: []string{"A"},
			wantSize:  1,
		},
		{
			name:      "multiple nodes",
			seedNames: []string{"A", "B", "C"},
			wantSize:  3,
		},
		{
			name:      "duplicate nodes",
			seedNames: []string{"A", "A", "B"},
			wantSize:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGraph()
			for _, name := range tt.seedNames {
				seed := seedermocks.NewMockSeed(name)
				g.AddNode(seed)
			}

			assert.Equal(t, tt.wantSize, g.Size())
		})
	}
}

func TestGraph_AddEdge(t *testing.T) {
	t.Parallel()

	g := NewGraph()
	seedA := seedermocks.NewMockSeed("A")
	seedB := seedermocks.NewMockSeed("B")

	g.AddNode(seedA)
	g.AddNode(seedB)
	g.AddEdge("A", "B")

	assert.Contains(t, g.edges["A"], "B")
	assert.Empty(t, g.edges["B"])
}

func TestGraph_BuildFromSeeds(t *testing.T) {
	t.Parallel()

	seeds := []Seed{
		seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B", "C")),
		seedermocks.NewMockSeed("B", seedermocks.WithDependencies("C")),
		seedermocks.NewMockSeed("C"),
	}

	g := NewGraph()
	g.BuildFromSeeds(seeds)

	assert.Equal(t, 3, g.Size())
	assert.ElementsMatch(t, []string{"B", "C"}, g.edges["A"])
	assert.ElementsMatch(t, []string{"C"}, g.edges["B"])
	assert.Empty(t, g.edges["C"])
}

func TestGraph_DetectCycle_NoCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		seeds []Seed
	}{
		{
			name: "linear chain",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("C")),
				seedermocks.NewMockSeed("C"),
			},
		},
		{
			name: "diamond pattern",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B", "C")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("D")),
				seedermocks.NewMockSeed("C", seedermocks.WithDependencies("D")),
				seedermocks.NewMockSeed("D"),
			},
		},
		{
			name: "disconnected components",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B"),
				seedermocks.NewMockSeed("C", seedermocks.WithDependencies("D")),
				seedermocks.NewMockSeed("D"),
			},
		},
		{
			name: "single node",
			seeds: []Seed{
				seedermocks.NewMockSeed("A"),
			},
		},
		{
			name:  "empty graph",
			seeds: []Seed{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGraph()
			g.BuildFromSeeds(tt.seeds)

			cycle := g.DetectCycle()
			assert.Nil(t, cycle, "expected no cycle")
		})
	}
}

func TestGraph_DetectCycle_WithCycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		seeds        []Seed
		wantCycleLen int
		wantCycleHas []string
	}{
		{
			name: "simple A->B->A cycle",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("A")),
			},
			wantCycleLen: 3,
			wantCycleHas: []string{"A", "B"},
		},
		{
			name: "three node cycle A->B->C->A",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("C")),
				seedermocks.NewMockSeed("C", seedermocks.WithDependencies("A")),
			},
			wantCycleLen: 4,
			wantCycleHas: []string{"A", "B", "C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGraph()
			g.BuildFromSeeds(tt.seeds)

			cycle := g.DetectCycle()
			require.NotNil(t, cycle, "expected cycle")
			assert.GreaterOrEqual(t, len(cycle), 2, "cycle should have at least 2 nodes")

			for _, expected := range tt.wantCycleHas {
				assert.Contains(t, cycle, expected)
			}
		})
	}
}

func TestGraph_TopologicalSort_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		seeds      []Seed
		wantFirst  string
		wantLast   string
		wantLength int
	}{
		{
			name: "linear chain",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("C")),
				seedermocks.NewMockSeed("C"),
			},
			wantFirst:  "C",
			wantLast:   "A",
			wantLength: 3,
		},
		{
			name: "single node",
			seeds: []Seed{
				seedermocks.NewMockSeed("A"),
			},
			wantFirst:  "A",
			wantLast:   "A",
			wantLength: 1,
		},
		{
			name:       "empty graph",
			seeds:      []Seed{},
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGraph()
			g.BuildFromSeeds(tt.seeds)

			result, err := g.TopologicalSort()
			require.NoError(t, err)
			assert.Len(t, result, tt.wantLength)

			if tt.wantLength > 0 {
				assert.Equal(
					t,
					tt.wantFirst,
					result[0],
					"first element should be dependency with no deps",
				)
				assert.Equal(
					t,
					tt.wantLast,
					result[len(result)-1],
					"last element should be dependent",
				)
			}
		})
	}
}

func TestGraph_TopologicalSort_DependencyOrder(t *testing.T) {
	t.Parallel()

	seeds := []Seed{
		seedermocks.NewMockSeed("App", seedermocks.WithDependencies("Users", "Config")),
		seedermocks.NewMockSeed("Users", seedermocks.WithDependencies("Organizations")),
		seedermocks.NewMockSeed("Config"),
		seedermocks.NewMockSeed("Organizations"),
	}

	g := NewGraph()
	g.BuildFromSeeds(seeds)

	result, err := g.TopologicalSort()
	require.NoError(t, err)
	require.Len(t, result, 4)

	indexOf := func(name string) int {
		for i, n := range result {
			if n == name {
				return i
			}
		}
		return -1
	}

	assert.Less(
		t,
		indexOf("Organizations"),
		indexOf("Users"),
		"Organizations must come before Users",
	)
	assert.Less(t, indexOf("Users"), indexOf("App"), "Users must come before App")
	assert.Less(t, indexOf("Config"), indexOf("App"), "Config must come before App")
}

func TestGraph_TopologicalSort_Cycle(t *testing.T) {
	t.Parallel()

	seeds := []Seed{
		seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
		seedermocks.NewMockSeed("B", seedermocks.WithDependencies("A")),
	}

	g := NewGraph()
	g.BuildFromSeeds(seeds)

	_, err := g.TopologicalSort()
	require.Error(t, err)

	var depErr *DependencyError
	assert.ErrorAs(t, err, &depErr)
}

func TestGraph_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		seeds            []Seed
		wantErr          bool
		wantMissingDeps  bool
		wantCircularDeps bool
	}{
		{
			name: "valid graph",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B"),
			},
			wantErr: false,
		},
		{
			name: "missing dependency",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B", "Missing")),
				seedermocks.NewMockSeed("B"),
			},
			wantErr:         true,
			wantMissingDeps: true,
		},
		{
			name: "circular dependency",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("A")),
			},
			wantErr:          true,
			wantCircularDeps: true,
		},
		{
			name:    "empty graph is valid",
			seeds:   []Seed{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGraph()
			g.BuildFromSeeds(tt.seeds)

			err := g.Validate()

			if tt.wantErr {
				require.Error(t, err)
				var depErr *DependencyError
				if assert.ErrorAs(t, err, &depErr) {
					if tt.wantMissingDeps {
						assert.NotEmpty(t, depErr.MissingDeps)
					}
					if tt.wantCircularDeps {
						assert.NotEmpty(t, depErr.CircularPath)
					}
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGraph_GetDependenciesFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		seeds    []Seed
		target   string
		wantDeps []string
		wantErr  bool
	}{
		{
			name: "direct dependencies",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B", "C")),
				seedermocks.NewMockSeed("B"),
				seedermocks.NewMockSeed("C"),
			},
			target:   "A",
			wantDeps: []string{"B", "C"},
			wantErr:  false,
		},
		{
			name: "transitive dependencies",
			seeds: []Seed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("C")),
				seedermocks.NewMockSeed("C", seedermocks.WithDependencies("D")),
				seedermocks.NewMockSeed("D"),
			},
			target:   "A",
			wantDeps: []string{"B", "C", "D"},
			wantErr:  false,
		},
		{
			name: "no dependencies",
			seeds: []Seed{
				seedermocks.NewMockSeed("A"),
			},
			target:   "A",
			wantDeps: nil,
			wantErr:  false,
		},
		{
			name: "seed not found",
			seeds: []Seed{
				seedermocks.NewMockSeed("A"),
			},
			target:  "NonExistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGraph()
			g.BuildFromSeeds(tt.seeds)

			deps, err := g.GetDependenciesFor(tt.target)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrSeedNotFound)
			} else {
				require.NoError(t, err)
				if tt.wantDeps == nil {
					assert.Empty(t, deps)
				} else {
					assert.ElementsMatch(t, tt.wantDeps, deps)
				}
			}
		})
	}
}

func TestGraph_GetSeed(t *testing.T) {
	t.Parallel()

	seed := seedermocks.NewMockSeed("TestSeed",
		seedermocks.WithVersion("2.0.0"),
		seedermocks.WithEnvironments(common.EnvProduction),
	)

	g := NewGraph()
	g.AddNode(seed)

	t.Run("existing seed", func(t *testing.T) {
		found, exists := g.GetSeed("TestSeed")
		assert.True(t, exists)
		assert.Equal(t, "TestSeed", found.Name())
		assert.Equal(t, "2.0.0", found.Version())
	})

	t.Run("non-existing seed", func(t *testing.T) {
		_, exists := g.GetSeed("NonExistent")
		assert.False(t, exists)
	})
}

func TestGraph_Size(t *testing.T) {
	t.Parallel()

	g := NewGraph()
	assert.Equal(t, 0, g.Size())

	g.AddNode(seedermocks.NewMockSeed("A"))
	assert.Equal(t, 1, g.Size())

	g.AddNode(seedermocks.NewMockSeed("B"))
	g.AddNode(seedermocks.NewMockSeed("C"))
	assert.Equal(t, 3, g.Size())
}
