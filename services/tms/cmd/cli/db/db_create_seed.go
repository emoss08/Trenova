package db

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/tools/imports"
)

var (
	seedDev  bool
	seedTest bool
)

var createSeedCmd = &cobra.Command{
	Use:   "create-seed <name>",
	Short: "Create a new database seed",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreateSeed,
}

func init() {
	createSeedCmd.Flags().BoolVar(&seedDev, "dev", false, "Create seed in development directory")
	createSeedCmd.Flags().BoolVar(&seedTest, "test", false, "Create seed in test directory")
}

func runCreateSeed(cmd *cobra.Command, args []string) error {
	seedName := args[0]

	var targetDir string
	var environments string

	if seedDev {
		targetDir = "./internal/infrastructure/database/seeds/development"
		environments = "common.EnvDevelopment"
	} else if seedTest {
		targetDir = "./internal/infrastructure/database/seeds/test"
		environments = "common.EnvTest"
	} else {
		targetDir = "./internal/infrastructure/database/seeds/base"
		environments = "common.EnvProduction, common.EnvStaging, common.EnvDevelopment, common.EnvTest"
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	nextNum, err := getNextSeedNumber(targetDir)
	if err != nil {
		return fmt.Errorf("get next seed number: %w", err)
	}

	filename := fmt.Sprintf("%02d_%s.go", nextNum, strings.ToLower(seedName))
	filepath := filepath.Join(targetDir, filename)

	if _, err := os.Stat(filepath); err == nil {
		return fmt.Errorf("seed file already exists: %s", filepath)
	}

	content, err := generateSeedContent(seedName, environments, seedDev || seedTest)
	if err != nil {
		return fmt.Errorf("generate seed content: %w", err)
	}

	formatted, err := format.Source([]byte(content))
	if err != nil {
		formatted, err = imports.Process(filepath, []byte(content), nil)
		if err != nil {
			color.Yellow("⚠ Could not format seed file: %v", err)
			formatted = []byte(content)
		}
	} else {
		formatted, _ = imports.Process(filepath, formatted, nil)
	}

	if err := os.WriteFile(filepath, formatted, 0o644); err != nil {
		return fmt.Errorf("write seed file: %w", err)
	}

	if err := applyGolines(filepath); err != nil {
		color.Yellow("⚠ golines warning (non-fatal): %v", err)
	}

	color.Green("✓ Created seed: %s", filepath)

	color.Cyan("→ Updating seed registry...")
	if err := regenerateRegistry(); err != nil {
		color.Yellow("⚠ Failed to update registry: %v", err)
		fmt.Println("\nManually regenerate with: trenova db seed-sync")
	} else {
		color.Green("✓ Registry updated")
	}

	fmt.Println("\nNext steps:")
	fmt.Println("1. Edit the seed file to add your data")
	fmt.Println("2. Run 'trenova db seed' to apply the seed")

	return nil
}

func getNextSeedNumber(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	maxNum := -1
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		parts := strings.Split(entry.Name(), "_")
		if len(parts) < 2 {
			continue
		}

		if num, err := strconv.Atoi(parts[0]); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}

	return maxNum + 1, nil
}

func generateSeedContent(name string, environments string, isDev bool) (string, error) {
	structName := lo.PascalCase(name)

	packageName := "base"
	if isDev {
		packageName = "development"
	} else if seedTest {
		packageName = "test"
	}

	tmplContent := `package {{ .Package }}

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type {{ .StructName }}Seed struct {
	seedhelpers.BaseSeed
}

func New{{ .StructName }}Seed() *{{ .StructName }}Seed {
	seed := &{{ .StructName }}Seed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"{{ .StructName }}",
		"1.0.0",
		"{{ .Description }}",
		[]common.Environment{
			{{ .Environments }},
		},
	)
	{{ if .IsDev }}seed.SetDependencies(seedhelpers.SeedAdminAccount)
	{{ end }}return seed
}

{{ if .IsDev }}func (s *{{ .StructName }}Seed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			_ = org

			return nil
		},
	)
}
{{ else }}func (s *{{ .StructName }}Seed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return nil
		},
	)
}
{{ end }}
func (s *{{ .StructName }}Seed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *{{ .StructName }}Seed) CanRollback() bool {
	return true
}
`

	tmpl, err := template.New("seed").Parse(tmplContent)
	if err != nil {
		return "", err
	}

	data := struct {
		Package      string
		StructName   string
		Name         string
		Description  string
		Environments string
		IsDev        bool
	}{
		Package:      packageName,
		StructName:   structName,
		Name:         name,
		Description:  fmt.Sprintf("Creates %s data", strings.ReplaceAll(name, "_", " ")),
		Environments: environments,
		IsDev:        isDev,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func applyGolines(filepath string) error {
	if _, err := exec.LookPath("golines"); err != nil {
		return nil
	}

	golinesCmd := exec.Command("golines", "-w", "-m", "120", "--base-formatter", "gofmt", filepath)
	var golinesStderr bytes.Buffer
	golinesCmd.Stderr = &golinesStderr

	if err := golinesCmd.Run(); err != nil {
		return fmt.Errorf("golines: %s", golinesStderr.String())
	}

	return nil
}

func regenerateRegistry() error {
	cmd := exec.Command("go", "generate", "./internal/infrastructure/database/seeder/...")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go generate failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}
