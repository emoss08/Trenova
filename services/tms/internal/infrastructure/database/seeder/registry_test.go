package seeder

import (
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	seedermocks "github.com/emoss08/trenova/shared/testutil/seeder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_NewRegistry(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	assert.NotNil(t, r)
	assert.NotNil(t, r.seeds)
	assert.NotNil(t, r.envSeeds)
	assert.Equal(t, 0, r.Size())
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		seeds     []*seedermocks.MockSeed
		wantErr   bool
		wantSize  int
		errString string
	}{
		{
			name: "single seed",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed("SeedA"),
			},
			wantErr:  false,
			wantSize: 1,
		},
		{
			name: "multiple seeds",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed("SeedA"),
				seedermocks.NewMockSeed("SeedB"),
				seedermocks.NewMockSeed("SeedC"),
			},
			wantErr:  false,
			wantSize: 3,
		},
		{
			name: "duplicate seed",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed("SeedA"),
				seedermocks.NewMockSeed("SeedA"),
			},
			wantErr:   true,
			wantSize:  1,
			errString: "seed already registered: SeedA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewRegistry()
			var lastErr error

			for _, seed := range tt.seeds {
				if err := r.Register(seed); err != nil {
					lastErr = err
				}
			}

			assert.Equal(t, tt.wantSize, r.Size())

			if tt.wantErr {
				require.Error(t, lastErr)
				assert.Contains(t, lastErr.Error(), tt.errString)
				assert.ErrorIs(t, lastErr, ErrSeedAlreadyRegistered)
			} else {
				assert.NoError(t, lastErr)
			}
		})
	}
}

func TestRegistry_MustRegister(t *testing.T) {
	t.Parallel()

	t.Run("valid registration", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		seed := seedermocks.NewMockSeed("TestSeed")

		assert.NotPanics(t, func() {
			r.MustRegister(seed)
		})
		assert.Equal(t, 1, r.Size())
	})

	t.Run("panic on duplicate", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		seed1 := seedermocks.NewMockSeed("TestSeed")
		seed2 := seedermocks.NewMockSeed("TestSeed")

		r.MustRegister(seed1)

		assert.Panics(t, func() {
			r.MustRegister(seed2)
		})
	})
}

func TestRegistry_RegisterForEnv(t *testing.T) {
	t.Parallel()

	t.Run("valid registration", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		seed := seedermocks.NewMockSeed("DevSeed",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		)

		err := r.RegisterForEnv(common.EnvDevelopment, seed)
		require.NoError(t, err)
		assert.Equal(t, 1, r.Size())
	})

	t.Run("environment mismatch", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		seed := seedermocks.NewMockSeed("DevSeed",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		)

		err := r.RegisterForEnv(common.EnvProduction, seed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support environment")
		assert.Equal(t, 0, r.Size())
	})

	t.Run("multiple seeds", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		seed1 := seedermocks.NewMockSeed("Seed1",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		)
		seed2 := seedermocks.NewMockSeed("Seed2",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		)

		err := r.RegisterForEnv(common.EnvDevelopment, seed1, seed2)
		require.NoError(t, err)
		assert.Equal(t, 2, r.Size())
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	seed := seedermocks.NewMockSeed("TestSeed",
		seedermocks.WithVersion("2.0.0"),
	)
	r.MustRegister(seed)

	t.Run("existing seed", func(t *testing.T) {
		found, exists := r.Get("TestSeed")
		assert.True(t, exists)
		assert.Equal(t, "TestSeed", found.Name())
		assert.Equal(t, "2.0.0", found.Version())
	})

	t.Run("non-existing seed", func(t *testing.T) {
		_, exists := r.Get("NonExistent")
		assert.False(t, exists)
	})
}

