#!/bin/bash
##
## Copyright 2023-2025 Eric Moss
## Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
## Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md##

# Quick monitoring script
PGPASSWORD=routing psql -h localhost -p 5433 -U routing -d routing -c "
SELECT 
    'Temp nodes: ' || to_char(COUNT(*), 'FM999,999,999') as status
FROM temp_nodes
UNION ALL
SELECT 
    'Main nodes: ' || to_char(COUNT(*), 'FM999,999,999')
FROM nodes
UNION ALL
SELECT 
    'Active queries: ' || COUNT(*)::text
FROM pg_stat_activity 
WHERE datname = 'routing' AND state = 'active' AND pid <> pg_backend_pid();"