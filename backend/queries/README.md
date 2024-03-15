## Overview
This README provides an overview of the SQL queries stored in the `queries` folder. Each query serves a specific purpose in analyzing the performance of PostgreSQL databases. They are focused on extracting insights related to CPU usage, execution time, frequency of queries, index usage, table size, and more.

### 1. CPU Queries (`cpu_queries.sql`)
Analyzes CPU usage of different queries in the PostgreSQL database. Key features:
- User and Database Identification
- Database Name
- Execution Time (Total and Average)
- CPU Usage Percentage
- Query Sample

### 2. Longest Queries (`longest_queries.sql`)
Identifies queries with the longest execution times. Key features:
- Execution Duration in Minutes
- Average Execution Time in Milliseconds
- Number of Calls
- Full Query Text

### 3. Often Queries (`often_queries.sql`)
Finds frequently executed queries within a short time frame. Key features:
- Database Name
- Query Text Snippet
- Execution Frequency (Runs per Second/Minute)

### 4. Index Usage Analysis (`index_usage.sql`)
Examines the effectiveness of index usage. Key features:
- Schema and Relation Name
- Index Name and Scan Counts
- Tuple Read and Fetch Counts

```sql
SELECT
    "pg_stat_user_indexes".schemaname,
    relname,
    indexrelname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM
    pg_stat_user_indexes
JOIN
    pg_indexes ON pg_stat_user_indexes.indexrelname = pg_indexes.indexname
WHERE
    "pg_stat_user_indexes".schemaname NOT IN ('pg_catalog', 'information_schema');
```

### 5. Table Size Information (`table_size.sql`)
Provides insight into the size of tables in the database. Key features:
- Table Schema and Name
- Size on Disk

```sql
SELECT
    table_schema,
    table_name,
    pg_size_pretty(pg_total_relation_size(table_schema || '.' || table_name)) AS size
FROM
    information_schema.tables
ORDER BY
    pg_total_relation_size(table_schema || '.' || table_name) DESC;
```

### 6. Dead Tuples Analysis (`dead_tuples.sql`)
Highlights tables with dead tuples, indicating potential need for vacuuming. Key features:
- Schema and Table Name
- Number of Dead Tuples
- Last Vacuum and Autovacuum Information

```sql
SELECT
    schemaname,
    relname,
    n_dead_tup,
    last_vacuum,
    last_autovacuum
FROM
    pg_stat_user_tables
WHERE
    n_dead_tup > 0;
```

## Conclusion
The queries in this folder are vital for database administrators and developers for optimizing database operations, identifying performance bottlenecks, and maintaining efficient database management. Regular monitoring and analysis using these queries are key to the health and performance of PostgreSQL databases.