func TestRegistry_GetForEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		seeds     []*seedermocks.MockSeed
		env       common.Environment
		wantNames []string
	}{
		{
			name: "single environment match",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed(
					"DevSeed",
					seedermocks.WithEnvironments(common.EnvDevelopment),
				),
				seedermocks.NewMockSeed(
					"ProdSeed",
					seedermocks.WithEnvironments(common.EnvProduction),
				),
			},
			env:       common.EnvDevelopment,
			wantNames: []string{"DevSeed"},
		},
		{
			name: "multiple environment match",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed(
					"AllEnvSeed",
					seedermocks.WithEnvironments(common.EnvDevelopment, common.EnvProduction),
				),
				seedermocks.NewMockSeed(
					"DevOnlySeed",
					seedermocks.WithEnvironments(common.EnvDevelopment),
				),
			},
			env:       common.EnvDevelopment,
			wantNames: []string{"AllEnvSeed", "DevOnlySeed"},
		},
		{
			name: "no match",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed(
					"DevSeed",
					seedermocks.WithEnvironments(common.EnvDevelopment),
				),
			},
			env:       common.EnvProduction,
			wantNames: nil,
		},
		{
			name:      "empty registry",
			seeds:     nil,
			env:       common.EnvDevelopment,
			wantNames: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewRegistry()
			for _, seed := range tt.seeds {
				r.MustRegister(seed)
			}

			result := r.GetForEnvironment(tt.env)

			if tt.wantNames == nil {
				assert.Empty(t, result)
			} else {
				var names []string
				for _, s := range result {
					names = append(names, s.Name())
				}
				assert.ElementsMatch(t, tt.wantNames, names)
			}
		})
	}
}

