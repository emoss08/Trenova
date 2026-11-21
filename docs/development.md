# Local development

How to run Trenova from source on your own machine. To run a released version on a server, read
the [self-hosting guide](./self-hosting/README.md) instead.

## Prerequisites

- Go 1.27.4
- Node.js 26 (`client/.nvmrc`) with pnpm
- Docker
- [Task](https://taskfile.dev/)

## Start the stack

```bash
git clone https://github.com/emoss08/trenova.git
cd trenova
cp .env.example .env

# Start PostgreSQL, Redis, MinIO, Meilisearch, Gotenberg, and docling-serve
docker compose -f dev/compose.yml up -d

# Run the TMS API
task tms:run

# Run the web client
cd client && pnpm install && pnpm dev
```

Run `task list` to see every available task.

## Low-resource machines

If a build uses all your RAM and CPU, turn on low-resource mode. Go then compiles one package at
a time on one thread and runs its garbage collector harder, `golangci-lint` and the test runners
are capped the same way, hot reload waits 2 s after the last save, and `docker-up` starts only
PostgreSQL, Redis, MinIO and Temporal. Builds take longer, but the machine stays usable.

```bash
export LOW_RESOURCE=true               # every task from now on (add it to your shell profile)
task tms:run-watch LOW_RESOURCE=true   # or a single command
task tms:docker-up-minimal             # just the minimal containers, in any mode
```

On Windows, `setx LOW_RESOURCE true` sets it for every new terminal.

`LOW_RESOURCE_JOBS=2` allows two packages at a time with two threads each; that is a sensible
setting on a 16 GB machine. Run only the client app you are working on
(`pnpm --filter @trenova/web dev`) rather than `pnpm dev`, which starts both.

## Malware scanning

ClamAV is in the local stack behind a profile, since it reserves 3 GB of memory and may use
4 GB while it reloads its signatures. Start it, wait for it to report healthy (the first start
downloads its signatures, which takes several minutes), and point the TMS at it:

```bash
docker compose -f dev/compose.yml --profile clamav up -d clamav
docker compose -f dev/compose.yml ps clamav          # wait for "healthy"
```

```yaml
# services/tms/config/config.yaml
contentScan:
  clamav:
    address: localhost:3310
```

Restart the API and worker. An upload then reads "Checking for viruses..." once it is sent. The
EICAR test file (`X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`, saved
as a `.txt`) is refused and shows as blocked. See "Malware scanning" in the
[self-hosting guide](./self-hosting/README.md#malware-scanning).

## Tech stack

| Layer | Technology |
| --- | --- |
| API | Go 1.27, [Gin](https://github.com/gin-gonic/gin), [gqlgen](https://github.com/99designs/gqlgen) GraphQL |
| Database | PostgreSQL via [pgx](https://github.com/jackc/pgx), SQL-first migrations |
| Background work | Temporal |
| Search and cache | Meilisearch, Redis (JSON and Streams) |
| Change data capture | [GTC](https://github.com/emoss08/gtc): a Go PostgreSQL logical-replication connector that projects rows into Meilisearch and Redis (runs as the `ghcr.io/emoss08/gtc` image; routing in `deploy/gtc/sinks.yaml`) |
| Web client | React 19, TypeScript, Vite, GraphQL |
| Documents | Rust imaging worker, Gotenberg (PDF), docling-serve (document extraction), MinIO / S3-compatible object storage |
| Deployment | Docker images on GHCR, Caddy, Docker Compose |

## Repository layout

```text
services/tms              Core TMS: API, worker and CLI (Go)
services/imaging          Imaging worker: document rendering, inspection and assembly (Rust)
client/apps/web           Web application (React)
client/apps/dash          Driver portal (React)
client/packages/          Shared TypeScript packages: design system, GraphQL client, config
desktop/capture           Windows companion for scanning and printing (Rust)
shared/                   Shared Go packages (money, geo, dispatch planner, PC*MILER, AS2, ...)
i18n/                     Translation catalogs and the extraction tooling
dev/                      Local environment: compose.yml, its configs, Samsara and EDI partner simulators
deploy/                   Dockerfiles, self-host compose, Caddy, GTC routing, observability, PgBouncer example
tools/                    Tooling that is not shipped: extraction fine-tuning (Python), screenshots, load tests, release
docs/                     Engineering guides, product guide, self-hosting
```

## Further reading

- [Engineering guides](./engineering/README.md): architecture and the patterns for each layer.
- [Contributing](../CONTRIBUTING.md): issues, pull requests and the checks to run.
- [trenova-documentation](https://github.com/emoss08/trenova-documentation): operations
  runbooks, design notes, roadmaps and research.
