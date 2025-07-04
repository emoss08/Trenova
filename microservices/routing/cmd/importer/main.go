package main

import (
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/emoss08/routing/internal/database"
	"github.com/emoss08/routing/internal/osm"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// _ Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	var (
		dbDSN       = flag.String("db", "postgres://postgres:password@localhost:5432/routing?sslmode=disable", "Database connection string")
		osmFile     = flag.String("file", "", "Path to OSM PBF file")
		osmURL      = flag.String("url", "", "URL to download OSM PBF file")
		runMigrate  = flag.Bool("migrate", true, "Run migrations before importing")
	)
	flag.Parse()

	if *osmFile == "" && *osmURL == "" {
		log.Fatal().Msg("Either -file or -url must be specified")
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

	// _ Create importer
	importer := osm.NewImporter(pool)

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
		reader, err = downloadOSM(*osmURL)
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
}

func downloadOSM(url string) (io.Reader, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// _ Check if gzipped
	if resp.Header.Get("Content-Encoding") == "gzip" {
		return gzip.NewReader(resp.Body)
	}

	// _ Read into temporary file
	tmpFile, err := os.CreateTemp("", "osm-*.pbf")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("saving file: %w", err)
	}

	// _ Seek to beginning
	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("seeking file: %w", err)
	}

	return tmpFile, nil
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