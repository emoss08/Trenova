/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package osm

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/sourcegraph/conc"
)

// Importer handles importing OSM data into the database
type Importer struct {
	db             *pgxpool.Pool
	nodeMap        map[int64]int64   // osm_id -> db_id mapping (pre-allocated)
	wayNodes       map[int64][]int64 // way_id -> []node_ids (pre-allocated)
	wayNodesMu     sync.RWMutex
	processedWays  atomic.Int64
	processedNodes atomic.Int64
	startTime      time.Time
}

// NewImporter creates a new OSM importer
func NewImporter(db *pgxpool.Pool) *Importer {
	return &Importer{
		db:        db,
		nodeMap:   make(map[int64]int64, 10000000),  // Pre-allocate for 10M nodes
		wayNodes:  make(map[int64][]int64, 1000000), // Pre-allocate for 1M ways
		startTime: time.Now(),
	}
}

// ImportPBF imports OSM data from a PBF file
func (i *Importer) ImportPBF(ctx context.Context, reader io.Reader) error {
	log.Println("Starting OSM import...")

	// _ Create temporary tables for faster bulk inserts
	log.Println("Creating temporary tables...")
	if err := i.createTempTables(ctx); err != nil {
		return fmt.Errorf("creating temp tables: %w", err)
	}
	log.Println("Temporary tables created successfully")

	log.Println("Initializing OSM scanner...")
	scanner := osmpbf.New(ctx, reader, 8) // Increased parallelism
	defer scanner.Close()
	log.Println("Scanner initialized, starting to read file...")

	// _ First pass: collect nodes and ways
	nodesChan := make(chan []*osm.Node, 100) // Batch channel
	waysChan := make(chan *osm.Way, 10000)

	wg := conc.NewWaitGroup()

	// _ Node batch collector
	nodeBatch := make([]*osm.Node, 0, 10000)
	nodeCollectorDone := make(chan struct{})
	singleNodeChan := make(chan *osm.Node, 10000)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		defer close(nodeCollectorDone)

		for {
			select {
			case node, ok := <-singleNodeChan:
				if !ok {
					if len(nodeBatch) > 0 {
						nodesChan <- nodeBatch
					}
					close(nodesChan)
					return
				}
				nodeBatch = append(nodeBatch, node)
				if len(nodeBatch) >= 10000 {
					nodesChan <- nodeBatch
					nodeBatch = make([]*osm.Node, 0, 10000)
				}
			case <-ticker.C:
				if len(nodeBatch) > 0 {
					nodesChan <- nodeBatch
					nodeBatch = make([]*osm.Node, 0, 10000)
				}
			}
		}
	}()

	// _ Multiple node processors for parallelism
	for j := 0; j < 4; j++ {
		wg.Go(func() {
			i.processNodeBatches(ctx, nodesChan)
		})
	}

	// _ Way collector (store for second pass)
	var ways []*osm.Way
	var waysMu sync.Mutex
	wg.Go(func() {
		for way := range waysChan {
			// _ Filter only driveable roads
			if isDriveableRoad(way) {
				waysMu.Lock()
				ways = append(ways, way)
				waysMu.Unlock()
				// _ Store node references
				nodeIDs := make([]int64, len(way.Nodes))
				for idx, node := range way.Nodes {
					nodeIDs[idx] = int64(node.ID)
				}
				i.wayNodesMu.Lock()
				i.wayNodes[int64(way.ID)] = nodeIDs
				i.wayNodesMu.Unlock()
			}
		}
	})

	// _ Progress reporter
	progressDone := make(chan struct{})
	go i.reportProgress(progressDone)

	// _ Scan the file
	for scanner.Scan() {
		switch obj := scanner.Object().(type) {
		case *osm.Node:
			singleNodeChan <- obj
		case *osm.Way:
			waysChan <- obj
		}
	}

	close(singleNodeChan)
	<-nodeCollectorDone
	close(waysChan)
	wg.Wait()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning OSM file: %w", err)
	}

	log.Printf(
		"Processed %d nodes in %.2f seconds",
		i.processedNodes.Load(),
		time.Since(i.startTime).Seconds(),
	)

	// _ Finalize node imports from temp table
	if err := i.finalizeNodes(ctx); err != nil {
		return fmt.Errorf("finalizing nodes: %w", err)
	}

	// _ Second pass: process ways and create edges
	log.Println("Processing ways and creating edges...")
	if err := i.processWays(ctx, ways); err != nil {
		return fmt.Errorf("processing ways: %w", err)
	}

	// _ Finalize edge imports
	if err := i.finalizeEdges(ctx); err != nil {
		return fmt.Errorf("finalizing edges: %w", err)
	}

	close(progressDone)
	log.Printf(
		"Import complete. Processed %d ways in %.2f seconds",
		i.processedWays.Load(),
		time.Since(i.startTime).Seconds(),
	)

	return nil
}

