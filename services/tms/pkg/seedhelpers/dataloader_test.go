package seedhelpers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataLoader_NewDataLoader(t *testing.T) {
	t.Parallel()

	loader := seedhelpers.NewDataLoader("/test/path")
	require.NotNil(t, loader)
	assert.Equal(t, "/test/path", loader.BasePath())
}

func TestDataLoader_LoadYAML(t *testing.T) {
	t.Parallel()

	t.Run("successfully loads valid YAML", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		yamlContent := `
name: Test
value: 123
items:
  - first
  - second
`
		yamlFile := filepath.Join(tempDir, "test.yaml")
		require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct {
			Name  string   `json:"name"`
			Value int      `json:"value"`
			Items []string `json:"items"`
		}

		err := loader.LoadYAML("test.yaml", &result)
		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, 123, result.Value)
		assert.Equal(t, []string{"first", "second"}, result.Items)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct{}
		err := loader.LoadYAML("missing.yaml", &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read file")
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		invalidYAML := `
name: Test
value: [invalid yaml structure
`
		yamlFile := filepath.Join(tempDir, "invalid.yaml")
		require.NoError(t, os.WriteFile(yamlFile, []byte(invalidYAML), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct{}
		err := loader.LoadYAML("invalid.yaml", &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal YAML")
	})

	t.Run("loads nested structures", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		yamlContent := `
organization:
  name: Acme Corp
  address:
    street: 123 Main St
    city: Springfield
    state: CA
`
		yamlFile := filepath.Join(tempDir, "nested.yaml")
		require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct {
			Organization struct {
				Name    string `json:"name"`
				Address struct {
					Street string `json:"street"`
					City   string `json:"city"`
					State  string `json:"state"`
				} `json:"address"`
			} `json:"organization"`
		}

		err := loader.LoadYAML("nested.yaml", &result)
		require.NoError(t, err)
		assert.Equal(t, "Acme Corp", result.Organization.Name)
		assert.Equal(t, "123 Main St", result.Organization.Address.Street)
		assert.Equal(t, "Springfield", result.Organization.Address.City)
		assert.Equal(t, "CA", result.Organization.Address.State)
	})

	t.Run("loads array of objects", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		yamlContent := `
users:
  - name: Alice
    age: 30
  - name: Bob
    age: 25
`
		yamlFile := filepath.Join(tempDir, "array.yaml")
		require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct {
			Users []struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			} `json:"users"`
		}

		err := loader.LoadYAML("array.yaml", &result)
		require.NoError(t, err)
		require.Len(t, result.Users, 2)
		assert.Equal(t, "Alice", result.Users[0].Name)
		assert.Equal(t, 30, result.Users[0].Age)
		assert.Equal(t, "Bob", result.Users[1].Name)
		assert.Equal(t, 25, result.Users[1].Age)
	})

	t.Run("uses json struct tags for YAML files", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		yamlContent := `
workers:
  - firstName: John
    lastName: Smith
    addressLine1: 123 Main St
    city: Los Angeles
    postalCode: "90001"
    status: Active
    type: Employee
`
		yamlFile := filepath.Join(tempDir, "workers.yaml")
		require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		type WorkerSeed struct {
			FirstName    string `json:"firstName"`
			LastName     string `json:"lastName"`
			AddressLine1 string `json:"addressLine1"`
			City         string `json:"city"`
			PostalCode   string `json:"postalCode"`
			Status       string `json:"status"`
			Type         string `json:"type"`
		}

		var result struct {
			Workers []WorkerSeed `json:"workers"`
		}

		err := loader.LoadYAML("workers.yaml", &result)
		require.NoError(t, err)
		require.Len(t, result.Workers, 1)
		assert.Equal(t, "John", result.Workers[0].FirstName)
		assert.Equal(t, "Smith", result.Workers[0].LastName)
		assert.Equal(t, "123 Main St", result.Workers[0].AddressLine1)
		assert.Equal(t, "Los Angeles", result.Workers[0].City)
		assert.Equal(t, "90001", result.Workers[0].PostalCode)
		assert.Equal(t, "Active", result.Workers[0].Status)
		assert.Equal(t, "Employee", result.Workers[0].Type)
	})
}

