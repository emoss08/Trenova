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
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

// SmartImporter only imports nodes that are part of the road network
type SmartImporter struct {
	db             *pgxpool.Pool
	roadNodeIDs    map[int64]bool // Set of node IDs that are part of roads
	roadNodeIDsMu  sync.RWMutex
	nodeCoords     map[int64][]float64 // Node coordinates (osm_id -> [lon, lat])
	nodeCoordsMu   sync.RWMutex
	processedWays  atomic.Int64
	processedNodes atomic.Int64
	skippedNodes   atomic.Int64
	startTime      time.Time
}

// NewSmartImporter creates a new smart OSM importer
func NewSmartImporter(db *pgxpool.Pool) *SmartImporter {
	return &SmartImporter{
		db:          db,
		roadNodeIDs: make(map[int64]bool, 20000000),      // Pre-allocate for ~20M road nodes
		nodeCoords:  make(map[int64][]float64, 20000000), // Pre-allocate for node coordinates
		startTime:   time.Now(),
	}
}

// ImportPBF imports OSM data, only keeping road network nodes
func (si *SmartImporter) ImportPBF(ctx context.Context, reader io.Reader) error {
	log.Println("Starting smart OSM import (road nodes only)...")

	// _ First pass: identify all nodes that are part of driveable roads
	log.Println("Pass 1: Identifying road network nodes...")
	if err := si.identifyRoadNodes(ctx, reader); err != nil {
		return fmt.Errorf("identifying road nodes: %w", err)
	}

	log.Printf("Identified %d nodes that are part of the road network", len(si.roadNodeIDs))

	// _ Reset reader for second pass
	seeker, ok := reader.(io.Seeker)
	if !ok {
		return fmt.Errorf("reader does not support seeking - cannot perform second pass")
	}
	if _, err := seeker.Seek(0, 0); err != nil {
		return fmt.Errorf("seeking to beginning: %w", err)
	}

	// _ Create tables
	if err := si.createTables(ctx); err != nil {
		return fmt.Errorf("creating tables: %w", err)
	}

	// _ Second pass: import only road nodes and ways
	log.Println("Pass 2: Importing road nodes and ways...")
	if err := si.importRoadData(ctx, reader); err != nil {
		return fmt.Errorf("importing road data: %w", err)
	}

	log.Printf("Smart import complete: %d road nodes, %d ways, %d nodes skipped",
		si.processedNodes.Load(), si.processedWays.Load(), si.skippedNodes.Load())

	return nil
}

func (si *SmartImporter) identifyRoadNodes(ctx context.Context, reader io.Reader) error {
	scanner := osmpbf.New(ctx, reader, 8)
	defer scanner.Close()

	wayCount := 0
	for scanner.Scan() {
		switch obj := scanner.Object().(type) {
		case *osm.Way:
			if isDriveableRoad(obj) {
				// _ Add all nodes from this way to our set
				si.roadNodeIDsMu.Lock()
				for _, node := range obj.Nodes {
					si.roadNodeIDs[int64(node.ID)] = true
				}
				si.roadNodeIDsMu.Unlock()
				wayCount++

				if wayCount%100000 == 0 {
					log.Printf("Processed %d ways, found %d road nodes so far", wayCount, len(si.roadNodeIDs))
				}
			}
		}
	}

	return scanner.Err()
}

func (si *SmartImporter) createTables(ctx context.Context) error {
	// _ Drop existing data
	log.Println("Dropping existing data...")
	_, err := si.db.Exec(ctx, "TRUNCATE nodes CASCADE")
	if err != nil {
		log.Printf("Warning: Could not truncate nodes: %v", err)
	}

	return nil
}

