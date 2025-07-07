package main

import (
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emoss08/routing/internal/database"
	"github.com/emoss08/routing/internal/osm"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	// _ Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/routing/")

	// _ Set defaults
	viper.SetDefault(
		"database.dsn",
		"postgres://postgres:password@localhost:5432/routing?sslmode=disable",
	)

	// _ Read environment variables
	viper.SetEnvPrefix("ROUTING")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("reading config: %w", err)
		}
		log.Warn().Msg("No config file found, using defaults and environment variables")
	}

	return nil
}

func main() {
	var (
		dbDSN         = flag.String("db", "", "Database connection string (overrides config)")
		osmFile       = flag.String("file", "", "Path to OSM PBF file")
		osmURL        = flag.String("url", "", "URL to download OSM PBF file")
		runMigrate    = flag.Bool("migrate", true, "Run migrations before importing")
		forceDownload = flag.Bool(
			"force-download",
			false,
			"Force re-download even if cached file exists",
		)
	)
	flag.Parse()

	if *osmFile == "" && *osmURL == "" {
		log.Fatal().Msg("Either -file or -url must be specified")
	}

	// _ Load configuration
	if err := loadConfig(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// _ Use config DSN if not provided via flag
	if *dbDSN == "" {
		*dbDSN = viper.GetString("database.dsn")
		if *dbDSN == "" {
			*dbDSN = "postgres://postgres:password@localhost:5432/routing?sslmode=disable"
		}
	}

	ctx := context.Background()

	// _ Run migrations if requested
	if *runMigrate {
		migrator, err := database.NewMigrator(*dbDSN, log.Logger)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create migrator")
		}
		defer migrator.Close()

		if err := migrator.Migrate(ctx); err != nil {
			log.Fatal().Err(err).Msg("Failed to run migrations")
		}
	}

	// _ Connect to database
	pool, err := pgxpool.New(ctx, *dbDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	// _ Create smart importer that only imports road nodes
	importer := osm.NewSmartImporter(pool)

	// _ Open OSM file
	var reader io.Reader
	if *osmFile != "" {
		file, err := os.Open(*osmFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to open OSM file")
		}
		defer file.Close()
		reader = file
	} else {
		// _ Download from URL
		log.Info().Str("url", *osmURL).Msg("Downloading OSM data")
		reader, err = downloadOSM(*osmURL, *forceDownload)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to download OSM data")
		}
	}

	// _ Import data
	start := time.Now()
	if err := importer.ImportPBF(ctx, reader); err != nil {
		log.Fatal().Err(err).Msg("Import failed")
	}

	log.Info().
		Dur("duration", time.Since(start)).
		Msg("Import completed successfully")

	// _ Create indexes and optimize
	if err := createIndexes(ctx, pool); err != nil {
		log.Error().Err(err).Msg("Failed to create indexes")
	}

	// _ Display import statistics
	if err := displayImportStats(ctx, pool); err != nil {
		log.Error().Err(err).Msg("Failed to display statistics")
	}
}

func downloadOSM(url string, forceDownload bool) (io.Reader, error) {
	// _ Create cache directory
	cacheDir := "./.osm-cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	// _ Generate cache filename from URL
	urlParts := strings.Split(url, "/")
	filename := urlParts[len(urlParts)-1]
	cachePath := filepath.Join(cacheDir, filename)

	// _ Check if file exists in cache
	if !forceDownload {
		if stat, err := os.Stat(cachePath); err == nil {
			// _ File exists, check if it's recent (less than 7 days old)
			if time.Since(stat.ModTime()) < 7*24*time.Hour {
				log.Printf(
					"Using cached file: %s (age: %v)",
					cachePath,
					time.Since(stat.ModTime()).Round(time.Hour),
				)
				return os.Open(cachePath)
			}
			log.Printf(
				"Cache file is stale (age: %v), downloading fresh copy",
				time.Since(stat.ModTime()).Round(time.Hour),
			)
		}
	} else {
		log.Printf("Force download requested, ignoring cache")
	}

	// _ Download file
	log.Printf("Downloading %s to cache...", filename)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// _ Save to cache file
	tmpPath := cachePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}

	// _ Show download progress
	contentLength := resp.ContentLength
	if contentLength > 0 {
		log.Printf("Downloading %.2f MB...", float64(contentLength)/(1024*1024))
	}

	// _ Check if gzipped and decompress if needed
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// _ Copy with progress
	written, err := io.Copy(tmpFile, reader)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("saving file: %w", err)
	}
	tmpFile.Close()

	log.Printf("Downloaded %.2f MB successfully", float64(written)/(1024*1024))

	// _ Move temp file to final location
	if err := os.Rename(tmpPath, cachePath); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("moving file to cache: %w", err)
	}

	// _ Open the cached file
	return os.Open(cachePath)
}

