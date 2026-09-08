<div align="center">
  <img src="https://raw.githubusercontent.com/emoss08/trenova/master/.github/logo.webp" alt="Trenova logo" width="160">
  <h1>Trenova: Transportation Management System for Trucking Carriers</h1>
  <p>
    Source-available trucking TMS written in Go, PostgreSQL, GraphQL, and React.<br>
    Dispatch, rating, billing, accounting, EDI, FMCSA compliance, and document automation for asset-based carriers.
  </p>
  <p>
    <a href="https://trenova.app">Website</a> ·
    <a href="https://trenova.app/features/">Features</a> ·
    <a href="https://trenova.app/pricing/">Pricing</a> ·
    <a href="https://trenova.app/self-hosted/">Self-hosting</a> ·
    <a href="https://github.com/emoss08/trenova/discussions">Discussions</a> ·
    <a href="https://discord.gg/XDBqyvrryq">Discord</a>
  </p>
  <p>
    <a href="https://github.com/emoss08/trenova/actions/workflows/test-tms.yml"><img src="https://github.com/emoss08/trenova/actions/workflows/test-tms.yml/badge.svg" alt="TMS tests"></a>
    <a href="https://github.com/emoss08/trenova/actions/workflows/test-client.yml"><img src="https://github.com/emoss08/trenova/actions/workflows/test-client.yml/badge.svg" alt="Client tests"></a>
    <a href="https://github.com/emoss08/trenova/releases"><img src="https://img.shields.io/github/v/release/emoss08/trenova?include_prereleases" alt="Latest release"></a>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26"></a>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/license-FSL--1.1--ALv2-blue" alt="License: FSL-1.1-ALv2"></a>
  </p>
</div>

> [!IMPORTANT]
> Trenova is pre-release software under active development. It is not yet suitable for production use. Follow [releases](https://github.com/emoss08/trenova/releases) or join the [waitlist](https://trenova.app/waitlist/) for updates.

## What is Trenova?

Trenova is a transportation management system (TMS) for small and mid-sized over-the-road carriers in the United States. Enterprise TMS platforms often cost more than $50,000 a year, which puts them out of reach for the carriers that make up most of the industry. Trenova aims to close that gap with a flat-priced, self-hostable TMS that covers the full order-to-cash cycle: shipment entry, dispatch, driver assignment, rating, billing, settlement, accounting, and compliance.

The source is public under the Fair Source [Functional Source License](https://fsl.software/), so carriers and integrators can read the code, audit it, and run it on their own infrastructure.

## Features

- **Dispatch and planning**: shipment lifecycle, stop sequencing, driver and equipment assignment, exception management.
- **Rating and accessorials**: formula-based rate calculation, fuel surcharge, detention and accessorial rules.
- **Billing and invoicing**: billing readiness checks, invoice generation, adjustments, and cash application.
- **Accounting**: general ledger, close workflows, and settlement / driver pay.
- **Document automation**: BOL, POD, rate confirmation, invoice, and weight-ticket extraction and templated document generation.
- **Compliance**: FMCSA-oriented compliance records, driver certification tracking, and audit trails.
- **EDI and integrations**: EDI 204 / 210 / 214 / 990 / 997 partner workflows, telematics (Samsara), PC*MILER, AS2.
- **Search and real-time**: Meilisearch-backed search and Redis-backed real-time updates fed by change data capture.
- **Identity and access**: OIDC sign-in, role-based permissions, and per-organization tenancy.

See the full capability list at [trenova.app/features](https://trenova.app/features/).

## Tech stack

| Layer | Technology |
| --- | --- |
| API | Go 1.26, [Gin](https://github.com/gin-gonic/gin), [gqlgen](https://github.com/99designs/gqlgen) GraphQL, OpenAPI |
| Database | PostgreSQL via [pgx](https://github.com/jackc/pgx), SQL-first migrations |
| Search and cache | Meilisearch, Redis (JSON and Streams) |
| Change data capture | [GTC](./services/gtc): a Go PostgreSQL logical-replication connector that projects rows into Meilisearch and Redis |
| Web client | React 19, TypeScript, Vite, GraphQL |
| Documents | Gotenberg (PDF), MinIO / S3-compatible object storage |
| Deployment | Docker images on GHCR, Caddy, Docker Compose |

## Repository layout

```text
services/tms          Core TMS API (Go)
services/gtc          PostgreSQL change-data-capture connector (Go)
services/samsara-sim  Samsara telematics simulator for local development
services/edi-partner-sim  EDI trading-partner simulator
shared/               Shared Go packages (money, geo, dispatch planner, PC*MILER, AS2, ...)
client/apps/web       Web application (React)
client/apps/dash      Admin dashboard (React)
deploy/               Dockerfiles, Caddyfile, observability stack
docs/                 Engineering guides and operations runbooks
```

## Quick start (local development)

Prerequisites: Go 1.26+, Node.js with pnpm, Docker, and [Task](https://taskfile.dev/).

```bash
git clone https://github.com/emoss08/trenova.git
cd trenova
cp .env.example .env

# Start PostgreSQL, Redis, MinIO, Meilisearch, and Gotenberg
docker compose -f docker-compose-local.yml up -d

# Run the TMS API
task tms:run

# Run the web client
cd client && pnpm install && pnpm dev
```

Run `task list` to see every available task. Engineering guides live in [`docs/engineering`](./docs/engineering) and operations runbooks in [`docs/operations-guides`](./docs/operations-guides).

## Self-hosting

Container images are published to GitHub Container Registry on every release:

- `ghcr.io/emoss08/trenova/tms`
- `ghcr.io/emoss08/trenova/client`

Deployment files (Dockerfiles, Caddyfile, observability) are in [`deploy/`](./deploy). See [trenova.app/self-hosted](https://trenova.app/self-hosted/) for the self-hosting overview. Trenova Cloud production servers use a separate private deployment repository that pins released image tags.

## Contributing

Trenova is Fair Source for transparency and community review, but external code contributions are not accepted at this time so the core team can keep the architecture and roadmap consistent. You can still help by:

- Reporting reproducible bugs through [GitHub Issues](https://github.com/emoss08/trenova/issues)
- Proposing features and asking questions in [Discussions](https://github.com/emoss08/trenova/discussions) or on [Discord](https://discord.gg/XDBqyvrryq)
- Reporting deployment and compatibility findings
- Reporting security vulnerabilities privately per [SECURITY.md](./SECURITY.md)

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## License

Trenova is licensed under the [Functional Source License, Version 1.1, Apache 2.0 Future License](./LICENSE) (FSL-1.1-ALv2). You may use, self-host, and modify Trenova, including in commercial environments. You may not offer it as a hosted service or use it to build a competing product. Each release converts to the Apache 2.0 license two years after publication.
