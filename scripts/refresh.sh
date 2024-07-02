#!/bin/bash

set -e  # Exit immediately if a command exits with a non-zero status.
set -o pipefail  # Catch errors in pipelines

# Define variables
CONTAINER_ID="14604943f66fc0eb7e795dc21bfa57933611b527e41b2b3a8d6734c2b141c4c0"
CONTAINER_NAME="trenova-db"
MIGRATIONS_DIR="./ent/migrate/migrations/*"
MIGRATIONS_BASE_DIR="./ent/migrate/migrations"
MIGRATIONS_INTERNAL_DIR="file://ent/migrate/migrations"
DB_URL="postgresql://postgres:postgres@localhost:5432/trenova_go_db?sslmode=disable"
DEV_URL="docker://postgres/15/test?search_path=public"
LOG_FILE="./migration.log"

# Function to handle errors
on_error() {
    echo "Error occurred. Exiting. See $LOG_FILE for details."
    exit 1
}

# Trap errors and call the error handler
trap 'on_error' ERR

# Start logging
exec > >(tee -i $LOG_FILE)
exec 2>&1

# Print start message
echo "Starting migration script at $(date)"

# Validate Docker container existence
if ! docker ps -q --filter "id=$CONTAINER_ID" > /dev/null; then
    echo "Error: Docker container with ID $CONTAINER_ID does not exist or is not running."
    exit 1
fi

# Validate the existence of the migrations directory
ABS_MIGRATIONS_BASE_DIR=$(realpath "$MIGRATIONS_BASE_DIR")
if [ ! -d "$ABS_MIGRATIONS_BASE_DIR" ]; then
    echo "Error: Migrations directory $ABS_MIGRATIONS_BASE_DIR does not exist."
    exit 1
fi

# Remove all files in the migrations folder but keep the folder itself
echo "Cleaning up the migrations folder..."
rm -rf $MIGRATIONS_DIR || { echo "Failed to clean up the migrations folder"; exit 1; }

# Drop the database schema using Docker command
# This will delete all tables and their data
# Useful if you want to start from scratch
# Comment out the next line if you want to keep the existing data
echo "Dropping the database schema..."
docker exec -i $CONTAINER_ID psql -U postgres -d trenova_go_db -c "
-- Drop the public schema if it exists
DROP SCHEMA IF EXISTS public CASCADE;

-- Create the public schema
CREATE SCHEMA public;

-- Drop the atlas related schemas
DROP SCHEMA IF EXISTS atlas_schema_revisions CASCADE;

GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO public;
" || { echo "Failed to drop the database schema"; exit 1; }

# Create new migrations based on the current schema
echo "Creating new migrations..."
atlas migrate diff initial_commit \
    --dir $MIGRATIONS_INTERNAL_DIR \
    --to "ent://ent/schema" \
    --dev-url $DEV_URL || { echo "Failed to create new migrations"; exit 1; }

# Apply the new migrations to the database
echo "Applying new migrations..."
atlas migrate apply \
    --dir $MIGRATIONS_INTERNAL_DIR \
    --url $DB_URL || { echo "Failed to apply new migrations"; exit 1; }

# Seed the database with initial data
echo "Seeding the database..."
go run main.go seeder || { echo "Failed to seed the database"; exit 1; }

# Print completion message
echo "Migration and seeding process completed at $(date)"
