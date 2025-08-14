/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/emoss08/routing/internal/graph"
	"github.com/emoss08/routing/internal/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	// _ Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	// _ Parse command line flags
	var (
		outputFile = flag.String("output", "graph.png", "Output file path")
		format     = flag.String("format", "png", "Output format (png, svg, pdf)")
		maxNodes   = flag.Int("max-nodes", 1000, "Maximum number of nodes to visualize")
		region     = flag.String("region", "", "Region to visualize (format: lat1,lon1,lat2,lon2)")
		zipCode    = flag.String("zip", "", "Visualize area around a specific zip code")
		radius     = flag.Float64("radius", 5.0, "Radius in miles for zip code visualization")
	)
	flag.Parse()

	// _ Load configuration
	if err := loadConfig(); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// _ Create context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// _ Initialize PostgreSQL
	dbDSN := viper.GetString("database.dsn")
	if dbDSN == "" {
		dbDSN = "postgres://postgres:password@localhost:5432/routing?sslmode=disable"
	}

	storage, err := storage.NewPostgresStorage(ctx, dbDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer storage.Close()

	// _ Determine bounds
	var bounds *graph.Bounds
	if *region != "" {
		bounds, err = parseBounds(*region)
		if err != nil {
			log.Fatal().Err(err).Msg("Invalid region format")
		}
	} else if *zipCode != "" {
		bounds, err = getBoundsForZip(ctx, storage, *zipCode, *radius)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get bounds for zip code")
		}
	} else {
		// _ Default to a small area for demonstration
		bounds = &graph.Bounds{
			MinLat: 34.0, MaxLat: 34.1,
			MinLon: -118.3, MaxLon: -118.2,
		}
		log.Info().Msg("No region specified, using default area (Los Angeles)")
	}

	// _ Generate DOT file
	dotFile, err := generateDotFile(ctx, storage, bounds, *maxNodes)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate DOT file")
	}
	defer os.Remove(dotFile)

	// _ Convert DOT to image using Graphviz
	if err := renderGraph(dotFile, *outputFile, *format); err != nil {
		log.Fatal().Err(err).Msg("Failed to render graph")
	}

	log.Info().
		Str("output", *outputFile).
		Str("format", *format).
		Msg("Graph visualization generated successfully")
}

func loadConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/routing/")

	// _ Set defaults
	viper.SetDefault("database.max_connections", 25)

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

func parseBounds(region string) (*graph.Bounds, error) {
	parts := strings.Split(region, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("region must be in format: lat1,lon1,lat2,lon2")
	}

	var coords [4]float64
	for i, part := range parts {
		if _, err := fmt.Sscanf(part, "%f", &coords[i]); err != nil {
			return nil, fmt.Errorf("invalid coordinate: %s", part)
		}
	}

	return &graph.Bounds{
		MinLat: coords[0],
		MinLon: coords[1],
		MaxLat: coords[2],
		MaxLon: coords[3],
	}, nil
}

func getBoundsForZip(
	ctx context.Context,
	st *storage.PostgresStorage,
	zipCode string,
	radiusMiles float64,
) (*graph.Bounds, error) {
	// _ Get the node associated with the zip code
	node, err := st.GetNodeForZip(ctx, zipCode)
	if err != nil {
		return nil, fmt.Errorf("getting node for zip %s: %w", zipCode, err)
	}

	// _ Calculate bounds based on radius
	// ! Rough approximation: 1 degree latitude ≈ 69 miles
	latDelta := radiusMiles / 69.0
	lonDelta := radiusMiles / (69.0 * 0.86) // Adjust for longitude at ~34° latitude

	return &graph.Bounds{
		MinLat: node.Lat - latDelta,
		MaxLat: node.Lat + latDelta,
		MinLon: node.Lon - lonDelta,
		MaxLon: node.Lon + lonDelta,
	}, nil
}

func generateDotFile(
	ctx context.Context,
	st *storage.PostgresStorage,
	bounds *graph.Bounds,
	maxNodes int,
) (string, error) {
	// _ Create temporary DOT file
	tmpFile, err := os.CreateTemp("", "graph-*.dot")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer tmpFile.Close()

	// _ Write DOT header
	fmt.Fprintln(tmpFile, "digraph RoadNetwork {")
	fmt.Fprintln(tmpFile, "  rankdir=LR;")
	fmt.Fprintln(tmpFile, "  node [shape=point, width=0.1, height=0.1, color=blue];")
	fmt.Fprintln(tmpFile, "  edge [color=gray, arrowsize=0.5];")
	fmt.Fprintln(tmpFile, "  overlap=false;")
	fmt.Fprintln(tmpFile, "  splines=true;")
	fmt.Fprintln(tmpFile)

	// _ Load nodes within bounds
	nodes, err := st.GetNodesInBounds(
		ctx,
		bounds.MinLat,
		bounds.MinLon,
		bounds.MaxLat,
		bounds.MaxLon,
		maxNodes,
	)
	if err != nil {
		return "", fmt.Errorf("loading nodes: %w", err)
	}

	log.Info().Int("count", len(nodes)).Msg("Loaded nodes")

	// _ Create a set of node IDs for filtering edges
	nodeSet := make(map[int64]bool)
	for _, node := range nodes {
		nodeSet[node.ID] = true
	}

	// _ Write nodes
	for _, node := range nodes {
		// _ Position nodes based on longitude/latitude
		// ! Scale coordinates for better visualization
		x := (node.Lon - bounds.MinLon) * 100
		y := (node.Lat - bounds.MinLat) * 100
		fmt.Fprintf(tmpFile, "  n%d [pos=\"%.2f,%.2f!\"];\n", node.ID, x, y)
	}

	fmt.Fprintln(tmpFile)

	// _ Load and write edges
	edgeCount := 0
	for _, node := range nodes {
		edges, err := st.GetOutgoingEdges(ctx, node.ID)
		if err != nil {
			log.Warn().Err(err).Int64("node_id", node.ID).Msg("Failed to load edges")
			continue
		}

		for _, edge := range edges {
			// _ Only include edges where both nodes are in our set
			if nodeSet[edge.ToNodeID] {
				// _ Color code by road type or restrictions
				color := "gray"
				if !edge.TruckAllowed {
					color = "red"
				} else if edge.Distance > 10 {
					color = "darkgreen"
				}

				fmt.Fprintf(tmpFile, "  n%d -> n%d [color=%s, penwidth=%.1f];\n",
					edge.FromNodeID, edge.ToNodeID, color, 1.0)
				edgeCount++
			}
		}
	}

	log.Info().Int("count", edgeCount).Msg("Wrote edges")

	// _ Write DOT footer
	fmt.Fprintln(tmpFile, "}")

	return tmpFile.Name(), nil
}

func renderGraph(dotFile, outputFile, format string) error {
	// _ Check if graphviz is installed
	if _, err := exec.LookPath("dot"); err != nil {
		return fmt.Errorf("graphviz not found. Please install it: apt-get install graphviz")
	}

	// _ Create output directory if needed
	outputDir := filepath.Dir(outputFile)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	// _ Run graphviz
	cmd := exec.Command("dot", "-T"+format, "-o", outputFile, dotFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running graphviz: %w", err)
	}

	return nil
}
