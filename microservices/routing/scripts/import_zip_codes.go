package main

import (
	"context"
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run import_zip_codes.go <csv-file> <database-url>")
	}

	csvFile := os.Args[1]
	dbURL := os.Args[2]

	// _ Open CSV file
	file, err := os.Open(csvFile)
	if err != nil {
		log.Fatalf("Error opening CSV: %v", err)
	}
	defer file.Close()

	// _ Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer pool.Close()

	// _ Read CSV
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.LazyQuotes = true

	// _ Skip header
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Error reading header: %v", err)
	}
	log.Printf("CSV Headers: %v", header)

	// _ Find column indices
	var zipIdx, stateCodeIdx, cityIdx, coordIdx int
	for i, col := range header {
		switch {
		case strings.Contains(col, "Zip Code"):
			zipIdx = i
		case col == "Official USPS State Code":
			stateCodeIdx = i
		case col == "Official State Name":
			// Not used, but keeping for reference
		case strings.Contains(col, "Official USPS city"):
			cityIdx = i
		case strings.Contains(col, "Geo Point"):
			coordIdx = i
		}
	}

	// _ Process records
	batch := &pgx.Batch{}
	count := 0
	caCount := 0

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		if len(record) <= coordIdx {
			continue
		}

		zipCode := strings.TrimSpace(record[zipIdx])
		stateCode := strings.TrimSpace(record[stateCodeIdx])
		city := strings.TrimSpace(record[cityIdx])
		coords := strings.TrimSpace(record[coordIdx])

		// _ Only process California zips for now
		if stateCode != "CA" {
			continue
		}

		// _ Parse coordinates
		parts := strings.Split(coords, ",")
		if len(parts) != 2 {
			continue
		}

		lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			continue
		}

		lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			continue
		}

		// _ Add to batch - find nearest node and insert
		batch.Queue(`
			INSERT INTO zip_nodes (zip_code, node_id, centroid, state, city)
			SELECT $1, n.id, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, $4, $5
			FROM nodes n
			ORDER BY n.location <-> ST_SetSRID(ST_MakePoint($3, $2), 4326)
			LIMIT 1
			ON CONFLICT (zip_code) DO UPDATE SET
				node_id = EXCLUDED.node_id,
				centroid = EXCLUDED.centroid,
				state = EXCLUDED.state,
				city = EXCLUDED.city
		`, zipCode, lat, lon, stateCode, city)

		count++
		caCount++

		// _ Execute batch every 100 records
		if batch.Len() >= 100 {
			br := pool.SendBatch(ctx, batch)
			for i := 0; i < batch.Len(); i++ {
				_, err := br.Exec()
				if err != nil {
					log.Printf("Error inserting zip: %v", err)
				}
			}
			br.Close()
			batch = &pgx.Batch{}
			log.Printf("Processed %d California zip codes...", caCount)
		}
	}

	// _ Execute remaining batch
	if batch.Len() > 0 {
		br := pool.SendBatch(ctx, batch)
		for i := 0; i < batch.Len(); i++ {
			_, err := br.Exec()
			if err != nil {
				log.Printf("Error inserting zip: %v", err)
			}
		}
		br.Close()
	}

	log.Printf("Import complete! Processed %d total records, %d California zip codes", count, caCount)

	// _ Verify results
	var finalCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM zip_nodes WHERE state = 'CA'").Scan(&finalCount)
	if err == nil {
		log.Printf("Total California zip codes in database: %d", finalCount)
	}
}
