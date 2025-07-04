package osm

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/sourcegraph/conc"
)

// Importer handles importing OSM data into the database
type Importer struct {
	db           *pgxpool.Pool
	nodeMap      sync.Map // osm_id -> db_id mapping
	wayNodes     sync.Map // way_id -> []node_ids
	processedWays atomic.Int64
	processedNodes atomic.Int64
}

// NewImporter creates a new OSM importer
func NewImporter(db *pgxpool.Pool) *Importer {
	return &Importer{
		db: db,
	}
}

// ImportPBF imports OSM data from a PBF file
func (i *Importer) ImportPBF(ctx context.Context, reader io.Reader) error {
	scanner := osmpbf.New(ctx, reader, 4)
	defer scanner.Close()

	log.Println("Starting OSM import...")

	// _ First pass: collect nodes and ways
	nodesChan := make(chan *osm.Node, 10000)
	waysChan := make(chan *osm.Way, 10000)

	wg := conc.NewWaitGroup()

	// _ Node processor
	wg.Go(func() {
		i.processNodes(ctx, nodesChan)
	})

	// _ Way collector (store for second pass)
	var ways []*osm.Way
	wg.Go(func() {
		for way := range waysChan {
			// _ Filter only driveable roads
			if isDriveableRoad(way) {
				ways = append(ways, way)
				// _ Store node references
				nodeIDs := make([]int64, len(way.Nodes))
				for idx, node := range way.Nodes {
					nodeIDs[idx] = int64(node.ID)
				}
				i.wayNodes.Store(way.ID, nodeIDs)
			}
		}
	})

	// _ Scan the file
	for scanner.Scan() {
		switch obj := scanner.Object().(type) {
		case *osm.Node:
			nodesChan <- obj
		case *osm.Way:
			waysChan <- obj
		}
	}

	close(nodesChan)
	close(waysChan)
	wg.Wait()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning OSM file: %w", err)
	}

	log.Printf("Processed %d nodes", i.processedNodes.Load())

	// _ Second pass: process ways and create edges
	log.Println("Processing ways and creating edges...")
	if err := i.processWays(ctx, ways); err != nil {
		return fmt.Errorf("processing ways: %w", err)
	}

	log.Printf("Import complete. Processed %d ways", i.processedWays.Load())

	return nil
}

func (i *Importer) processNodes(ctx context.Context, nodes <-chan *osm.Node) {
	batch := make([]*osm.Node, 0, 1000)
	
	for node := range nodes {
		batch = append(batch, node)
		
		if len(batch) >= 1000 {
			i.insertNodeBatch(ctx, batch)
			batch = batch[:0]
		}
	}
	
	// _ Insert remaining nodes
	if len(batch) > 0 {
		i.insertNodeBatch(ctx, batch)
	}
}

func (i *Importer) insertNodeBatch(ctx context.Context, nodes []*osm.Node) {
	query := `
		INSERT INTO nodes (osm_id, location)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326))
		ON CONFLICT (osm_id) DO UPDATE SET location = EXCLUDED.location
		RETURNING id, osm_id
	`

	tx, err := i.db.Begin(ctx)
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	for _, node := range nodes {
		var dbID int64
		var osmID int64
		
		err := tx.QueryRow(ctx, query, node.ID, node.Lon, node.Lat).Scan(&dbID, &osmID)
		if err != nil {
			log.Printf("Error inserting node %d: %v", node.ID, err)
			continue
		}
		
		// _ Store mapping
		i.nodeMap.Store(osmID, dbID)
		i.processedNodes.Add(1)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error committing node batch: %v", err)
	}
}

func (i *Importer) processWays(ctx context.Context, ways []*osm.Way) error {
	wg := conc.NewWaitGroup()
	
	// _ Process ways in parallel batches
	batchSize := 100
	for start := 0; start < len(ways); start += batchSize {
		end := start + batchSize
		if end > len(ways) {
			end = len(ways)
		}
		
		batch := ways[start:end]
		wg.Go(func() {
			i.processWayBatch(ctx, batch)
		})
	}
	
	wg.Wait()
	return nil
}

func (i *Importer) processWayBatch(ctx context.Context, ways []*osm.Way) {
	for _, way := range ways {
		// _ Get node IDs for this way
		nodeIDsInterface, ok := i.wayNodes.Load(way.ID)
		if !ok {
			continue
		}
		
		nodeIDs := nodeIDsInterface.([]int64)
		
		// _ Create edges between consecutive nodes
		for j := 0; j < len(nodeIDs)-1; j++ {
			fromOSMID := nodeIDs[j]
			toOSMID := nodeIDs[j+1]
			
			// _ Get database IDs
			fromDBID, ok1 := i.nodeMap.Load(fromOSMID)
			toDBID, ok2 := i.nodeMap.Load(toOSMID)
			
			if !ok1 || !ok2 {
				continue
			}
			
			// _ Calculate edge properties
			distance := calculateDistance(way.Nodes[j], way.Nodes[j+1])
			travelTime := calculateTravelTime(distance, way)
			restrictions := extractRestrictions(way)
			
			// _ Insert edge
			query := `
				INSERT INTO edges (
					from_node_id, to_node_id, distance, travel_time,
					max_height, max_weight, truck_allowed, road_type, osm_way_id
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`
			
			_, err := i.db.Exec(ctx, query,
				fromDBID, toDBID, distance, travelTime,
				restrictions.maxHeight, restrictions.maxWeight,
				restrictions.truckAllowed, getRoadType(way), way.ID,
			)
			
			if err != nil {
				log.Printf("Error inserting edge: %v", err)
				continue
			}
			
			// _ Insert reverse edge for two-way roads
			if !isOneWay(way) {
				_, err = i.db.Exec(ctx, query,
					toDBID, fromDBID, distance, travelTime,
					restrictions.maxHeight, restrictions.maxWeight,
					restrictions.truckAllowed, getRoadType(way), way.ID,
				)
				
				if err != nil {
					log.Printf("Error inserting reverse edge: %v", err)
				}
			}
		}
		
		i.processedWays.Add(1)
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
		"motorway":      true,
		"trunk":         true,
		"primary":       true,
		"secondary":     true,
		"tertiary":      true,
		"unclassified":  true,
		"residential":   true,
		"motorway_link": true,
		"trunk_link":    true,
		"primary_link":  true,
		"secondary_link": true,
		"tertiary_link": true,
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
		fmt.Sscanf(maxHeight, "%f", &height)
		r.maxHeight = height
	}
	
	// _ Check for weight restrictions
	if maxWeight := way.Tags.Find("maxweight"); maxWeight != "" {
		// _ Parse weight (simplified - in production, handle units properly)
		// _ For now, assume tons, convert to kg
		var weight float64
		fmt.Sscanf(maxWeight, "%f", &weight)
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
		fmt.Sscanf(maxSpeed, "%f", &speed)
		return speed
	}
	return 0
}

func getDefaultSpeed(way *osm.Way) float64 {
	highway := way.Tags.Find("highway")
	
	// _ Default speeds in km/h for trucks
	speeds := map[string]float64{
		"motorway":      80,
		"trunk":         70,
		"primary":       60,
		"secondary":     50,
		"tertiary":      40,
		"unclassified":  30,
		"residential":   25,
		"motorway_link": 50,
		"trunk_link":    40,
		"primary_link":  40,
		"secondary_link": 30,
		"tertiary_link": 30,
	}
	
	if speed, ok := speeds[highway]; ok {
		return speed
	}
	
	return 30 // Default fallback
}