func (i *Importer) createTempTables(ctx context.Context) error {
	// _ First terminate any conflicting connections
	log.Println("Cleaning up any existing temp tables...")
	_, err := i.db.Exec(ctx, `
		SELECT pg_terminate_backend(pid) 
		FROM pg_stat_activity 
		WHERE datname = current_database() 
		AND pid <> pg_backend_pid()
		AND query LIKE '%temp_nodes%' OR query LIKE '%temp_edges%';
	`)
	if err != nil {
		log.Printf("Warning: Could not terminate conflicting connections: %v", err)
	}

	// _ Drop tables with CASCADE to handle dependencies
	_, err = i.db.Exec(ctx, `DROP TABLE IF EXISTS temp_nodes CASCADE`)
	if err != nil {
		return fmt.Errorf("dropping temp_nodes: %w", err)
	}

	_, err = i.db.Exec(ctx, `DROP TABLE IF EXISTS temp_edges CASCADE`)
	if err != nil {
		return fmt.Errorf("dropping temp_edges: %w", err)
	}

	// _ Create new temp tables
	log.Println("Creating fresh temp tables...")
	_, err = i.db.Exec(ctx, `
		CREATE UNLOGGED TABLE temp_nodes (
			osm_id BIGINT PRIMARY KEY,
			location GEOMETRY(Point, 4326)
		)
	`)
	if err != nil {
		return fmt.Errorf("creating temp_nodes: %w", err)
	}

	_, err = i.db.Exec(ctx, `
		CREATE UNLOGGED TABLE temp_edges (
			from_osm_id BIGINT,
			to_osm_id BIGINT,
			distance DOUBLE PRECISION,
			travel_time DOUBLE PRECISION,
			max_height DOUBLE PRECISION,
			max_weight DOUBLE PRECISION,
			truck_allowed BOOLEAN,
			road_type TEXT,
			osm_way_id BIGINT
		)
	`)
	if err != nil {
		return fmt.Errorf("creating temp_edges: %w", err)
	}

	return nil
}

func (i *Importer) processNodeBatches(ctx context.Context, batches <-chan []*osm.Node) {
	for batch := range batches {
		i.insertNodeBatch(ctx, batch)
	}
}

func (i *Importer) insertNodeBatch(ctx context.Context, nodes []*osm.Node) {
	if len(nodes) == 0 {
		return
	}

	// _ Direct COPY to temp_nodes table
	conn, err := i.db.Acquire(ctx)
	if err != nil {
		log.Printf("Error acquiring connection: %v", err)
		return
	}
	defer conn.Release()

	// _ Build multi-value INSERT with ST_MakePoint - more reliable than COPY for geometry
	// _ Split into smaller chunks to avoid query size limits
	chunkSize := 1000
	for start := 0; start < len(nodes); start += chunkSize {
		end := start + chunkSize
		if end > len(nodes) {
			end = len(nodes)
		}
		chunk := nodes[start:end]

		// _ Build VALUES clause
		values := make([]string, 0, len(chunk))
		for _, node := range chunk {
			values = append(values, fmt.Sprintf("(%d, ST_SetSRID(ST_MakePoint(%f, %f), 4326))",
				node.ID, node.Lon, node.Lat))
		}

		// _ Execute INSERT
		query := fmt.Sprintf(`
			INSERT INTO temp_nodes (osm_id, location)
			VALUES %s
			ON CONFLICT (osm_id) DO NOTHING
		`, strings.Join(values, ","))

		_, err := conn.Exec(ctx, query)
		if err != nil {
			log.Printf("Error inserting node chunk: %v", err)
			continue
		}

		i.processedNodes.Add(int64(len(chunk)))
	}
}

