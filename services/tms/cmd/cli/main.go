package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/cmd/cli/api"
	"github.com/emoss08/trenova/cmd/cli/codegen"
	"github.com/emoss08/trenova/cmd/cli/db"
	"github.com/emoss08/trenova/cmd/cli/redis"
	"github.com/emoss08/trenova/cmd/cli/worker"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/domainregistry"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

func main() {
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
		loader := config.NewLoader(config.WithConfigPath("config"))
		var err error
		cfg, err = loader.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		// Set config for db package
		db.SetConfig(cfg)
		// Set config for redis package
		redis.SetConfig(cfg)
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Trenova %s\n", cfg.App.Version)
		fmt.Printf("Environment: %s\n", cfg.App.Env)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration files",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("✓ Configuration is valid")
		fmt.Printf("  Environment: %s\n", cfg.App.Env)
		fmt.Printf("  Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
		fmt.Printf(
			"  Database: postgres://%s:****@%s:%d/%s\n",
			cfg.Database.User,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.Name,
		)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := "config/config.yaml"
		if cfgFile != "" {
			configPath = cfgFile
		}

		if _, err := exec.LookPath("bat"); err == nil {
			cmd := exec.Command("bat", "--style=full", "--language=yaml", configPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}

		if _, err := exec.LookPath("cat"); err == nil {
			fmt.Printf("Configuration file: %s\n", configPath)
			fmt.Println(strings.Repeat("=", 80))
			cmd := exec.Command("cat", configPath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}

		content, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		fmt.Printf("Configuration file: %s\n", configPath)
		fmt.Println(strings.Repeat("=", 80))
		fmt.Print(string(content))
		return nil
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
				generateSQL(searchable, config, domainName)
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
				generateSQL(searchable, config, domainName)
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
	domainName string,
) {
	tableName := searchable.GetTableName()

	fmt.Printf("-- Search Vector and Relationship Indexes SQL for %s\n", domainName)
	fmt.Printf("-- Generated for optimal query performance including relationships\n")

	fmt.Printf("-- 1. Search Vector Column\n")
	fmt.Printf("ALTER TABLE \"%s\" ADD COLUMN IF NOT EXISTS search_vector tsvector;\n\n", tableName)
	fmt.Println("--bun:split")

	fmt.Printf("-- 2. Search Vector Index (GIN for full-text search)\n")
	fmt.Printf(
		"CREATE INDEX IF NOT EXISTS idx_%s_search_vector ON \"%s\" USING GIN(search_vector);\n\n",
		tableName,
		tableName,
	)
	fmt.Println("--bun:split")

	if len(config.Relationships) > 0 {
		fmt.Printf("-- 3. Relationship Indexes (only commonly queried relationships)\n")
		fmt.Printf("-- Note: Only add these if you actually filter/join on these relationships\n")
		for _, rel := range config.Relationships {
			if !rel.Queryable {
				continue
			}

			switch rel.Type {
			case domaintypes.RelationshipTypeBelongsTo:
				fmt.Printf(
					"-- Index for %s (belongs-to) - ADD ONLY if you JOIN or filter on this\n",
					rel.Field,
				)
				fmt.Printf("-- CREATE INDEX IF NOT EXISTS idx_%s_%s ON \"%s\"(\"%s\");\n\n",
					tableName, rel.ForeignKey, tableName, rel.ForeignKey)

			case domaintypes.RelationshipTypeHasOne, domaintypes.RelationshipTypeHasMany:
				fmt.Printf(
					"-- Index for %s (%s) - UNCOMMENT only if you query from the reverse side\n",
					rel.Field,
					rel.Type,
				)
				fmt.Printf("-- CREATE INDEX IF NOT EXISTS idx_%s_%s ON \"%s\"(\"%s\");\n\n",
					rel.TargetTable, rel.ForeignKey, rel.TargetTable, rel.ForeignKey)

			case domaintypes.RelationshipTypeManyToMany:
				fmt.Printf(
					"-- Indexes for %s (many-to-many) - IMPORTANT for M2M queries\n",
					rel.Field,
				)

				fmt.Printf("-- Primary lookup index (from %s to %s)\n", tableName, rel.TargetTable)
				fmt.Printf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON \"%s\"(\"%s\");\n",
					rel.JoinTable, rel.JoinTableSourceKey, rel.JoinTable, rel.JoinTableSourceKey)

				fmt.Printf("-- Reverse lookup index (from %s to %s)\n", rel.TargetTable, tableName)
				fmt.Printf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON \"%s\"(\"%s\");\n",
					rel.JoinTable, rel.JoinTableTargetKey, rel.JoinTable, rel.JoinTableTargetKey)

				fmt.Printf(
					"-- Optional: Composite index for covering index scans (uncomment if needed)\n",
				)
				fmt.Printf(
					"-- CREATE INDEX IF NOT EXISTS idx_%s_%s_%s ON \"%s\"(\"%s\", \"%s\");\n\n",
					rel.JoinTable,
					rel.JoinTableSourceKey,
					rel.JoinTableTargetKey,
					rel.JoinTable,
					rel.JoinTableSourceKey,
					rel.JoinTableTargetKey,
				)
			}
		}
		fmt.Println("--bun:split")
	}

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

	fmt.Printf("-- 4. Search Vector Trigger Function\n")
	fmt.Printf(`CREATE OR REPLACE FUNCTION %s_search_trigger() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := %s;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

`, tableName, strings.Join(searchFields, " || \n        "))

	fmt.Println("--bun:split")

	fmt.Printf("-- 5. Search Vector Trigger\n")
	fmt.Printf(`DROP TRIGGER IF EXISTS %s_search_update ON "%s";
CREATE TRIGGER %s_search_update
    BEFORE INSERT OR UPDATE ON "%s"
    FOR EACH ROW
    EXECUTE FUNCTION %s_search_trigger();

`,
		tableName, tableName, tableName, tableName, tableName)

	fmt.Println("--bun:split")

	fmt.Printf("-- 6. Update Existing Rows\n")
	fmt.Printf("UPDATE \"%s\" SET search_vector = %s;\n",
		tableName, strings.ReplaceAll(strings.Join(searchFields, " || \n    "), "NEW.", ""))

	fmt.Printf("\n-- Performance Optimization Notes:\n")
	fmt.Printf("-- - Only add indexes that match your actual query patterns\n")
	fmt.Printf("-- - Run EXPLAIN ANALYZE on slow queries to identify missing indexes\n")
	fmt.Printf("-- - After adding indexes, run: ANALYZE \"%s\";\n", tableName)
	if len(config.Relationships) > 0 {
		fmt.Printf("-- - Monitor pg_stat_user_indexes to find unused indexes\n")
		fmt.Printf("-- - Consider partial indexes for large tables with WHERE clauses\n")
	}

	fmt.Printf("\n-- Useful Analysis Queries:\n")
	fmt.Printf("-- Check index usage:\n")
	fmt.Printf(
		"-- SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch\n",
	)
	fmt.Printf(
		"-- FROM pg_stat_user_indexes WHERE tablename = '%s' ORDER BY idx_scan;\n",
		tableName,
	)

	fmt.Printf("\n-- Find missing indexes (run after typical workload):\n")
	fmt.Printf("-- SELECT * FROM pg_stat_user_tables WHERE tablename = '%s';\n", tableName)
	fmt.Printf("-- (Look for high seq_scan with high n_tup_fetched)\n")
}

func init() {
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is config/config.yaml)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(api.APICmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(redis.RedisCmd)
	rootCmd.AddCommand(db.DbCmd)
	rootCmd.AddCommand(worker.WorkerCmd)
	rootCmd.AddCommand(codegen.CodegenCmd)
	searchVectorCmd.Flags().BoolP("list", "l", false, "List all searchable domains")
	searchVectorCmd.Flags().BoolP("all", "a", false, "Generate SQL for all searchable domains")
	rootCmd.AddCommand(searchVectorCmd)
}
