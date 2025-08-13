# EDI Service

The EDI parser can now run as a standalone service with database persistence, dependency injection, and REST API endpoints.

## Quick Start

1. **Setup Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your database credentials
   ```

2. **Run the Service**
   ```bash
   make service
   # or for development with auto-reload
   make dev
   ```

3. **Process EDI Documents**
   ```bash
   curl -X POST http://localhost:8080/process?partner_id=PARTNER123 \
     -H "Content-Type: text/plain" \
     --data-binary @testdata/204/sample1.edi
   ```

## Architecture

### Hexagonal/Clean Architecture
- **Domain Layer** (`internal/core/domain/`): Business entities
- **Ports Layer** (`internal/core/ports/`): Repository interfaces  
- **Services Layer** (`internal/core/services/`): Business logic
- **Infrastructure Layer** (`internal/infrastructure/`): Database, logging
- **Bootstrap** (`internal/bootstrap/`): Dependency injection with uber-go/fx

### Database Models
- `edi_documents`: Main EDI document records
- `edi_transactions`: Individual transactions (e.g., 204s)
- `edi_shipments`: Parsed shipment data
- `edi_stops`: Shipment stop details
- `edi_acknowledgments`: 997/999 acknowledgments
- `edi_partner_profiles`: Partner-specific configurations

### Dependency Injection
Uses uber-go/fx for clean dependency management:
- Automatic lifecycle management
- Database connection pooling
- Graceful shutdown
- Modular architecture

## API Endpoints

### Health Check
```
GET /health
```

### Process EDI Document
```
POST /process?partner_id={partner_id}
Content-Type: text/plain
Body: [EDI content]
```

### Profile Management

#### List Profiles
```
GET /profiles
Response: {"profiles": [{"partner_id": "...", "partner_name": "...", "active": "true"}]}
```

#### Import Profile
```
POST /profiles
Content-Type: application/json
Body: [Full PartnerProfile JSON from testdata/profiles/]

Response: {"partner_id": "...", "partner_name": "..."}
```

Example profile import:
```bash
curl -X POST http://localhost:8080/profiles \
  -H "Content-Type: application/json" \
  --data-binary @testdata/profiles/meritor-4010.json
```

## Configuration

Environment variables (see `.env.example`):
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: PostgreSQL connection
- `SERVICE_PORT`: HTTP server port (default: 8080)
- `ENVIRONMENT`: development/production
- `DB_MAX_CONNECTIONS`: Connection pool size
- `DB_SSL_MODE`: SSL mode for database connection

## Development

### Running Tests
```bash
make test
```

### Building Binaries
```bash
make build
```

### Database Migrations
Migrations run automatically on service startup. Tables are created if they don't exist.

## Integration with Main TMS

The service can be deployed separately from the main Trenova TMS and accessed via:
1. REST API for EDI document processing
2. Direct database access for reporting
3. gRPC service (existing configuration service)

## Production Deployment

1. Set `ENVIRONMENT=production` in environment
2. Configure proper database credentials
3. Enable SSL for database connections
4. Use connection pooling via PGBouncer if needed
5. Deploy behind a reverse proxy (Caddy/Nginx)
6. Monitor with Prometheus/OpenTelemetry (hooks available)