func (i *Importer) finalizeNodes(ctx context.Context) error {
	log.Println("Finalizing nodes...")

	// _ Check if we need to clear existing data
	var existingCount int64
	err := i.db.QueryRow(ctx, "SELECT COUNT(*) FROM nodes").Scan(&existingCount)
	if err != nil {
		return fmt.Errorf("checking existing nodes: %w", err)
	}

	if existingCount > 0 {
		log.Printf("Found %d existing nodes, truncating...", existingCount)
		_, err = i.db.Exec(ctx, "TRUNCATE nodes CASCADE")
		if err != nil {
			return fmt.Errorf("truncating nodes: %w", err)
		}
	}

	// _ Drop indexes for faster insert
	log.Println("Dropping indexes for faster bulk insert...")
	_, err = i.db.Exec(ctx, `
		DROP INDEX IF EXISTS idx_nodes_osm_id;
		DROP INDEX IF EXISTS idx_nodes_location_cluster;
	`)
	if err != nil {
		log.Printf("Warning: Could not drop indexes: %v", err)
	}

	// _ Count total nodes to insert
	var totalNodes int64
	err = i.db.QueryRow(ctx, "SELECT COUNT(*) FROM temp_nodes").Scan(&totalNodes)
	if err != nil {
		return fmt.Errorf("counting temp nodes: %w", err)
	}
	log.Printf("Total nodes to insert: %d", totalNodes)

	// _ Bulk insert in chunks for better performance and progress tracking
	log.Println("Bulk inserting nodes from temp table in chunks...")
	start := time.Now()
	const chunkSize = 5000000 // 5M rows per chunk
	var totalInserted int64

	for offset := int64(0); offset < totalNodes; offset += chunkSize {
		chunkStart := time.Now()
		limit := int64(chunkSize)
		if offset+limit > totalNodes {
			limit = totalNodes - offset
		}

		// _ Insert chunk using LIMIT/OFFSET
		result, err := i.db.Exec(ctx, fmt.Sprintf(`
			INSERT INTO nodes (osm_id, location)
			SELECT osm_id, location FROM temp_nodes
			ORDER BY osm_id
			LIMIT %d OFFSET %d
		`, limit, offset))
		if err != nil {
			return fmt.Errorf("inserting chunk at offset %d: %w", offset, err)
		}

		chunkRows := result.RowsAffected()
		totalInserted += chunkRows
		chunkElapsed := time.Since(chunkStart)
		overallElapsed := time.Since(start)

		log.Printf(
			"Inserted chunk %d-%d (%.2f%%) in %.1fs - Overall: %d nodes in %.1fs (%.0f nodes/sec)",
			offset,
			offset+chunkRows,
			float64(totalInserted)/float64(totalNodes)*100,
			chunkElapsed.Seconds(),
			totalInserted,
			overallElapsed.Seconds(),
			float64(totalInserted)/overallElapsed.Seconds(),
		)
	}

	elapsed := time.Since(start)
	log.Printf("Completed: Inserted %d nodes in %.2f seconds (%.0f nodes/sec)",
		totalInserted, elapsed.Seconds(), float64(totalInserted)/elapsed.Seconds())

	// _ Create index on osm_id for faster lookups during edge creation
	log.Println("Creating index on osm_id for faster edge processing...")
	_, err = i.db.Exec(ctx, `CREATE INDEX idx_nodes_osm_id ON nodes(osm_id)`)
	if err != nil {
		log.Printf("Warning: Could not create osm_id index: %v", err)
	}

	// _ Drop temp table
	log.Println("Dropping temporary nodes table...")
	_, err = i.db.Exec(ctx, "DROP TABLE temp_nodes")
	if err != nil {
		log.Printf("Warning: Could not drop temp table: %v", err)
	}

	return nil
}

func (i *Importer) processWays(ctx context.Context, ways []*osm.Way) error {
	// _ Process ways in parallel batches
	wg := conc.NewWaitGroup()
	batchSize := 1000
	numWorkers := 8

	wayChan := make(chan []*osm.Way, numWorkers)

	// _ Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Go(func() {
			for batch := range wayChan {
				i.processWayBatch(ctx, batch)
			}
		})
	}

	// _ Send batches to workers
	batch := make([]*osm.Way, 0, batchSize)
	for _, way := range ways {
		batch = append(batch, way)
		if len(batch) >= batchSize {
			wayChan <- batch
			batch = make([]*osm.Way, 0, batchSize)
		}
	}
	if len(batch) > 0 {
		wayChan <- batch
	}

	close(wayChan)
	wg.Wait()
	return nil
}