func TestDataLoader_LoadJSON(t *testing.T) {
	t.Parallel()

	t.Run("successfully loads valid JSON", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		jsonContent := `{
  "name": "Test",
  "value": 123,
  "items": ["first", "second"]
}`
		jsonFile := filepath.Join(tempDir, "test.json")
		require.NoError(t, os.WriteFile(jsonFile, []byte(jsonContent), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct {
			Name  string   `json:"name"`
			Value int      `json:"value"`
			Items []string `json:"items"`
		}

		err := loader.LoadJSON("test.json", &result)
		require.NoError(t, err)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, 123, result.Value)
		assert.Equal(t, []string{"first", "second"}, result.Items)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct{}
		err := loader.LoadJSON("missing.json", &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read file")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		invalidJSON := `{
  "name": "Test",
  "value": invalid json
}`
		jsonFile := filepath.Join(tempDir, "invalid.json")
		require.NoError(t, os.WriteFile(jsonFile, []byte(invalidJSON), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct{}
		err := loader.LoadJSON("invalid.json", &result)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal JSON")
	})

	t.Run("loads nested JSON structures", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		jsonContent := `{
  "organization": {
    "name": "Acme Corp",
    "address": {
      "street": "123 Main St",
      "city": "Springfield",
      "state": "CA"
    }
  }
}`
		jsonFile := filepath.Join(tempDir, "nested.json")
		require.NoError(t, os.WriteFile(jsonFile, []byte(jsonContent), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct {
			Organization struct {
				Name    string `json:"name"`
				Address struct {
					Street string `json:"street"`
					City   string `json:"city"`
					State  string `json:"state"`
				} `json:"address"`
			} `json:"organization"`
		}

		err := loader.LoadJSON("nested.json", &result)
		require.NoError(t, err)
		assert.Equal(t, "Acme Corp", result.Organization.Name)
		assert.Equal(t, "123 Main St", result.Organization.Address.Street)
		assert.Equal(t, "Springfield", result.Organization.Address.City)
		assert.Equal(t, "CA", result.Organization.Address.State)
	})
}

func TestDataLoader_FileExists(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "exists.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)
		exists := loader.FileExists("exists.txt")
		assert.True(t, exists)
	})

	t.Run("returns false for non-existent file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		loader := seedhelpers.NewDataLoader(tempDir)
		exists := loader.FileExists("missing.txt")
		assert.False(t, exists)
	})

	t.Run("returns true for existing subdirectory file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))
		testFile := filepath.Join(subDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)
		exists := loader.FileExists(filepath.Join("subdir", "test.txt"))
		assert.True(t, exists)
	})
}

func TestDataLoader_BasePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "absolute path", path: "/test/path"},
		{name: "relative path", path: "./data"},
		{name: "nested path", path: "/var/lib/seeds/data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := seedhelpers.NewDataLoader(tt.path)
			assert.Equal(t, tt.path, loader.BasePath())
		})
	}
}

func TestDataLoader_Integration(t *testing.T) {
	t.Parallel()

	t.Run("loads multiple files in sequence", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		yamlContent := `name: YAML Data`
		jsonContent := `{"name": "JSON Data"}`

		require.NoError(
			t,
			os.WriteFile(filepath.Join(tempDir, "data.yaml"), []byte(yamlContent), 0644),
		)
		require.NoError(
			t,
			os.WriteFile(filepath.Join(tempDir, "data.json"), []byte(jsonContent), 0644),
		)

		loader := seedhelpers.NewDataLoader(tempDir)

		var yamlResult struct {
			Name string `json:"name"`
		}
		var jsonResult struct {
			Name string `json:"name"`
		}

		err := loader.LoadYAML("data.yaml", &yamlResult)
		require.NoError(t, err)
		assert.Equal(t, "YAML Data", yamlResult.Name)

		err = loader.LoadJSON("data.json", &jsonResult)
		require.NoError(t, err)
		assert.Equal(t, "JSON Data", jsonResult.Name)
	})

	t.Run("handles empty files gracefully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		emptyFile := filepath.Join(tempDir, "empty.yaml")
		require.NoError(t, os.WriteFile(emptyFile, []byte(""), 0644))

		loader := seedhelpers.NewDataLoader(tempDir)

		var result struct {
			Name string `json:"name"`
		}

		err := loader.LoadYAML("empty.yaml", &result)
		assert.NoError(t, err)
	})
}
