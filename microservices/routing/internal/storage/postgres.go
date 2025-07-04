package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/routing/internal/graph"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"
)

// PostgresStorage implements the storage interface for PostgreSQL with PostGIS
type PostgresStorage struct {
	db *pgxpool.Pool
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(ctx context.Context, dsn string) (*PostgresStorage, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// _ Optimize connection pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &PostgresStorage{db: pool}, nil
}

// LoadGraphForRegion loads the graph data for a specific region
func (s *PostgresStorage) LoadGraphForRegion(ctx context.Context, minLat, minLon, maxLat, maxLon float64) (*graph.Graph, error) {
	g := graph.NewGraph()

	// _ Load nodes within the bounding box
	nodeQuery := `
		SELECT id, ST_AsBinary(location) as location, osm_id
		FROM nodes
		WHERE location && ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography
	`

	rows, err := s.db.Query(ctx, nodeQuery, minLon, minLat, maxLon, maxLat)
	if err != nil {
		return nil, fmt.Errorf("querying nodes: %w", err)
	}
	defer rows.Close()

	nodeMap := make(map[int64]*graph.Node)

	for rows.Next() {
		var (
			id       int64
			locBytes []byte
			osmID    *int64
		)

		if err := rows.Scan(&id, &locBytes, &osmID); err != nil {
			return nil, fmt.Errorf("scanning node: %w", err)
		}

		// _ Decode the location
		geom, err := wkb.Unmarshal(locBytes)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling location: %w", err)
		}

		point, ok := geom.(orb.Point)
		if !ok {
			return nil, fmt.Errorf("location is not a point")
		}

		node := &graph.Node{
			ID:       id,
			Location: point,
			Edges:    []*graph.Edge{},
		}

		nodeMap[id] = node
		g.AddNode(node)
	}

	// _ Load edges connected to these nodes
	edgeQuery := `
		SELECT id, from_node_id, to_node_id, distance, travel_time,
		       max_height, max_weight, truck_allowed
		FROM edges
		WHERE from_node_id = ANY($1::bigint[])
	`

	nodeIDs := make([]int64, 0, len(nodeMap))
	for id := range nodeMap {
		nodeIDs = append(nodeIDs, id)
	}

	rows, err = s.db.Query(ctx, edgeQuery, nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("querying edges: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			edge        graph.Edge
			fromNodeID  int64
			toNodeID    int64
		)

		if err := rows.Scan(
			&edge.ID,
			&fromNodeID,
			&toNodeID,
			&edge.Distance,
			&edge.TravelTime,
			&edge.MaxHeight,
			&edge.MaxWeight,
			&edge.TruckAllowed,
		); err != nil {
			return nil, fmt.Errorf("scanning edge: %w", err)
		}

		// _ Link nodes to edges
		if fromNode, ok := nodeMap[fromNodeID]; ok {
			if toNode, ok := nodeMap[toNodeID]; ok {
				edge.From = fromNode
				edge.To = toNode
				g.AddEdge(&edge)
			}
		}
	}

	return g, nil
}

// FindNearestNode finds the nearest node to a given coordinate
func (s *PostgresStorage) FindNearestNode(ctx context.Context, lon, lat float64) (int64, error) {
	query := `
		SELECT id
		FROM nodes
		ORDER BY location <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)
		LIMIT 1
	`

	var nodeID int64
	err := s.db.QueryRow(ctx, query, lon, lat).Scan(&nodeID)
	if err != nil {
		return 0, fmt.Errorf("finding nearest node: %w", err)
	}

	return nodeID, nil
}

// GetNodeIDForZip returns the node ID associated with a zip code
func (s *PostgresStorage) GetNodeIDForZip(ctx context.Context, zipCode string) (int64, error) {
	query := `SELECT node_id FROM zip_nodes WHERE zip_code = $1`

	var nodeID int64
	err := s.db.QueryRow(ctx, query, zipCode).Scan(&nodeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("zip code not found: %s", zipCode)
		}
		return 0, fmt.Errorf("querying zip node: %w", err)
	}

	return nodeID, nil
}

// SaveCachedRoute saves a calculated route to the cache
func (s *PostgresStorage) SaveCachedRoute(ctx context.Context, originZip, destZip string, distance, travelTime float64) error {
	query := `
		INSERT INTO cached_routes (origin_zip, dest_zip, distance, travel_time)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (origin_zip, dest_zip) 
		DO UPDATE SET 
			distance = EXCLUDED.distance,
			travel_time = EXCLUDED.travel_time,
			calculated_at = CURRENT_TIMESTAMP,
			expires_at = CURRENT_TIMESTAMP + INTERVAL '48 hours'
	`

	_, err := s.db.Exec(ctx, query, originZip, destZip, distance, travelTime)
	if err != nil {
		return fmt.Errorf("saving cached route: %w", err)
	}

	return nil
}

// GetCachedRoute retrieves a cached route if it exists and hasn't expired
func (s *PostgresStorage) GetCachedRoute(ctx context.Context, originZip, destZip string) (distance, travelTime float64, found bool, err error) {
	query := `
		SELECT distance, travel_time
		FROM cached_routes
		WHERE origin_zip = $1 AND dest_zip = $2 AND expires_at > CURRENT_TIMESTAMP
	`

	err = s.db.QueryRow(ctx, query, originZip, destZip).Scan(&distance, &travelTime)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, 0, false, nil
		}
		return 0, 0, false, fmt.Errorf("querying cached route: %w", err)
	}

	return distance, travelTime, true, nil
}

// Close closes the database connection
func (s *PostgresStorage) Close() {
	s.db.Close()
}