func (i *Importer) processWayBatch(ctx context.Context, ways []*osm.Way) {
	if len(ways) == 0 {
		return
	}

	// _ Prepare edge data for batch insert

	edges := make([]edge, 0, len(ways)*10) // Estimate ~10 edges per way

	for _, way := range ways {
		i.wayNodesMu.RLock()
		nodeIDs, ok := i.wayNodes[int64(way.ID)]
		i.wayNodesMu.RUnlock()

		if !ok {
			continue
		}

		// _ Create edges between consecutive nodes
		for j := 0; j < len(nodeIDs)-1; j++ {
			fromOSMID := nodeIDs[j]
			toOSMID := nodeIDs[j+1]

			// _ Calculate edge properties
			distance := calculateDistance(way.Nodes[j], way.Nodes[j+1])
			travelTime := calculateTravelTime(distance, way)
			restrictions := extractRestrictions(way)

			e := edge{
				fromOSMID:    fromOSMID,
				toOSMID:      toOSMID,
				distance:     distance,
				travelTime:   travelTime,
				maxHeight:    restrictions.maxHeight,
				maxWeight:    restrictions.maxWeight,
				truckAllowed: restrictions.truckAllowed,
				roadType:     getRoadType(way),
				osmWayID:     int64(way.ID),
			}

			edges = append(edges, e)

			// _ Add reverse edge for two-way roads
			if !isOneWay(way) {
				reverseEdge := e
				reverseEdge.fromOSMID = toOSMID
				reverseEdge.toOSMID = fromOSMID
				edges = append(edges, reverseEdge)
			}
		}

		i.processedWays.Add(1)
	}

	// _ Batch insert edges using COPY
	if len(edges) > 0 {
		i.insertEdgeBatch(ctx, edges)
	}
}

type edge struct {
	fromOSMID    int64
	toOSMID      int64
	distance     float64
	travelTime   float64
	maxHeight    float64
	maxWeight    float64
	truckAllowed bool
	roadType     string
	osmWayID     int64
}

func (i *Importer) insertEdgeBatch(ctx context.Context, edges []edge) {
	if len(edges) == 0 {
		return
	}

	conn, err := i.db.Acquire(ctx)
	if err != nil {
		log.Printf("Error acquiring connection for edges: %v", err)
		return
	}
	defer conn.Release()

	// _ Use COPY for edges - much faster than batch
	_, err = conn.CopyFrom(
		ctx,
		pgx.Identifier{"temp_edges"},
		[]string{"from_osm_id", "to_osm_id", "distance", "travel_time",
			"max_height", "max_weight", "truck_allowed", "road_type", "osm_way_id"},
		pgx.CopyFromSlice(len(edges), func(i int) ([]interface{}, error) {
			e := edges[i]
			return []interface{}{
				e.fromOSMID,
				e.toOSMID,
				e.distance,
				e.travelTime,
				e.maxHeight,
				e.maxWeight,
				e.truckAllowed,
				e.roadType,
				e.osmWayID,
			}, nil
		}),
	)

	if err != nil {
		log.Printf("Error inserting edge batch: %v", err)
	}
}

