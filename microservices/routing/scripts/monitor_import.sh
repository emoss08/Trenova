#!/bin/bash
# # Copyright 2023-2025 Eric Moss
# # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md


# Monitor OSM import progress
# Usage: ./scripts/monitor_import.sh

DB_URL="postgres://routing:routing@localhost:5433/routing?sslmode=disable"

echo "Monitoring OSM import progress..."
echo "Press Ctrl+C to stop"
echo ""

while true; do
    clear
    echo "=== OSM Import Progress ==="
    echo "Time: $(date)"
    echo ""
    
    # Node counts
    echo "NODES:"
    psql "$DB_URL" -t -c "
        SELECT 
            'Temp nodes: ' || COUNT(*) FROM temp_nodes
        UNION ALL
        SELECT 
            'Final nodes: ' || COUNT(*) FROM nodes
    " 2>/dev/null || echo "  (waiting for data...)"
    
    echo ""
    echo "EDGES:"
    psql "$DB_URL" -t -c "
        SELECT 
            'Temp edges: ' || COUNT(*) FROM temp_edges
        UNION ALL
        SELECT 
            'Final edges: ' || COUNT(*) FROM edges
    " 2>/dev/null || echo "  (waiting for data...)"
    
    echo ""
    echo "TABLE SIZES:"
    psql "$DB_URL" -t -c "
        SELECT 
            tablename || ': ' || pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
        FROM pg_tables 
        WHERE tablename IN ('nodes', 'edges', 'temp_nodes', 'temp_edges')
        ORDER BY tablename
    " 2>/dev/null || echo "  (waiting for data...)"
    
    sleep 5
done