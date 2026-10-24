// @title Trenova TMS API
// @version 1.0
// @description API documentation for Trenova TMS. Protected routes accept either a Bearer token in the Authorization header or an authenticated session cookie.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/cmd/cli/api"
	"github.com/emoss08/trenova/cmd/cli/db"
	"github.com/emoss08/trenova/cmd/cli/desk"
	"github.com/emoss08/trenova/cmd/cli/encryption"
	"github.com/emoss08/trenova/cmd/cli/redis"
	"github.com/emoss08/trenova/cmd/cli/update"
	"github.com/emoss08/trenova/cmd/cli/worker"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/emoss08/trenova/pkg/domaintypes"

	"github.com/emoss08/trenova/internal/bootstrap/edition"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

func main() {
	rootCmd.AddCommand(edition.Current().Commands...)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "trenova",
	Short: "Trenova CLI - Transportation Management System",
	Long: `Trenova CLI provides administrative tools for managing
the Trenova transportation management system.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		loader := config.NewLoader(append(
			[]config.LoaderOption{config.WithConfigPath("config")},
			edition.Current().LoaderOptions()...,
		)...)

		cf, err := loader.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		cfg = cf
		db.SetConfig(cf)
		desk.SetConfig(cf)
		encryption.SetConfig(cf)
		redis.SetConfig(cf)
		update.SetConfig(cf)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Trenova %s\n", cfg.App.Version)
		fmt.Printf("Edition: %s\n", edition.Current().Name)
		fmt.Printf("Environment: %s\n", cfg.App.Env)
	},
}

var searchVectorCmd = &cobra.Command{
	Use:   "searchvector [domain]",
	Short: "Generate search vector SQL for a domain",
	Long: `Generate SQL statements to create and maintain search vectors for a domain.
This includes creating the search_vector column, GIN index, trigger function, and trigger.

Examples:
  trenova searchvector Location
  trenova searchvector LocationCategory
  trenova searchvector --list
  trenova searchvector --all    # Generate for all domains`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		listFlag, _ := cmd.Flags().GetBool("list")
		allFlag, _ := cmd.Flags().GetBool("all")

		if listFlag {
			return listSearchableDomains()
		}

		if allFlag {
			return generateAllSearchVectorSQL()
		}

		if len(args) == 0 {
			return fmt.Errorf(
				"domain name required (use --list to see available domains, or --all for all)",
			)
		}

		return generateSearchVectorSQL(args[0])
	},
}

func listSearchableDomains() error {
	entities := domainregistry.RegisterEntities()

	fmt.Println("Available searchable domains:")
	fmt.Println(strings.Repeat("=", 40))

	for _, entity := range entities {
		if searchable, ok := entity.(domaintypes.PostgresSearchable); ok {
			config := searchable.GetPostgresSearchConfig()
			if config.UseSearchVector {
				domainName := reflect.TypeOf(entity).Elem().Name()
				tableName := searchable.GetTableName()
				fmt.Printf("  %-20s (table: %s)\n", domainName, tableName)
			}
		}
	}

	return nil
}

func generateSearchVectorSQL(domainName string) error {
	entities := domainregistry.RegisterEntities()

	var found bool
	for _, entity := range entities {
		entityType := reflect.TypeOf(entity).Elem()
		if entityType.Name() == domainName {
			if searchable, ok := entity.(domaintypes.PostgresSearchable); ok {
				config := searchable.GetPostgresSearchConfig()
				if !config.UseSearchVector {
					return fmt.Errorf("domain %s does not use search vectors", domainName)
				}
				generateSQL(searchable, config)
				found = true
				break
			} else {
				return fmt.Errorf("domain %s is not searchable", domainName)
			}
		}
	}

	if !found {
		return fmt.Errorf("domain %s not found in registry", domainName)
	}

	return nil
}

func generateAllSearchVectorSQL() error {
	entities := domainregistry.RegisterEntities()
	generated := 0

	for _, entity := range entities {
		if searchable, ok := entity.(domaintypes.PostgresSearchable); ok {
			config := searchable.GetPostgresSearchConfig()
			if config.UseSearchVector {
				domainName := reflect.TypeOf(entity).Elem().Name()
				fmt.Printf("\n-- ========================================\n")
				fmt.Printf("-- Domain: %s\n", domainName)
				fmt.Printf("-- ========================================\n\n")
				generateSQL(searchable, config)
				generated++
			}
		}
	}

	if generated == 0 {
		return fmt.Errorf("no searchable domains found")
	}

	fmt.Printf("\n-- Generated SQL for %d domains\n", generated)
	return nil
}

func generateSQL(
	searchable domaintypes.PostgresSearchable,
	config domaintypes.PostgresSearchConfig,
) {
	tableName := searchable.GetTableName()
	fmt.Printf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS search_vector tsvector;\n\n", tableName)
	fmt.Printf(
		"CREATE INDEX IF NOT EXISTS idx_%s_search_vector ON \"%s\" USING GIN(search_vector);\n\n",
		tableName,
		tableName,
	)
	var searchFields []string
	for _, field := range config.SearchableFields {
		if field.Type == domaintypes.FieldTypeText {
			weightChar := string(field.Weight)
			if weightChar == "" {
				weightChar = "B"
			}
			searchFields = append(searchFields,
				fmt.Sprintf("setweight(to_tsvector('english', COALESCE(NEW.%s, '')), '%s')",
					field.Name, weightChar))
		}
	}

	if len(searchFields) == 0 {
		fmt.Println("-- Warning: No text fields found for search vector")
		return
	}

	fmt.Printf(`CREATE OR REPLACE FUNCTION %s_search_trigger() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := %s;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

`, tableName, strings.Join(searchFields, " || \n        "))

	fmt.Printf(`DROP TRIGGER IF EXISTS %s_search_update ON "%s";
CREATE TRIGGER %s_search_update
    BEFORE INSERT OR UPDATE ON "%s"
    FOR EACH ROW
    EXECUTE FUNCTION %s_search_trigger();

`,
		tableName, tableName, tableName, tableName, tableName)

	fmt.Printf("UPDATE \"%s\" SET search_vector = %s;\n",
		tableName, strings.ReplaceAll(strings.Join(searchFields, " || \n    "), "NEW.", ""))
}

func init() {
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is config/config.yaml)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(api.APICmd)
	rootCmd.AddCommand(db.DbCmd)
	rootCmd.AddCommand(desk.DeskCmd)
	rootCmd.AddCommand(encryption.EncryptionCmd)
	rootCmd.AddCommand(redis.RedisCmd)
	rootCmd.AddCommand(worker.WorkerCmd)
	rootCmd.AddCommand(update.UpdateCmd)
	searchVectorCmd.Flags().BoolP("list", "l", false, "List all searchable domains")
	searchVectorCmd.Flags().BoolP("all", "a", false, "Generate SQL for all searchable domains")
	rootCmd.AddCommand(searchVectorCmd)
}
