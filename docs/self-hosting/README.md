# Self-hosting Trenova

Trenova runs on one Linux host with Docker Compose. The stack in
[`deploy/selfhost`](../../deploy/selfhost) pulls the released images from GitHub Container
Registry and runs everything they need; Caddy terminates TLS with a Let's Encrypt certificate
and is the only thing listening on the network.

Trenova Cloud, the hosted service, runs the same images. Self-hosting is for operators who
want the data on their own hardware and will look after it: backups, upgrades and the host
itself are yours.

> Trenova is pre-release software. Pin a release, back up before every upgrade, and read the
> release notes before you move to a new version.

## Contents

- [What runs](#what-runs)
- [Requirements](#requirements)
- [Install](#install)
- [DNS and TLS](#dns-and-tls)
- [First sign-in](#first-sign-in)
- [Configuration](#configuration)
- [Upgrades](#upgrades)
- [Backups and restore](#backups-and-restore)
- [Operating the stack](#operating-the-stack)
- [Hardening](#hardening)
- [Troubleshooting](#troubleshooting)
- [Uninstall](#uninstall)

## What runs

| Service | Image | Role |
|---|---|---|
| `client` | `ghcr.io/emoss08/trenova/client` | Caddy: HTTPS, the web app at `/`, the driver portal at `/dash/`, the API and object storage behind them |
| `tms-api` | `ghcr.io/emoss08/trenova/tms` | REST and GraphQL API, live updates (server-sent events) |
| `tms-worker` | `ghcr.io/emoss08/trenova/tms` | Background jobs and every AI feature, on Temporal |
| `tms-migrate` | `ghcr.io/emoss08/trenova/tms` | One-shot: database migrations and base seeds, on every start |
| `db-bootstrap` | `ghcr.io/emoss08/trenova/postgres` | One-shot: change-data-capture publication, first-run account hardening |
| `postgres` | `ghcr.io/emoss08/trenova/postgres` | PostgreSQL 18 with PostGIS, pg_cron and pgvector |
| `redis` | `redis:8` | Sessions, rate limits, caches, the live-update streams |
| `minio` | `pgsty/silo` (MinIO) | Documents and other uploaded files |
| `meilisearch` | `getmeili/meilisearch` | Global search |
| `temporal` | `temporalio/auto-setup` | Durable workflows; state in its own two PostgreSQL databases |
| `gotenberg` | `gotenberg/gotenberg` | HTML-to-PDF rendering, locked down and unreachable from outside |
| `gtc` | `ghcr.io/emoss08/gtc` | Change data capture: PostgreSQL to Meilisearch and Redis |

The `tms-api`, `tms-worker` and `tms-migrate` services are one image with different commands.
Only Caddy publishes ports (80 and 443, TCP and UDP). The databases, queues, search, Temporal,
Gotenberg and GTC sit on an internal Docker network with no route off the host; the API and
worker get a separate network for outbound calls (AI providers, email, accounting
integrations, release checks).

## Requirements

- A 64-bit Linux host (x86_64; the TMS and postgres images are built for `linux/amd64`).
- **4 vCPU, 8 GB RAM, 50 GB SSD** for a small fleet. Give PostgreSQL more memory as data grows
  (see [Tuning](#tuning)).
- Docker Engine 24 or newer with the Compose plugin v2.24 or newer (`docker compose version`).
- `git` and `openssl` on the host.
- For a public install: a hostname for Trenova and one for its file storage (for example
  `trenova.example.com` and `storage.trenova.example.com`), both pointing at the host, and
  inbound TCP 80 and 443 open. Let's Encrypt validates over port 80.

## Install

1. **Get the deployment files** for the release you will run. The tag must match the
   `TRENOVA_VERSION` you deploy, so the compose file, configuration and change-data-capture
   routing are the ones that release expects.

   ```bash
   git clone --depth 1 --branch v0.9.17 https://github.com/emoss08/trenova.git
   cd trenova/deploy/selfhost
   ```

2. **Create `.env`** with generated secrets:

   ```bash
   ./scripts/init-env.sh trenova.example.com
   ```

   This copies [`.env.example`](../../deploy/selfhost/.env.example) to `.env` (mode 600),
   generates every password and key, and sets `DOMAIN=trenova.example.com` and
   `STORAGE_DOMAIN=storage.trenova.example.com`. Run it with no argument for a trial on
   `https://localhost`. Review the file afterwards; the comments explain every value.

   **Back up `.env` now, somewhere other than this host.** `TRENOVA_SECURITY_ENCRYPTION_KEY`
   encrypts sensitive fields in the database; lose it and that data is gone.

3. **Point DNS** for both hostnames at the host (see [DNS and TLS](#dns-and-tls)).

4. **Start it:**

   ```bash
   docker compose pull
   docker compose up -d
   docker compose ps
   ```

   The first start takes a few minutes: PostgreSQL initializes, Temporal creates its schema,
   `tms-migrate` runs every migration and the base seeds, and Caddy obtains certificates.
   `tms-migrate` and `db-bootstrap` show as `exited (0)` when they have finished; every other
   service should become `healthy` or `running`.

5. **Sign in** at `https://trenova.example.com` (see [First sign-in](#first-sign-in)).

### A local trial

With `DOMAIN=localhost` and `STORAGE_DOMAIN=storage.localhost` (the defaults), Caddy serves
`https://localhost` with a certificate from its own local certificate authority. Your browser
will warn about it; either accept the warning or trust Caddy's root certificate:

```bash
docker compose cp client:/data/caddy/pki/authorities/local/root.crt ./caddy-root.crt
```

and import `caddy-root.crt` into your operating system or browser trust store. Accept it for
`https://storage.localhost` too, or uploads and document previews will fail.

## DNS and TLS

Create two DNS records pointing at the host's public address:

| Record | Example | Used for |
|---|---|---|
| `DOMAIN` | `trenova.example.com` | The web app, the driver portal (`/dash/`), the API (`/api/`, `/graphql`) |
| `STORAGE_DOMAIN` | `storage.trenova.example.com` | Presigned uploads and downloads to the bundled MinIO |

Storage needs a hostname of its own because presigned URLs are signed for a host and MinIO
cannot be served under a path.

Caddy requests a Let's Encrypt certificate for each name on first start and renews it
automatically; certificates are kept in the `caddy_data` volume. For notices about expiring
certificates, set `CADDY_GLOBAL_OPTIONS=email ops@example.com` in `.env`.

**Private networks.** When Let's Encrypt cannot reach the host (an intranet name, a VPN-only
install), set `CADDY_GLOBAL_OPTIONS=local_certs`. Caddy then issues both certificates from its
local CA; distribute its root certificate (above) to every machine that uses Trenova.

**Another proxy in front.** Caddy expects to be the edge. If a load balancer or another
reverse proxy must sit in front, have it pass TCP 443 through (TLS passthrough), or terminate
TLS there and forward to Caddy on 443 with the original `Host` header; the session cookie is
`__Host-` prefixed and `Secure`, so the browser must always see HTTPS.

## First sign-in

The base seeds, which `tms-migrate` applies on the first start, create:

- the reference data every install needs (US states, DOT hazardous materials, document types,
  IFTA jurisdictions, roles and permissions, document templates);
- two organizations, **Trenova Logistics** and **Trenova Transportation**, in one business
  unit;
- three administrators, all seeded with the password `admin123!`:

| Username | Email | After the first start |
|---|---|---|
| `admin` | `admin@trenova.app` | Active; **must choose a new password at first sign-in** |
| `admin-logistics` | `admin.logistics@trenova.app` | Locked |
| `admin-transport` | `admin.transport@trenova.app` | Locked |

Because the seeded password is public, `db-bootstrap` secures these accounts once, right
after the first seeding: `admin` is made to change its password at its first sign-in, and the
two organization administrators are locked until an administrator gives them a password.
It never touches them again.

Then, straight away:

1. Sign in as `admin` / `admin123!` and choose a strong password. Do this before the hostname
   is public if you can; until you do, anyone who reaches the sign-in page can try the
   seeded password.
2. Change `admin`'s email address to one you control (**Users**, `/admin/users`), so password
   resets reach you.
3. Rename the organizations to your company (**Organization settings**,
   `/admin/organization-settings`). Most carriers use one organization; you can leave the
   second unused.
4. Invite your users, and give `admin-logistics` / `admin-transport` a password only if you
   want them.
5. Set up outgoing email (**Email profiles**, `/organization/email-profiles`, with a Resend or
   Postmark account): password resets, invitations and scheduled reports are sent through the
   provider an organization configures.

## Configuration

Configuration lives in two files next to `compose.yml`:

| File | Holds |
|---|---|
| `.env` | Image versions, hostnames, every secret, optional integration keys |
| `config/config.yaml` | Everything else: timeouts, rate limits, AI, rendering, logging |

`compose.yml` turns `.env` into `TRENOVA_<SECTION>_<KEY>` environment variables, which
override `config/config.yaml` (for example `TRENOVA_DATABASE_PASSWORD` overrides
`database.password`). Values in `config/config.yaml` marked "from .env" are placeholders; a
placeholder secret that reached production is refused at startup.

Trenova reads its configuration strictly: an unknown key fails startup and names the key. The
full reference, with every key and its default, is
[`services/tms/config/config.example.yaml`](../../services/tms/config/config.example.yaml).
Edit `config/config.yaml`, then apply it with `docker compose up -d` (which recreates the
services whose configuration changed) or `docker compose restart tms-api tms-worker`.

Settings you are likely to change:

| Setting | Where |
|---|---|
| Turn every AI feature off | `ai.enabled: false` in `config/config.yaml`. With it on, nothing reaches a model vendor until an administrator adds a provider under **AI control** (`/admin/agent-control`). |
| Models served on your own network (Ollama, vLLM) | `ai.privateNetworkProviders` (on by default here) |
| Driver portal web push | `TRENOVA_PUSH_VAPIDPUBLICKEY`, `TRENOVA_PUSH_VAPIDPRIVATEKEY`, `TRENOVA_PUSH_SUBJECT` in `.env` |
| One-click QuickBooks or Xero for every organization | `TRENOVA_QUICKBOOKS_*`, `TRENOVA_XERO_*` in `.env`; register the redirect URI `https://DOMAIN/admin/integrations/quickbooks/callback` (or `/xero/callback`) |
| Rate limits | `security.rateLimit` |
| External S3-compatible storage | `TRENOVA_STORAGE_*` and `STORAGE_ORIGIN` in `.env` (see the comments there) |
| Release notices in the app | `update` |

### Tuning

[`postgresql.conf`](../../deploy/selfhost/postgresql.conf) is sized for an 8 GB host. On a
larger one, raise `shared_buffers` to about a quarter of RAM and `effective_cache_size` to
about half, then `docker compose restart postgres`. Redis is capped at 1 GB
(`REDIS_MAXMEMORY` in `.env`).

## Upgrades

A Trenova release is a set of images tagged with its version. To upgrade:

1. Read the release notes on
   [GitHub releases](https://github.com/emoss08/trenova/releases), especially for changes to
   configuration keys or to the deployment files.
2. [Back up](#backups-and-restore).
3. Move the deployment files to the new release and set the version:

   ```bash
   git fetch --depth 1 origin tag v0.9.18
   git checkout v0.9.18
   # in .env: TRENOVA_VERSION=0.9.18, and TRENOVA_POSTGRES_VERSION if the release names one
   ```

   `git checkout` keeps `.env` (it is not tracked). If you edited `config/config.yaml`, git
   refuses to overwrite it; merge your changes into the new version by hand
   (`git stash`, `git checkout`, `git stash pop`).
4. Pull and restart:

   ```bash
   docker compose pull
   docker compose up -d
   ```

   `tms-migrate` runs the new migrations and any new base seeds before the API and worker
   start on the new version. If a migration fails, the API and worker are not started; read
   `docker compose logs tms-migrate`.

`docker compose exec tms-api trenova update check` reports whether a newer release exists, and
administrators see the same notice in the app. `trenova update apply` performs steps 3 and 4
from a host where the `trenova` binary is installed in this directory; the steps above do the
same with nothing but Docker.

Pin exact versions. Never deploy the `latest` tag: an unattended restart would upgrade the
database schema under you.

**Rolling back** the images is only safe when the newer release did not migrate the
database. Otherwise restore the backup you took before upgrading.

## Backups and restore

What to keep, from most to least important:

| Data | Where | How |
|---|---|---|
| Secrets | `.env` | Copy it off the host whenever it changes. Without `TRENOVA_SECURITY_ENCRYPTION_KEY` the encrypted fields of a database backup cannot be read. |
| Trenova's database | `postgres` service | `pg_dump` (below), daily |
| Uploaded files | `minio_data` volume | Mirror the bucket (below), daily |
| Your configuration | `config/config.yaml` | Keep it in version control or with the backups |
| Workflow history | `temporal`, `temporal_visibility` databases | Optional: in-flight workflows resume from it after a restore |

Meilisearch and the Redis caches are rebuilt from PostgreSQL by GTC and need no backup.

**Database.** A custom-format dump, consistent while Trenova runs:

```bash
docker compose exec -T postgres sh -c 'pg_dump -Fc -U "$POSTGRES_USER" "$POSTGRES_DB"' \
  > "trenova-$(date +%Y%m%d-%H%M).dump"
```

**Files.** Archive MinIO's volume while MinIO is stopped, so the archive is consistent
(uploads fail for the minute this takes):

```bash
docker compose stop minio
docker run --rm -v trenova_minio_data:/data:ro -v "$PWD:/backup" alpine \
  tar czf "/backup/trenova-files-$(date +%Y%m%d-%H%M).tar.gz" -C /data .
docker compose start minio
```

(The volume is `<COMPOSE_PROJECT_NAME>_minio_data`; `docker volume ls` lists them.) If you
cannot stop uploads, mirror the bucket with any S3 tool instead, against
`https://STORAGE_DOMAIN` with the `TRENOVA_STORAGE_ACCESSKEY` / `TRENOVA_STORAGE_SECRETKEY`
credentials.

Keep copies off the host, and test a restore now and then.

**Restore** onto a fresh install (same `.env`, empty volumes):

```bash
docker compose up -d postgres
docker compose exec -T postgres sh -c \
  'pg_restore --clean --if-exists --no-owner -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < trenova-YYYYMMDD-HHMM.dump
docker compose create minio
docker run --rm -v trenova_minio_data:/data -v "$PWD:/backup" alpine \
  tar xzf /backup/trenova-files-YYYYMMDD-HHMM.tar.gz -C /data
docker compose up -d
```

GTC creates a new replication slot on start and backfills the search indexes from the restored
database.

## Operating the stack

```bash
docker compose ps                          # state and health of every service
docker compose logs -f tms-api tms-worker  # application logs (JSON)
docker compose logs tms-migrate            # what the last migration run did
docker compose exec tms-api trenova db status       # applied migrations and seeds
docker compose restart tms-api tms-worker  # after a configuration change
```

Logs are rotated by Docker (5 files of 25 MB per service).

The **GTC dashboard** (backfills, dead letters) and the **Temporal UI** are deliberately not
exposed. [`deploy/Caddyfile.admin.example`](../../deploy/Caddyfile.admin.example) shows an
admin-only Caddy bound to `127.0.0.1` for use over an SSH tunnel or Tailscale.

## Hardening

The stack is secure by default for a single-operator host: TLS everywhere, secrets generated
per install, every backing service on an internal network, Gotenberg sandboxed, the seeded
accounts secured, and nothing published except Caddy. Beyond that:

- **Firewall.** Allow inbound 22 (or your SSH port), 80 and 443 only. Docker publishes ports
  around `ufw`/`firewalld` rules, which is why nothing but Caddy publishes one; keep it that
  way when you add services.
- **Host.** Keep the OS and Docker patched; enable unattended security updates.
- **Tenant isolation in PostgreSQL.** This stack runs with `database.rls.mode: "off"`: the
  application connects as the database owner, and tenant isolation is enforced by the
  application. Trenova can also have PostgreSQL itself refuse rows of another organization
  (row-level security), which needs separate application, system and migrator database roles
  and a scope-signing key. Follow "Rolling out" in
  [docs/engineering/row-level-security.md](../engineering/row-level-security.md): it is
  `trenova db rls generate-key`, setting `database.migrator` to the current owner role,
  `trenova db rls provision-roles`, a period in `observe`, then `enforce` with
  `database.user` switched to the new application role. GTC keeps connecting as the owner.
- **Encryption key.** The local key manager keeps `TRENOVA_SECURITY_ENCRYPTION_KEY` in `.env`.
  Treat `.env` like the database itself.

## Troubleshooting

**`tms-migrate` exited with an error.** `docker compose logs tms-migrate`. A configuration
error names the key at fault (an unknown key, a missing secret, a placeholder value). A
database error usually means PostgreSQL was not ready or the database password in `.env`
changed after the volume was created; PostgreSQL keeps the password it was initialized with.

**`validation failed: encryption key contains insecure default value`.** A secret in `.env`
is still a placeholder. Run `./scripts/init-env.sh` on a fresh checkout, or generate the value
with `openssl rand -hex 32`.

**No certificate / the browser shows a TLS error.** `docker compose logs client`. Check that
both hostnames resolve to this host, that port 80 is reachable from the internet (Let's
Encrypt validates over it), and that you have not hit Let's Encrypt's rate limits by
recreating the `caddy_data` volume repeatedly.

**Signed in, then immediately signed out, or every write fails with 403.** The session cookie
needs HTTPS and the browser's origin must be `https://DOMAIN` exactly. Open Trenova at the
hostname in `DOMAIN`, not by IP address or another alias.

**Uploads fail, or document previews do not load.** The browser talks to
`https://STORAGE_DOMAIN` directly. Check its DNS record and certificate, open
`https://STORAGE_DOMAIN/minio/health/live` in the browser, and check that `STORAGE_ORIGIN`
(defaulting to `https://STORAGE_DOMAIN`) matches where files are served from.

**Search finds nothing.** `docker compose logs gtc`. GTC backfills the indexes the first time
it starts; on a large database that takes a while. `docker compose logs db-bootstrap` shows
the tables published to it.

**The assistant or agents never answer.** `docker compose logs temporal tms-worker`. The
worker runs every AI job; an AI provider must also be configured under **AI control** (`/admin/agent-control`).

**PDFs fail to render.** `docker compose ps gotenberg` must be healthy. Templates cannot load
remote images by design; images are inlined before rendering.

**A service is restarting in a loop.** `docker compose ps` and then `docker compose logs
<service>`. `docker stats` shows whether the host is out of memory.

## Uninstall

```bash
docker compose down          # stop and remove the containers, keep the data
docker compose down -v       # also delete every volume: the database, files, certificates
```

`down -v` is irreversible. Take a backup first.