func (si *SmartImporter) importRoadData(ctx context.Context, reader io.Reader) error {
	scanner := osmpbf.New(ctx, reader, 8)
	defer scanner.Close()

	// _ Prepare batch inserters
	nodeBatch := make([]roadNode, 0, 10000)
	edgeBatch := make([]edge, 0, 10000)

	// _ Progress reporting
	progressDone := make(chan struct{})
	go si.reportProgress(progressDone)

	// _ Process the file
	for scanner.Scan() {
		switch obj := scanner.Object().(type) {
		case *osm.Node:
			// _ Only import if it's a road node
			if si.roadNodeIDs[int64(obj.ID)] {
				// _ Store node coordinates for edge distance calculation
				si.nodeCoordsMu.Lock()
				si.nodeCoords[int64(obj.ID)] = []float64{obj.Lon, obj.Lat}
				si.nodeCoordsMu.Unlock()

				nodeBatch = append(nodeBatch, roadNode{
					osmID: int64(obj.ID),
					lon:   obj.Lon,
					lat:   obj.Lat,
				})
				si.processedNodes.Add(1)

				if len(nodeBatch) >= 10000 {
					if err := si.insertNodeBatch(ctx, nodeBatch); err != nil {
						log.Printf("Error inserting node batch: %v", err)
					}
					nodeBatch = nodeBatch[:0]
				}
			} else {
				si.skippedNodes.Add(1)
			}

		case *osm.Way:
			if isDriveableRoad(obj) {
				// _ Create edges
				for i := 0; i < len(obj.Nodes)-1; i++ {
					fromID := int64(obj.Nodes[i].ID)
					toID := int64(obj.Nodes[i+1].ID)

					// _ Look up actual coordinates
					si.nodeCoordsMu.RLock()
					fromCoords, fromExists := si.nodeCoords[fromID]
					toCoords, toExists := si.nodeCoords[toID]
					si.nodeCoordsMu.RUnlock()

					if !fromExists || !toExists {
						// _ Skip edges where we don't have node coordinates
						continue
					}

					// _ Calculate actual distance using coordinates
					distance := calculateDistanceFromCoords(fromCoords[1], fromCoords[0], toCoords[1], toCoords[0])
					travelTime := calculateTravelTime(distance, obj)
					restrictions := extractRestrictions(obj)

					e := edge{
						fromOSMID:    fromID,
						toOSMID:      toID,
						distance:     distance,
						travelTime:   travelTime,
						maxHeight:    restrictions.maxHeight,
						maxWeight:    restrictions.maxWeight,
						truckAllowed: restrictions.truckAllowed,
						roadType:     getRoadType(obj),
						osmWayID:     int64(obj.ID),
					}

					edgeBatch = append(edgeBatch, e)

					// _ Add reverse edge for two-way roads
					if !isOneWay(obj) {
						reverseEdge := e
						reverseEdge.fromOSMID = toID
						reverseEdge.toOSMID = fromID
						edgeBatch = append(edgeBatch, reverseEdge)
					}
				}

				if len(edgeBatch) >= 10000 {
					if err := si.insertEdgeBatch(ctx, edgeBatch); err != nil {
						log.Printf("Error inserting edge batch: %v", err)
					}
					edgeBatch = edgeBatch[:0]
				}

				si.processedWays.Add(1)
			}
		}
	}

	// _ Insert remaining batches
	if len(nodeBatch) > 0 {
		if err := si.insertNodeBatch(ctx, nodeBatch); err != nil {
			log.Printf("Error inserting final node batch: %v", err)
		}
	}
	if len(edgeBatch) > 0 {
		if err := si.insertEdgeBatch(ctx, edgeBatch); err != nil {
			log.Printf("Error inserting final edge batch: %v", err)
		}
	}

	close(progressDone)
	return scanner.Err()
}

type roadNode struct {
	osmID int64
	lon   float64
	lat   float64
}

func (si *SmartImporter) insertNodeBatch(ctx context.Context, nodes []roadNode) error {
	if len(nodes) == 0 {
		return nil
	}

	conn, err := si.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	// _ Use batch inserts with ST_MakePoint
	batch := &pgx.Batch{}
	for _, n := range nodes {
		batch.Queue(`
			INSERT INTO nodes (osm_id, location)
			VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326))
			ON CONFLICT (osm_id) DO NOTHING
		`, n.osmID, n.lon, n.lat)
	}

	br := conn.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("inserting node: %w", err)
		}
	}

	return nil
}

func (si *SmartImporter) insertEdgeBatch(ctx context.Context, edges []edge) error {
	if len(edges) == 0 {
		return nil
	}

	conn, err := si.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	// _ Batch insert edges with subquery for node ID lookup
	batch := &pgx.Batch{}
	for _, e := range edges {
		batch.Queue(`
			INSERT INTO edges (from_node_id, to_node_id, distance, travel_time,
				max_height, max_weight, truck_allowed, road_type, osm_way_id)
			SELECT n1.id, n2.id, $3, $4, $5, $6, $7, $8, $9
			FROM nodes n1, nodes n2
			WHERE n1.osm_id = $1 AND n2.osm_id = $2
		`, e.fromOSMID, e.toOSMID, e.distance, e.travelTime,
			e.maxHeight, e.maxWeight, e.truckAllowed, e.roadType, e.osmWayID)
	}

	br := conn.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			// _ Log but don't fail on individual edge errors
			log.Printf("Warning: edge insert error: %v", err)
		}
	}

	return nil
}

func (si *SmartImporter) reportProgress(done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nodes := si.processedNodes.Load()
			ways := si.processedWays.Load()
			skipped := si.skippedNodes.Load()
			elapsed := time.Since(si.startTime).Seconds()
			nodesPerSec := float64(nodes) / elapsed

			log.Printf("Progress: %d road nodes (%.0f/sec), %d ways, %d nodes skipped",
				nodes, nodesPerSec, ways, skipped)
		case <-done:
			return
		}
	}
}

// calculateDistanceFromCoords calculates distance between two coordinate pairs using Haversine formula
func calculateDistanceFromCoords(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := lat1 * (math.Pi / 180)
	lat2Rad := lat2 * (math.Pi / 180)
	deltaLat := (lat2 - lat1) * (math.Pi / 180)
	deltaLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
