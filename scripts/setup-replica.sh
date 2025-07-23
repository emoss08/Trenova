#!/bin/bash
# # Copyright 2023-2025 Eric Moss
# # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

set -e

# * This script sets up a PostgreSQL replica

if [ "$POSTGRES_REPLICATION_MODE" = "slave" ]; then
    echo "Setting up PostgreSQL replica..."
    
    # * Wait for the master to be ready
    until PGPASSWORD=$POSTGRES_REPLICATION_PASSWORD psql -h "$POSTGRES_MASTER_SERVICE" -U "$POSTGRES_REPLICATION_USER" -d "$POSTGRES_DB" -c '\q'; do
        >&2 echo "Master is unavailable - sleeping"
        sleep 1
    done
    
    >&2 echo "Master is up - setting up replica"
    
    # * Stop PostgreSQL
    pg_ctl -D "$PGDATA" -m fast -w stop || true
    
    # * Clear data directory
    rm -rf ${PGDATA}/*
    
    # * Perform base backup from master
    PGPASSWORD=$POSTGRES_REPLICATION_PASSWORD pg_basebackup \
        -h $POSTGRES_MASTER_SERVICE \
        -p $POSTGRES_MASTER_PORT \
        -U $POSTGRES_REPLICATION_USER \
        -D $PGDATA \
        -Fp -Xs -P -R -v
    
    # * Create standby signal file
    touch $PGDATA/standby.signal
    
    # * Configure connection to master
    cat >> $PGDATA/postgresql.auto.conf <<EOF
primary_conninfo = 'host=$POSTGRES_MASTER_SERVICE port=$POSTGRES_MASTER_PORT user=$POSTGRES_REPLICATION_USER password=$POSTGRES_REPLICATION_PASSWORD'
hot_standby = on
EOF
    
    # * Start PostgreSQL
    pg_ctl -D "$PGDATA" -w start
    
    echo "Replica setup completed"
fi