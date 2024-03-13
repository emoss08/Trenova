# Trenova: Atlas and Entgo Command Guide

This document provides a concise guide on how to run Atlas and Entgo commands, essential for fellow developers working on database migrations and entity management.

## Atlas Commands

Atlas is a powerful tool for managing database schemas and migrations. Here's how to utilize its primary commands:

### Migration Commands

1. **Create a Migration Diff**:
   Generate a new migration file by comparing your current schema to the target schema.
   ```bash
   atlas migrate diff migration_name \
           --dir "file://ent/migrate/migrations" \
           --to "ent://ent/schema" \
           --dev-url "docker://postgres/15/test?search_path=public"
   ```
   *Note: Replace `migration_name` with the name you wish to give your migration.*

2. **Generate Migration Hash**:
   Create a hash for your migration files.
   ```bash
   atlas migrate hash --dir file://ent/migrate/migrations
   ```

3. **Apply Migration**:
   Execute the migration files to update your database schema.
   ```bash
   atlas migrate apply \
     --dir "file://ent/migrate/migrations" \
     --url "postgresql://postgres:postgres@localhost:5432/trenova_go_db?sslmode=disable"
   ```

4. **Check Migration Status**:
   View the current status of your migrations.
   ```bash
   atlas migrate status \
     --dir "file://ent/migrate/migrations" \
     --url "postgresql://postgres:postgres@localhost:5432/trenova_go_db?sslmode=disable"
   ```

5. **Inspect the Schema**:
   Analyze and output the details of your database schema.
   ```bash
   atlas schema inspect \
           -u "ent://ent/schema" \
           --dev-url "sqlite://file?mode=memory&_fk=1" \
           -w
   ```

### Entity Framework (Entgo) Commands

Entgo is an entity framework for Go, simplifying the process of working with entities and schemas.

1. **Auto Generate Go Code**:
   Automatically generate Go code based on your schema definitions.
   ```bash
   go generate ./ent
   ```

2. **Create a New Entity**:
   Define a new entity in your schema.
   ```bash
   go run -mod=mod entgo.io/ent/cmd/ent new NewEntity
   ```
   *Note: Replace `NewEntity` with the name of the entity you're creating.*

3. **Describe the Schema**:
   Output a textual representation of your schema structure.
   ```bash
   go run -mod=mod entgo.io/ent/cmd/ent describe ./ent/schema
   ```

This guide should assist developers in efficiently managing database schemas and entities using Atlas and Entgo commands.