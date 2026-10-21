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
- [Document extraction](#document-extraction)
- [Malware scanning](#malware-scanning)
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
| `tms-bootstrap` | `ghcr.io/emoss08/trenova/tms` | One-shot: creates the first organization and administrator once (`trenova db bootstrap`); a no-op afterwards |
| `db-bootstrap` | `ghcr.io/emoss08/trenova/postgres` | One-shot: change-data-capture publication |
| `postgres` | `ghcr.io/emoss08/trenova/postgres` | PostgreSQL 18 with PostGIS, pg_cron and pgvector |
| `redis` | `redis:8` | Sessions, rate limits, caches, the live-update streams |
| `minio` | `pgsty/silo` (MinIO) | Documents and other uploaded files |
| `meilisearch` | `getmeili/meilisearch` | Global search |
| `temporal` | `temporalio/auto-setup` | Durable workflows; state in its own two PostgreSQL databases |
| `gotenberg` | `gotenberg/gotenberg` | HTML-to-PDF rendering, locked down and unreachable from outside |
| `docling` | `ghcr.io/docling-project/docling-serve-cpu` | Reads uploaded files for document extraction (PDFs, scans, photos, Office files, email), locked down and unreachable from outside |
| `imaging` | `ghcr.io/emoss08/trenova/imaging` | Document thumbnails, page views, capture page inspection, PDF assembly and image normalization; every file is parsed in a hardened child process |
| `imaging-certs` | `ghcr.io/emoss08/trenova/imaging` | One-shot: issues the private CA and the certificates the TMS and the imaging worker authenticate each other with, once |
| `gtc` | `ghcr.io/emoss08/gtc` | Change data capture: PostgreSQL to Meilisearch and Redis |
| `clamav` (optional, `--profile clamav`) | `clamav/clamav` | Malware scanning of every file before it is opened; see [Malware scanning](#malware-scanning) |

The `tms-api`, `tms-worker`, `tms-migrate` and `tms-bootstrap` services are one image with
different commands.
Only Caddy publishes ports (80 and 443, TCP and UDP). The databases, queues, search, Temporal,
Gotenberg, docling-serve, the imaging worker and GTC sit on an internal Docker network with no route off the host; the API and
worker get a separate network for outbound calls (AI providers, email, accounting
integrations, release checks).

## Requirements

- A 64-bit Linux host (x86_64; the TMS and postgres images are built for `linux/amd64`).
- **4 vCPU, 8 GB RAM, 50 GB SSD** for a small fleet. Give PostgreSQL more memory as data grows
  (see [Tuning](#tuning)).
- The `docling` service adds about **8 GB of disk** for its image (its models are inside it)
  and **2.5 GB of memory** while it reads (about 1.5 GB idle; compose caps it at 4 GB). On
  4 vCPU it takes roughly 8 to 15 seconds a page for PDFs and scans and well under a second
  for Office files and email. Document extraction needs it; see
  [Document extraction](#document-extraction).
- The `imaging` service takes **2 vCPU and up to 3 GB of memory** by default
  (`TRENOVA_IMAGING_CPUS` and `TRENOVA_IMAGING_MEMORY` in `.env`); it runs one child process
  per CPU, each needing about 1 GB under load, and idles at a few megabytes. Every document
  preview, page view and capture page goes through it, so the API and worker wait for it to be
  healthy.
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

   It also asks for your first organization and its administrator: the organization name,
   the administrator's name and email address, and a password. Leave the password empty
   and it generates one and **prints it once**; note it down. Without a terminal (in
   automation), set `TRENOVA_BOOTSTRAP_ORG_NAME`, `TRENOVA_BOOTSTRAP_ADMIN_NAME`,
   `TRENOVA_BOOTSTRAP_ADMIN_EMAIL` and optionally `TRENOVA_BOOTSTRAP_ADMIN_PASSWORD` in the
   environment before running it. No install ever starts with a default password.

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
   `tms-migrate` runs every migration and the base seeds, `tms-bootstrap` creates your
   organization and administrator, and Caddy obtains certificates. `tms-migrate`,
   `tms-bootstrap` and `db-bootstrap` show as `exited (0)` when they have finished; every
   other service should become `healthy` or `running`.

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

An install starts with no account at all. The base seeds that `tms-migrate` applies create
only the reference data every install needs (US states, DOT hazardous materials, IFTA
jurisdictions, jurisdiction rules, roles and permissions); they never create a user.

On the first start, `tms-bootstrap` runs `trenova db bootstrap` with the `TRENOVA_BOOTSTRAP_*`
values `init-env.sh` wrote to `.env`, and creates:

- your organization, in a business unit of its own, with its chart of accounts, document
  types, document templates, sequences and the other defaults a new organization gets;
- its first administrator, holding the **Organization Administrator** role, who signs in
  with `TRENOVA_BOOTSTRAP_ADMIN_EMAIL` and the password you chose or were shown;
- the internal system account background work runs as (password
  `TRENOVA_SYSTEM_SYSTEMUSERPASSWORD`, which nobody signs in with).

Each of these is recorded in the audit trail as a critical entry. The password must have at
least 12 characters, must not be a common password and must not contain the email address;
`tms-bootstrap` exits with an error naming the value at fault otherwise, and the API does not
start until it succeeds (`docker compose logs tms-bootstrap`).

`tms-bootstrap` runs on every start, and after the first it does nothing: it recognizes the
same organization name, administrator name and email and exits successfully without reading
the password. You may delete `TRENOVA_BOOTSTRAP_ADMIN_PASSWORD` from `.env` once you have
signed in. It refuses, and leaves the database untouched, when:

- those three values differ from the ones it was first run with: restore them, or empty
  `TRENOVA_BOOTSTRAP_ADMIN_EMAIL` to skip the step. Rename the organization and manage
  administrators in the app instead;
- the database already has users it did not create, such as a restored backup or an install
  from before this step existed: sign in with an existing administrator.

`docker compose run --rm tms-bootstrap` runs the step on its own (after the migrations), for
example after correcting a value in `.env`. To give the values on the command line instead of
in `.env`, leave `TRENOVA_BOOTSTRAP_ADMIN_EMAIL` empty there and run:

```bash
docker compose run --rm --entrypoint trenova tms-bootstrap db bootstrap \
  --org-name "Acme Freight" --admin-name "Dana Whitfield" --admin-email dana@acme.example
```

It prompts for the password (or reads `TRENOVA_BOOTSTRAP_ADMIN_PASSWORD`, or the first line of
standard input with `--password-stdin`; the password is never a flag). `--dry-run` reports what
it would do. `trenova db bootstrap --help` lists the optional organization details
(`--timezone`, `--address-line1`, `--city`, `--state`, `--postal-code`, `--scac`,
`--dot-number`, each also a `TRENOVA_BOOTSTRAP_*` variable); placeholders are used for any you
leave out.

Then, straight away:

1. Sign in at `https://DOMAIN` with the administrator's email address and password.
2. Complete your organization's details (**Organization settings**,
   `/admin/organization-settings`): address, SCAC code, DOT number and time zone, if you did
   not give them to the bootstrap.
3. Set up outgoing email (**Email profiles**, `/organization/email-profiles`, with a Resend or
   Postmark account): password resets, invitations and scheduled reports are sent through the
   provider an organization configures.
4. Add your users (**Users**, `/admin/users`): create each person, give them roles, and send
   them a password reset link to choose their own password. Give a second person the
   Organization Administrator role, so you are not locked out if one administrator leaves.

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
| Models served on your own network (vLLM, SGLang, LM Studio and other OpenAI-compatible servers) | `ai.privateNetworkProviders` (on by default here) |
| Driver portal web push | `TRENOVA_PUSH_VAPIDPUBLICKEY`, `TRENOVA_PUSH_VAPIDPRIVATEKEY`, `TRENOVA_PUSH_SUBJECT` in `.env` |
| One-click QuickBooks or Xero for every organization | `TRENOVA_QUICKBOOKS_*`, `TRENOVA_XERO_*` in `.env`; register the redirect URI `https://DOMAIN/admin/integrations/quickbooks/callback` (or `/xero/callback`) |
| Rate limits | `security.rateLimit` |
| How long people stay signed in | `security.session`: `maxAge` (12 hours by default) ends a session that long after sign-in however busy it is; `idleTimeout` (30 minutes) signs out a browser left untouched, after a warning `idleWarning` (2 minutes) before. `idleTimeout: 0` turns the idle sign-out off, for wall displays. Sessions signed in before a change follow the new limits from their next request. |
| External S3-compatible storage | `TRENOVA_STORAGE_*` and `STORAGE_ORIGIN` in `.env` (see the comments there) |
| Release notices in the app | `update` |
| Trenova Capture, the Windows scanning companion | `app.webBaseUrl` must be the address people open Trenova at (the compose file sets it from `DOMAIN`), or no computer can be paired. To offer the installer from **My scanners** and keep computers updating themselves, set `update.capturePublicKey` to the key the releases are signed with; without it the app points at the [GitHub releases](https://github.com/emoss08/Trenova/releases?q=capture) instead. The client image publishes `/.well-known/trenova-capture.json`, which the companion reads to find the API; nothing to configure when the API is served beside the app as here. |

### Tuning

[`postgresql.conf`](../../deploy/selfhost/postgresql.conf) is sized for an 8 GB host. On a
larger one, raise `shared_buffers` to about a quarter of RAM and `effective_cache_size` to
about half, then `docker compose restart postgres`. Redis is capped at 1 GB
(`REDIS_MAXMEMORY` in `.env`). If page views or capture intake queue up, give the imaging worker
more CPUs (`TRENOVA_IMAGING_CPUS`, with about 1 GB of `TRENOVA_IMAGING_MEMORY` per CPU) and
`docker compose up -d imaging`.

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

### Upgrading to a release with document conversion

Releases that bundle the `docling` service read a new key from `.env`,
`TRENOVA_DOCUMENTCONVERSION_APIKEY`. An `.env` written before it existed still works: every
compose command runs, the stack starts, and document extraction reports that it is not set
up. The `docling` service itself refuses to start without a key and says so in
`docker compose logs docling`. Add a generated key to the existing `.env`, from this
directory, without re-running `init-env.sh` (which would replace every other secret):

```bash
grep -q '^TRENOVA_DOCUMENTCONVERSION_APIKEY=.' .env || {
  sed -i.bak '/^TRENOVA_DOCUMENTCONVERSION_APIKEY=/d' .env
  printf 'TRENOVA_DOCUMENTCONVERSION_APIKEY=%s\n' "$(openssl rand -hex 32)" >> .env
}
docker compose up -d
```

The first line leaves a key that is already set alone, so the command is safe to run twice.

### Upgrading to a release with keyed recovery codes

Two-factor recovery codes are now stored as HMAC-SHA256 digests keyed by
`TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER`, which production requires: `docker compose` and the
API both refuse to start without it. Add a generated value to the existing `.env` the same
way:

```bash
grep -q '^TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER=.' .env || {
  sed -i.bak '/^TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER=/d' .env
  printf 'TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER=%s\n' "$(openssl rand -hex 32)" >> .env
}
docker compose up -d
```

Codes issued before the upgrade keep working until their owner generates new ones; the
two-factor settings ask each such user to do so. Back the value up with `.env`, and never
change it once set: every recovery code stored under it would stop working.

### Upgrading to a release with an encryption keyring

The local key manager now derives its key-encryption key from
`TRENOVA_SECURITY_ENCRYPTION_KEY` with HKDF-SHA256 and names it by a fingerprint
(`local-hkdf-v2:…`). Values sealed before the upgrade name the old SHA-256 key
(`local-sha256-v1`) and stay readable with the same `TRENOVA_SECURITY_ENCRYPTION_KEY`;
nothing has to be changed for the upgrade. To move them onto the new key, run once after
the upgrade:

```bash
docker compose run --rm tms-api encryption rewrap --dry-run
docker compose run --rm tms-api encryption rewrap
```

The API, worker and every `trenova` command now refuse to start when the key manager
cannot be built, instead of starting with encryption silently switched off: an empty
`TRENOVA_SECURITY_ENCRYPTION_KEY` with `keyManager: local`, or with `keyManager:
gcp-autokey` a missing crypto key or a KMS client that cannot be created (no credentials,
an unreadable credentials file).

## Backups and restore

What to keep, from most to least important:

| Data | Where | How |
|---|---|---|
| Secrets | `.env` | Copy it off the host whenever it changes. Without `TRENOVA_SECURITY_ENCRYPTION_KEY` the encrypted fields of a database backup cannot be read, and without `TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER` no stored recovery code can be checked. |
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

### Rotating the encryption key

Secrets and documents are sealed with a random data key, and the data key with the key
named in `.env`. Rotating replaces the outer key without decrypting anything else:

1. Move the current value to `TRENOVA_SECURITY_ENCRYPTION_RETIREDKEYS` (comma-separated
   if there are already retired keys) and set a new `TRENOVA_SECURITY_ENCRYPTION_KEY`
   (`openssl rand -hex 32`). Back up `.env`.
2. `docker compose up -d`. New values are sealed with the new key; values sealed with a
   retired key are still read.
3. `docker compose run --rm tms-api encryption rewrap --dry-run` reports how many values
   and documents each kind of record still holds under a retired key, and
   `docker compose run --rm tms-api encryption rewrap` moves them. It writes each row only
   if nothing changed it since it was read, so it can run while Trenova is in use; a row
   changed meanwhile is reported as a conflict and moved by the next run. It also moves
   AI provider keys, agent extension secrets and the CarrierOK and FMCSA QCMobile keys
   that older releases sealed without a tenant binding onto their binding.
4. When a run reports `Every sealed value is on the active key.`, remove the retired value
   from `.env` and restart. Keep the backup of the old `.env` for as long as you keep
   database or bucket backups taken before the rewrap: they can only be read with the old
   key.

Documents and capture pages are rewritten in place in the bucket. With bucket versioning
or object lock, the earlier versions keep the old wrapping until they expire, so keep the
retired key until they have. `--skip-objects` moves the database values alone and
`--org org_…` limits a run to one organization.

To move from the local key manager to Cloud KMS, set `keyManager: gcp-autokey` with its
crypto key, keep the local key in `TRENOVA_SECURITY_ENCRYPTION_KEY` or
`TRENOVA_SECURITY_ENCRYPTION_RETIREDKEYS` so existing values stay readable, and run the
rewrap the same way.

## Document extraction

Uploaded rate confirmations, bills of lading and delivery receipts are read so the AI can
classify them and draft shipments from them. Docling reads them: the worker sends each file to
the `docling` service, which reads PDFs, scans and phone photos (it does all OCR, with RapidOCR
by default), DOCX, XLSX, PPTX and email into page text, real tables and boxes for highlighting.
The older binary Office formats (DOC, XLS, PPT) and RTF are refused; save them as PDF or DOCX
first. Whether scans are read at all is the **Enable OCR** switch in **Admin → Document
extraction**.

Extraction needs three things:

- **The `docling` service**, which the worker reaches at `documentConversion.doclingUrl` with
  the key in `documentConversion.apiKey`.
- **`ai.documentExtraction: true`** in `config/config.yaml`, which ships `false`.
- **An extraction provider**: an AI provider assigned to the **Document extraction** task
  (**Admin → AI providers**). A provider assigned only to chat does not count.

Without the setting or a provider nothing is read. Without the service, nothing but plain text
is read. Either way documents are still stored and searchable by name, description and tags,
and the upload screens, **Admin → Document extraction** and the document's failure message name
what is missing.

`config/config.yaml` sets `documentConversion.doclingUrl` to the bundled service, and
`.env` holds the key both sides use (`TRENOVA_DOCUMENTCONVERSION_APIKEY`, generated by
`init-env.sh`). The file and page limits in `documentConversion` match the service's
`DOCLING_SERVE_MAX_FILE_SIZE` and `DOCLING_SERVE_MAX_NUM_PAGES`; change them together. An
install with no AI provider can leave `doclingUrl` empty and delete the `docling` service and
the `tms-worker` dependency on it from `compose.yml`.

**GPU (optional).** On a host with an NVIDIA GPU and the container toolkit,
`docker compose --profile gpu up -d` adds `docling-gpu` (the CUDA build) and `docling-vlm`, a
vision model that reads each page image. Put the `granite-docling-258M` weights in the
`docling_vlm_models` volume first (the internal network cannot download them), then set
`documentConversion.doclingUrl: http://docling-gpu:5001` and
`documentConversion.vlmEndpoint: http://docling-vlm:8000`. Only this profile lets
docling-serve call another service, and both stay on the internal network.

## Malware scanning

Trenova can check every file it receives for malware before anything opens it: documents
uploaded in the web app and the driver portal, profile pictures and organization logos, rate
sheets, inbound email attachments, pages from the capture companion, and inbound EDI files
received over AS2 or SFTP. The check runs on the bytes as they arrive, before the file is
stored, and before PDFium, pdfcpu, the spreadsheet readers, the zip and XML parsers or the
EDI parser read it. Scanning uses ClamAV's `clamd` over its `INSTREAM` command, so nothing is
written to disk on either side.

Scanning is off until an address is set. With it off, the API and worker log a warning at
startup saying files are not scanned.

**Turning it on.**

1. Start the bundled scanner: `docker compose --profile clamav up -d clamav`. On first start
   it downloads the signature databases (a few hundred megabytes) into the `clamav_data`
   volume, which takes several minutes; `docker compose ps clamav` shows it healthy when it is
   ready. It needs about 2 GB of memory (4 GB while it reloads signatures) and the outbound
   network for its hourly signature updates; `clamd` itself is reachable only on the internal
   network.
2. Add `TRENOVA_CONTENTSCAN_CLAMAV_ADDRESS=clamav:3310` to `.env`, and add `--profile clamav`
   to every later `docker compose up`, or set `COMPOSE_PROFILES=clamav` in `.env`.
3. `docker compose up -d` so the API and worker pick up the address. Their startup log says
   scanning is on, or that the scanner did not answer.

To use a `clamd` you already run, point the address at it instead (`host:3310`, or
`unix:/path/to/clamd.sock` with the socket mounted into `tms-api` and `tms-worker`). Its
`StreamMaxLength` must be at least the largest file you accept (`storage.maxFileSize`).

**What happens to a file.**

- *Clean*: it is stored and processed as before.
- *Infected*: it is refused and never stored. The person uploading it is told the signature
  the scanner found; an inbound email attachment is marked on its message as not filed, with
  the reason; an inbound EDI file is moved to **Quarantined** with the signature as its
  failure reason, which raises the usual quarantined-file alert. Every refusal writes a
  critical audit entry against the document resource (operation *reject*) naming the
  organization, the file, where it came from and the signature.
- *Scanner unreachable*: the file is refused (a person is asked to try again in a few
  minutes; an inbound EDI file stays **Received** and is picked up on the next attempt).
  Scanning fails closed: a configured scanner that is down holds every file until it is back.

**Settings** (`contentScan` in `config/config.yaml`, or the matching `TRENOVA_CONTENTSCAN_*`
variables):

| Setting | Default | Meaning |
|---|---|---|
| `clamav.address` | empty (off) | `host:port` or `unix:/path` of `clamd` |
| `clamav.timeout` | `60s` | Longest one scan may take, connection included |
| `clamav.maxStreamBytes` | `104857600` (100 MiB) | Largest file sent to the scanner; a larger file is refused as not checkable. Keep it at or below `clamd`'s `StreamMaxLength` |
| `failOpen` | `false` | Let files through unscanned while a configured scanner is unreachable. Refused in production and staging |

## Hardening

The stack is secure by default for a single-operator host: TLS everywhere, secrets generated
per install, no account with a default or published password, every backing service on an
internal network, Gotenberg and docling-serve sandboxed, the imaging worker parsing every file
in a child process under seccomp, landlock and resource limits and reached only over mutual
TLS, and nothing published except Caddy. Beyond that:

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
- **Malware scanning.** Turn on ClamAV (see [Malware scanning](#malware-scanning)) when files
  arrive from outside your company: email attachments, EDI partners, customers' portals.
- **Encryption key.** The local key manager keeps `TRENOVA_SECURITY_ENCRYPTION_KEY` in `.env`.
  Treat `.env` like the database itself, and rotate the key (see "Rotating the encryption
  key") whenever someone who could read `.env` should no longer be able to.

## Troubleshooting

**`tms-migrate` exited with an error.** `docker compose logs tms-migrate`. A configuration
error names the key at fault (an unknown key, a missing secret, a placeholder value). A
database error usually means PostgreSQL was not ready or the database password in `.env`
changed after the volume was created; PostgreSQL keeps the password it was initialized with.

**`tms-bootstrap` exited with an error, and the API never starts.** `docker compose logs
tms-bootstrap`. `bootstrap inputs are not valid` lists each value at fault with the flag and
variable that set it, for example a password shorter than 12 characters; fix it in `.env` and
run `docker compose up -d` again. `bootstrap refused: instance was bootstrapped with
different inputs` means `TRENOVA_BOOTSTRAP_ORG_NAME`, `_ADMIN_NAME` or `_ADMIN_EMAIL` changed
after the first start: restore them, or empty `TRENOVA_BOOTSTRAP_ADMIN_EMAIL`. `bootstrap
refused: instance already has users` means the database was set up some other way (a restored
backup); empty `TRENOVA_BOOTSTRAP_ADMIN_EMAIL` and sign in with an existing administrator.

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

**Documents are not extracted.** `docker compose ps docling` must be healthy, and `docker compose logs tms-worker` must not
report `document converter is not reachable` or a refused API key (the key in `.env` must
reach both `docling` and `tms-worker`; recreate both after changing it). A document marked
**Unavailable** with no draft means no AI provider can serve document extraction for that
organization; add one, then extract the document again. A file refused as too large or too
long is over `documentConversion.maxFileBytes` or `maxPages`.

**Previews, page views or capture pages fail.** `docker compose ps imaging` must be healthy,
and `imaging-certs` must have exited 0. `docker compose logs imaging` says why a call was
refused: a file the worker could not read is reported as such and stays downloadable; `a child
was lost during a job` means a file faulted a native library, and only that file fails. If the
TMS logs certificate errors after the volumes were recreated, reissue the certificates with
`docker compose run --rm imaging-certs certs --force --server-dir=/certs/server
--client-dir=/certs/client --ca-dir=/certs/ca --server-name=imaging` and
`docker compose up -d --force-recreate imaging tms-api tms-worker`.

**A service is restarting in a loop.** `docker compose ps` and then `docker compose logs
<service>`. `docker stats` shows whether the host is out of memory.

## Uninstall

```bash
docker compose down          # stop and remove the containers, keep the data
docker compose down -v       # also delete every volume: the database, files, certificates
```

`down -v` is irreversible. Take a backup first.