func createIndexes(ctx context.Context, pool *pgxpool.Pool) error {
	indexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_edges_distance ON edges(distance)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_edges_travel_time ON edges(travel_time)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_nodes_location_cluster ON nodes USING GIST(location) WITH (fillfactor = 90)",
		"CLUSTER nodes USING idx_nodes_location_cluster",
		"ANALYZE nodes",
		"ANALYZE edges",
	}

	for _, index := range indexes {
		log.Info().Str("query", index).Msg("Creating index")
		if _, err := pool.Exec(ctx, index); err != nil {
			log.Error().Err(err).Str("query", index).Msg("Failed to create index")
		}
	}

	return nil
}

func importZipCodes(ctx context.Context, pool *pgxpool.Pool, regionFilter string) error {
	// _ This would typically import from the zip codes CSV
	// _ For now, we'll just log that it would happen
	log.Info().Str("region", regionFilter).Msg("Zip code import would happen here")

	// _ Run the populate_zip_nodes.sql script logic
	_, err := pool.Exec(ctx, `
		INSERT INTO zip_nodes (zip_code, node_id, centroid, state, city)
		SELECT 
			z.zip_code,
			(
				SELECT n.id 
				FROM nodes n
				ORDER BY n.location <-> z.centroid
				LIMIT 1
			) as node_id,
			z.centroid,
			z.state,
			z.city
		FROM (
			SELECT DISTINCT zip_code, centroid, state, city 
			FROM zip_nodes
		) z
		ON CONFLICT (zip_code) DO UPDATE
		SET node_id = EXCLUDED.node_id
	`)

	return err
}

func displayImportStats(ctx context.Context, pool *pgxpool.Pool) error {
	var stats struct {
		NodeCount      int64
		EdgeCount      int64
		HeightRestrict int64
		WeightRestrict int64
		TruckRestrict  int64
		TollRoads      int64
		HazmatRestrict int64
	}

	err := pool.QueryRow(ctx, `
		SELECT 
			(SELECT COUNT(*) FROM nodes),
			(SELECT COUNT(*) FROM edges),
			(SELECT COUNT(*) FROM edges WHERE max_height > 0),
			(SELECT COUNT(*) FROM edges WHERE max_weight > 0),
			(SELECT COUNT(*) FROM edges WHERE truck_allowed = false),
			(SELECT COUNT(*) FROM edges WHERE toll_road = true),
			(SELECT COUNT(*) FROM edges WHERE hazmat_allowed = false)
	`).Scan(
		&stats.NodeCount,
		&stats.EdgeCount,
		&stats.HeightRestrict,
		&stats.WeightRestrict,
		&stats.TruckRestrict,
		&stats.TollRoads,
		&stats.HazmatRestrict,
	)

	if err != nil {
		return err
	}

	log.Info().
		Int64("nodes", stats.NodeCount).
		Int64("edges", stats.EdgeCount).
		Int64("height_restrictions", stats.HeightRestrict).
		Int64("weight_restrictions", stats.WeightRestrict).
		Int64("truck_restrictions", stats.TruckRestrict).
		Int64("toll_roads", stats.TollRoads).
		Int64("hazmat_restrictions", stats.HazmatRestrict).
		Msg("Import statistics")

	return nil
}