func TestRegistry_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		seeds            []*seedermocks.MockSeed
		wantErr          bool
		wantMissingDeps  bool
		wantCircularDeps bool
	}{
		{
			name: "valid graph",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B"),
			},
			wantErr: false,
		},
		{
			name: "missing dependency",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B", "Missing")),
				seedermocks.NewMockSeed("B"),
			},
			wantErr:         true,
			wantMissingDeps: true,
		},
		{
			name: "circular dependency",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed("A", seedermocks.WithDependencies("B")),
				seedermocks.NewMockSeed("B", seedermocks.WithDependencies("A")),
			},
			wantErr:          true,
			wantCircularDeps: true,
		},
		{
			name:    "empty registry is valid",
			seeds:   nil,
			wantErr: false,
		},
		{
			name: "complex valid graph",
			seeds: []*seedermocks.MockSeed{
				seedermocks.NewMockSeed("App", seedermocks.WithDependencies("Users", "Config")),
				seedermocks.NewMockSeed("Users", seedermocks.WithDependencies("Organizations")),
				seedermocks.NewMockSeed("Config"),
				seedermocks.NewMockSeed("Organizations"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewRegistry()
			for _, seed := range tt.seeds {
				r.MustRegister(seed)
			}

			err := r.Validate()

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

func TestRegistry_GetExecutionOrder(t *testing.T) {
	t.Parallel()

	t.Run("full order", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.MustRegister(seedermocks.NewMockSeed("App",
			seedermocks.WithDependencies("Users", "Config"),
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("Users",
			seedermocks.WithDependencies("Organizations"),
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("Config",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("Organizations",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))

		order, err := r.GetExecutionOrder(common.EnvDevelopment, "")
		require.NoError(t, err)
		require.Len(t, order, 4)

		indexOf := func(name string) int {
			for i, s := range order {
				if s.Name() == name {
					return i
				}
			}
			return -1
		}

		assert.Less(t, indexOf("Organizations"), indexOf("Users"))
		assert.Less(t, indexOf("Users"), indexOf("App"))
		assert.Less(t, indexOf("Config"), indexOf("App"))
	})

	t.Run("with target", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.MustRegister(seedermocks.NewMockSeed("App",
			seedermocks.WithDependencies("Users"),
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("Users",
			seedermocks.WithDependencies("Organizations"),
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("Organizations",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("Unrelated",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))

		order, err := r.GetExecutionOrder(common.EnvDevelopment, "Users")
		require.NoError(t, err)
		require.Len(t, order, 2)

		var names []string
		for _, s := range order {
			names = append(names, s.Name())
		}
		assert.ElementsMatch(t, []string{"Organizations", "Users"}, names)
	})

	t.Run("empty registry", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		order, err := r.GetExecutionOrder(common.EnvDevelopment, "")
		require.NoError(t, err)
		assert.Empty(t, order)
	})

	t.Run("target not found", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.MustRegister(seedermocks.NewMockSeed("Seed1",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))

		_, err := r.GetExecutionOrder(common.EnvDevelopment, "NonExistent")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSeedNotFound)
	})

	t.Run("filters by environment", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.MustRegister(seedermocks.NewMockSeed("DevSeed",
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("ProdSeed",
			seedermocks.WithEnvironments(common.EnvProduction),
		))

		order, err := r.GetExecutionOrder(common.EnvDevelopment, "")
		require.NoError(t, err)
		require.Len(t, order, 1)
		assert.Equal(t, "DevSeed", order[0].Name())
	})

	t.Run("circular dependency error", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.MustRegister(seedermocks.NewMockSeed("A",
			seedermocks.WithDependencies("B"),
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))
		r.MustRegister(seedermocks.NewMockSeed("B",
			seedermocks.WithDependencies("A"),
			seedermocks.WithEnvironments(common.EnvDevelopment),
		))

		_, err := r.GetExecutionOrder(common.EnvDevelopment, "")
		require.Error(t, err)

		var depErr *DependencyError
		assert.ErrorAs(t, err, &depErr)
	})
}

func TestRegistry_All(t *testing.T) {
	t.Parallel()

	t.Run("empty registry", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		all := r.All()
		assert.Empty(t, all)
	})

	t.Run("with seeds", func(t *testing.T) {
		t.Parallel()

		r := NewRegistry()
		r.MustRegister(seedermocks.NewMockSeed("A"))
		r.MustRegister(seedermocks.NewMockSeed("B"))
		r.MustRegister(seedermocks.NewMockSeed("C"))

		all := r.All()
		assert.Len(t, all, 3)

		var names []string
		for _, s := range all {
			names = append(names, s.Name())
		}
		assert.ElementsMatch(t, []string{"A", "B", "C"}, names)
	})
}

func TestRegistry_Size(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	assert.Equal(t, 0, r.Size())

	r.MustRegister(seedermocks.NewMockSeed("A"))
	assert.Equal(t, 1, r.Size())

	r.MustRegister(seedermocks.NewMockSeed("B"))
	r.MustRegister(seedermocks.NewMockSeed("C"))
	assert.Equal(t, 3, r.Size())
}

func TestRegistry_Concurrent(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	var wg sync.WaitGroup
	const numGoroutines = 100

	for i := range numGoroutines {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			seed := seedermocks.NewMockSeed(fmt.Sprintf("Seed%d", n))
			_ = r.Register(seed)
		}(i)
	}

	wg.Wait()
	assert.Equal(t, numGoroutines, r.Size())

	for i := range numGoroutines {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = r.Get(fmt.Sprintf("Seed%d", n))
		}(i)
	}
	wg.Wait()
}

func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	var wg sync.WaitGroup

	for i := range 50 {
		r.MustRegister(seedermocks.NewMockSeed(fmt.Sprintf("Initial%d", i)))
	}

	for i := range 100 {
		wg.Add(3)

		go func(n int) {
			defer wg.Done()
			seed := seedermocks.NewMockSeed(fmt.Sprintf("New%d", n))
			_ = r.Register(seed)
		}(i)

		go func() {
			defer wg.Done()
			_ = r.All()
		}()

		go func() {
			defer wg.Done()
			_ = r.GetForEnvironment(common.EnvDevelopment)
		}()
	}

	wg.Wait()
	assert.GreaterOrEqual(t, r.Size(), 50)
}