func (i *Importer) finalizeEdges(ctx context.Context) error {
	log.Println("Finalizing edges...")

	// _ Create indexes on temp_edges for faster joins
	log.Println("Creating indexes on temp_edges for faster processing...")
	_, err := i.db.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_temp_edges_from ON temp_edges(from_osm_id);
		CREATE INDEX IF NOT EXISTS idx_temp_edges_to ON temp_edges(to_osm_id);
	`)
	if err != nil {
		log.Printf("Warning: Could not create temp indexes: %v", err)
	}

	// _ Convert OSM IDs to database IDs and insert in batches
	log.Println("Inserting edges with node ID resolution...")
	result, err := i.db.Exec(ctx, `
		INSERT INTO edges (from_node_id, to_node_id, distance, travel_time,
			max_height, max_weight, truck_allowed, road_type, osm_way_id)
		SELECT 
			n1.id, n2.id, te.distance, te.travel_time,
			te.max_height, te.max_weight, te.truck_allowed, te.road_type, te.osm_way_id
		FROM temp_edges te
		JOIN nodes n1 ON n1.osm_id = te.from_osm_id
		JOIN nodes n2 ON n2.osm_id = te.to_osm_id
		ON CONFLICT (from_node_id, to_node_id) DO UPDATE SET
			distance = EXCLUDED.distance,
			travel_time = EXCLUDED.travel_time,
			max_height = EXCLUDED.max_height,
			max_weight = EXCLUDED.max_weight,
			truck_allowed = EXCLUDED.truck_allowed,
			road_type = EXCLUDED.road_type,
			osm_way_id = EXCLUDED.osm_way_id
	`)

	if err != nil {
		return fmt.Errorf("finalizing edges: %w", err)
	}

	rowsAffected := result.RowsAffected()
	log.Printf("Inserted/updated %d edges", rowsAffected)

	// _ Drop temp table
	_, err = i.db.Exec(ctx, "DROP TABLE temp_edges")
	return err
}

func (i *Importer) reportProgress(done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nodes := i.processedNodes.Load()
			ways := i.processedWays.Load()
			elapsed := time.Since(i.startTime).Seconds()
			nodesPerSec := float64(nodes) / elapsed
			waysPerSec := float64(ways) / elapsed

			log.Printf("Progress: %d nodes (%.0f/sec), %d ways (%.0f/sec)",
				nodes, nodesPerSec, ways, waysPerSec)
		case <-done:
			return
		}
	}
}

type restrictions struct {
	maxHeight    float64
	maxWeight    float64
	truckAllowed bool
}

func isDriveableRoad(way *osm.Way) bool {
	highway := way.Tags.Find("highway")
	if highway == "" {
		return false
	}

	// _ Include major road types
	driveableTypes := map[string]bool{
		"motorway":       true,
		"trunk":          true,
		"primary":        true,
		"secondary":      true,
		"tertiary":       true,
		"unclassified":   true,
		"residential":    true,
		"motorway_link":  true,
		"trunk_link":     true,
		"primary_link":   true,
		"secondary_link": true,
		"tertiary_link":  true,
	}

	return driveableTypes[highway]
}

func isOneWay(way *osm.Way) bool {
	oneway := way.Tags.Find("oneway")
	return oneway == "yes" || oneway == "true" || oneway == "1"
}

func getRoadType(way *osm.Way) string {
	return way.Tags.Find("highway")
}

func extractRestrictions(way *osm.Way) restrictions {
	r := restrictions{
		maxHeight:    0, // 0 means no restriction
		maxWeight:    0,
		truckAllowed: true,
	}

	// _ Check for height restrictions
	if maxHeight := way.Tags.Find("maxheight"); maxHeight != "" {
		// _ Parse height (simplified - in production, handle units properly)
		// _ For now, assume meters
		var height float64
		_, _ = fmt.Sscanf(maxHeight, "%f", &height)
		r.maxHeight = height
	}

	// _ Check for weight restrictions
	if maxWeight := way.Tags.Find("maxweight"); maxWeight != "" {
		// _ Parse weight (simplified - in production, handle units properly)
		// _ For now, assume tons, convert to kg
		var weight float64
		_, _ = fmt.Sscanf(maxWeight, "%f", &weight)
		r.maxWeight = weight * 1000
	}

	// _ Check if trucks are restricted
	if hgv := way.Tags.Find("hgv"); hgv == "no" {
		r.truckAllowed = false
	}

	return r
}

func calculateDistance(n1, n2 osm.WayNode) float64 {
	// _ Use haversine formula
	p1 := orb.Point{n1.Lon, n1.Lat}
	p2 := orb.Point{n2.Lon, n2.Lat}
	return haversineDistance(p1, p2)
}

// haversineDistance calculates the distance between two points using the haversine formula
func haversineDistance(p1, p2 orb.Point) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := p1[1] * (math.Pi / 180)
	lat2Rad := p2[1] * (math.Pi / 180)
	deltaLat := (p2[1] - p1[1]) * (math.Pi / 180)
	deltaLon := (p2[0] - p1[0]) * (math.Pi / 180)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func calculateTravelTime(distance float64, way *osm.Way) float64 {
	// _ Get speed limit or use defaults based on road type
	speedKmh := getSpeedLimit(way)
	if speedKmh == 0 {
		speedKmh = getDefaultSpeed(way)
	}

	// _ Convert to m/s and calculate time
	speedMs := speedKmh / 3.6
	return distance / speedMs
}

func getSpeedLimit(way *osm.Way) float64 {
	if maxSpeed := way.Tags.Find("maxspeed"); maxSpeed != "" {
		var speed float64
		_, _ = fmt.Sscanf(maxSpeed, "%f", &speed)
		return speed
	}
	return 0
}

func getDefaultSpeed(way *osm.Way) float64 {
	highway := way.Tags.Find("highway")

	// _ Default speeds in km/h for trucks
	speeds := map[string]float64{
		"motorway":       80,
		"trunk":          70,
		"primary":        60,
		"secondary":      50,
		"tertiary":       40,
		"unclassified":   30,
		"residential":    25,
		"motorway_link":  50,
		"trunk_link":     40,
		"primary_link":   40,
		"secondary_link": 30,
		"tertiary_link":  30,
	}

	if speed, ok := speeds[highway]; ok {
		return speed
	}

	return 30 // Default fallback
}
