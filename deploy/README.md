# Trenova TMS Deployment Guide

Self-hosted deployment documentation for Trenova Transportation Management System.

---

## Table of Contents

1. [Quick Start](#1-quick-start)
2. [Prerequisites](#2-prerequisites)
3. [Installation Guide](#3-installation-guide)
4. [Configuration Reference](#4-configuration-reference)
5. [TLS/HTTPS Certificate Setup](#5-tlshttps-certificate-setup)
6. [Network & VPN Considerations](#6-network--vpn-considerations)
7. [Deployment Checklist](#7-deployment-checklist)
8. [Troubleshooting](#8-troubleshooting)
9. [Maintenance](#9-maintenance)
10. [Architecture Reference](#10-architecture-reference)

---

## 1. Quick Start

### Production Deployment (Recommended)

For production deployments, use pre-built images from GitHub Container Registry:

```bash
cd deploy

# Copy and configure environment
cp .env.example .env
# Edit .env with your domain and generate secure passwords

# Pull and start services
export TRENOVA_VERSION=latest  # or specific version like 1.0.0
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

### Local Development

For local development or testing, use the setup script which builds images locally:

```bash
cd deploy
./setup.sh
```

The setup script will:

- Check prerequisites (Docker, Docker Compose, OpenSSL)
- Prompt for your domain (or use `localhost`)
- Generate cryptographically secure credentials
- Build and start all services
- Run database migrations and seed initial data

**Default Admin Credentials:**

- Email: `admin@trenova.app`
- Password: `admin123!`

**DNS/Hosts Configuration:**

If using a custom domain (not localhost), add both the main domain and storage subdomain to your DNS or hosts file:

```
# Example for trenova.local
127.0.0.1  trenova.local
127.0.0.1  storage.trenova.local
```

**Access URLs (localhost deployment):**

| Service | URL |
|---------|-----|
| Application | <http://localhost> |
| API Health | <http://localhost/api/v1/health> |
| Temporal UI | <http://localhost/temporal-ui> |
| MinIO Console | <http://localhost/minio> |

> **Important:** Change the default admin password immediately after first login.

---

## 2. Prerequisites

### Hardware Requirements

| Spec | Minimum | Recommended |
|------|---------|-------------|
| CPU | 4 cores | 8 cores |
| RAM | 8 GB | 16 GB |
| Storage | 50 GB SSD | 100 GB SSD |

The deployment uses the following resource limits:

- PostgreSQL: 2 GB RAM
- Redis: 2 GB RAM
- Meilisearch: 1 GB RAM
- MinIO: 1 GB RAM
- Temporal: 1 GB RAM
- TMS API: 512 MB RAM
- TMS Worker: 512 MB RAM
- Caddy: 256 MB RAM

### Software Requirements

| Software | Version | Notes |
|----------|---------|-------|
| Docker Engine | 24.0+ | Required |
| Docker Compose | v2 | Plugin or standalone |
| OpenSSL | Any | For secret generation |

### Operating System Support

**Linux:**

- Ubuntu 22.04 LTS or newer
- RHEL 8+ / Rocky Linux 8+
- Debian 12+
- Any distribution with Docker support

**Windows:**

- Windows Server 2019 or newer
- Windows 10/11 with WSL2 (development only)

### Network Requirements

| Port | Protocol | Purpose |
|------|----------|---------|
| 80 | TCP | HTTP (redirect to HTTPS) |
| 443 | TCP | HTTPS (application access) |

**Outbound access required for:**

- Pulling Docker images (see Docker Image Sources below)
- Let's Encrypt ACME challenges (if using automatic certificates)
- GitHub API for update checks (optional)

### Docker Image Sources

| Image | Registry | Purpose |
|-------|----------|---------|
| `ghcr.io/emoss08/trenova-2/tms` | GitHub Container Registry | TMS API and Worker |
| `ghcr.io/emoss08/trenova-2/client` | GitHub Container Registry | Web frontend (Caddy) |
| `ghcr.io/emoss08/trenova-2/postgres` | GitHub Container Registry | PostgreSQL with PostGIS |
| `redis/redis-stack` | Docker Hub | Redis with modules |
| `getmeili/meilisearch` | Docker Hub | Meilisearch |
| `quay.io/minio/minio` | Quay.io | MinIO object storage |
| `temporalio/auto-setup` | Docker Hub | Temporal server |
| `temporalio/ui` | Docker Hub | Temporal UI |

Trenova images are tagged with semantic versions (e.g., `1.0.0`, `1.0`) and `latest`.

---

## 3. Installation Guide

### 3.1 Linux Installation

#### Install Docker

**Ubuntu/Debian:**

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

sudo usermod -aG docker $USER
newgrp docker
```

**RHEL/Rocky Linux:**

```bash
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager --add-repo https://download.docker.com/linux/rhel/docker-ce.repo
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

sudo systemctl enable docker
sudo systemctl start docker
sudo usermod -aG docker $USER
newgrp docker
```

#### Deploy Trenova

```bash
git clone https://github.com/emoss08/trenova-2.git
cd trenova-2/deploy
./setup.sh
```

#### Create Systemd Service (Optional)

For automatic startup on boot, create a systemd service:

```bash
sudo tee /etc/systemd/system/trenova.service > /dev/null <<EOF
[Unit]
Description=Trenova TMS
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/trenova/deploy
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable trenova
```

### 3.2 Windows Server Installation

#### Install Docker

**Option 1: Docker Desktop (Recommended for GUI management)**

1. Download Docker Desktop from <https://www.docker.com/products/docker-desktop>
2. Run the installer
3. Enable WSL2 backend when prompted
4. Restart when prompted
5. Start Docker Desktop

**Option 2: Docker Engine (CLI only)**

Open PowerShell as Administrator:

```powershell
# Install Windows Containers feature
Install-WindowsFeature -Name Containers

# Install Docker via PowerShell
Invoke-WebRequest -UseBasicParsing "https://raw.githubusercontent.com/microsoft/Windows-Containers/Main/helpful_tools/Install-DockerCE/install-docker-ce.ps1" -o install-docker-ce.ps1
.\install-docker-ce.ps1

# Restart after installation
Restart-Computer
```

#### Deploy Trenova

Open PowerShell:

```powershell
git clone https://github.com/emoss08/trenova-2.git
cd trenova-2\deploy

# Convert line endings if needed (Git may have converted them)
dos2unix setup.sh  # Or use: (Get-Content setup.sh -Raw) -replace "`r`n", "`n" | Set-Content setup.sh -NoNewline

# Run setup via WSL or Git Bash
bash setup.sh
```

**Windows-Specific Notes:**

- Use forward slashes `/` in paths within Docker contexts
- Ensure line endings are LF (not CRLF) for shell scripts
- Volume paths use `/c/path/to/dir` format in Docker

#### Windows Service Setup

Use NSSM (Non-Sucking Service Manager) to run as a Windows service:

```powershell
# Download NSSM from https://nssm.cc/download
nssm install Trenova "docker" "compose -f C:\path\to\trenova-2\deploy\docker-compose.yml up"
nssm set Trenova AppDirectory "C:\path\to\trenova-2\deploy"
nssm set Trenova Start SERVICE_AUTO_START
nssm start Trenova
```

---

## 4. Configuration Reference

### Environment Variables (.env)

The `setup.sh` script generates a `.env` file with secure random values. You can customize these:

| Variable | Description | Default |
|----------|-------------|---------|
| `DOMAIN` | Domain name for the application | `localhost` |
| `VITE_API_URL` | API endpoint URL for frontend | `/api/v1` |
| `TRENOVA_DATABASE_NAME` | PostgreSQL database name | `trenova_db` |
| `TRENOVA_DATABASE_USER` | PostgreSQL username | `trenova` |
| `TRENOVA_DATABASE_PASSWORD` | PostgreSQL password | Auto-generated |
| `TRENOVA_CACHE_PASSWORD` | Redis password | Auto-generated |
| `TRENOVA_SECURITY_SESSION_SECRET` | Session signing key (min 32 chars) | Auto-generated |
| `TRENOVA_SECURITY_ENCRYPTION_KEY` | Data encryption key (min 32 chars) | Auto-generated |
| `TRENOVA_GOOGLE_APIKEY` | Google API key used by backend services | Auto-generated placeholder |
| `TRENOVA_SYSTEM_SYSTEMUSERPASSWORD` | Internal system user bootstrap password | Auto-generated |
| `TRENOVA_STORAGE_ACCESSKEY` | MinIO access key | `trenova-minio` |
| `TRENOVA_STORAGE_SECRETKEY` | MinIO secret key | Auto-generated |
| `TRENOVA_MEILI_MASTER_KEY` | Meilisearch master key | Auto-generated |

### Application Configuration (config/config.prod.yaml)

Key configuration sections:

#### Server Settings

```yaml
server:
  host: 0.0.0.0
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s
  idleTimeout: 120s
  cors:
    enabled: false  # Caddy handles CORS
```

#### Database Connection Pool

```yaml
database:
  maxIdleConns: 25   # Reduce for low-memory systems
  maxOpenConns: 200  # Adjust based on expected load
```

```yaml
pgbouncer:
  POOL_MODE: session
  DEFAULT_POOL_SIZE: 80
  MAX_DB_CONNECTIONS: 100
  MAX_CLIENT_CONN: 1000
```

TMS services connect through PgBouncer (`pgbouncer:6432`) while Temporal stays connected directly to PostgreSQL (`postgres:5432`).

#### Redis Cache Pool

```yaml
cache:
  poolSize: 50       # Connection pool size
  minIdleConns: 10   # Minimum idle connections
```

#### Security Settings

```yaml
security:
  session:
    maxAge: 24h           # Session lifetime
    secure: true          # Requires HTTPS
    sameSite: lax         # Cookie policy
    domain: trenova.local # Change to your domain
  rateLimit:
    enabled: true
    requestsPerMinute: 60
    burstSize: 10
```

#### Resource Limits Tuning

Modify `docker-compose.yml` to adjust resource limits:

```yaml
services:
  postgres:
    deploy:
      resources:
        limits:
          memory: 4G  # Increase for larger datasets
        reservations:
          memory: 2G

  tms-api:
    deploy:
      resources:
        limits:
          memory: 1G  # Increase for high concurrency
          cpus: '2'
```

---

## 5. TLS/HTTPS Certificate Setup

Trenova uses Caddy as a reverse proxy, which handles TLS automatically.

### 5.1 Development/Internal (Self-Signed via Caddy)

By default, the Caddyfile uses `tls internal`:

```
{$DOMAIN:trenova.app} {
    tls internal
    ...
}
```

This creates certificates signed by Caddy's internal CA. The certificates are trusted within the container network but not by browsers on client machines.

#### Extracting the Root CA Certificate

```bash
# Copy the root CA from the Caddy container
docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-root-ca.crt

# Verify the certificate
openssl x509 -in caddy-root-ca.crt -text -noout
```

#### Installing the Certificate on End-User Machines

**Windows (Command Line - Recommended for automation):**

```powershell
# Import for current user
certutil -addstore -user Root C:\path\to\caddy-root-ca.crt

# Import for all users (requires Administrator)
certutil -addstore Root C:\path\to\caddy-root-ca.crt
```

**Windows (GUI):**

1. Double-click `caddy-root-ca.crt`
2. Click "Install Certificate..."
3. Select "Local Machine" (for all users) or "Current User"
4. Select "Place all certificates in the following store"
5. Click "Browse" and select "Trusted Root Certification Authorities"
6. Click "Next" then "Finish"

**macOS (Command Line):**

```bash
# Requires admin password
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain caddy-root-ca.crt
```

**macOS (Keychain Access):**

1. Open Keychain Access
2. File → Import Items
3. Select `caddy-root-ca.crt`
4. Double-click the imported certificate
5. Expand "Trust"
6. Set "When using this certificate" to "Always Trust"
7. Close and enter password when prompted

**Linux (Debian/Ubuntu):**

```bash
sudo cp caddy-root-ca.crt /usr/local/share/ca-certificates/trenova-ca.crt
sudo update-ca-certificates
```

**Linux (RHEL/Rocky/Fedora):**

```bash
sudo cp caddy-root-ca.crt /etc/pki/ca-trust/source/anchors/trenova-ca.crt
sudo update-ca-trust
```

**Firefox (All Platforms):**

Firefox uses its own certificate store:

1. Open Firefox Settings
2. Search for "certificates"
3. Click "View Certificates"
4. Go to "Authorities" tab
5. Click "Import"
6. Select `caddy-root-ca.crt`
7. Check "Trust this CA to identify websites"
8. Click OK

#### Windows Group Policy Distribution (Enterprise)

For deploying certificates to multiple Windows machines via Active Directory:

1. Open Group Policy Management Console
2. Create or edit a GPO linked to your OU
3. Navigate to: Computer Configuration → Windows Settings → Security Settings → Public Key Policies → Trusted Root Certification Authorities
4. Right-click → Import
5. Select the certificate file
6. The certificate will be distributed to all computers in the OU

**PowerShell script for bulk deployment:**

```powershell
# Run on domain controller or management workstation
$cert = "\\server\share\caddy-root-ca.crt"
$gpoName = "Trenova Root CA"

# Create GPO
New-GPO -Name $gpoName | New-GPLink -Target "OU=Computers,DC=domain,DC=com"

# Import certificate to GPO (requires GPMC and manual steps, or use LGPO.exe)
```

### 5.2 Production (Let's Encrypt / ACME)

For internet-facing deployments, use automatic certificates from Let's Encrypt:

**Modify the Caddyfile:**

```
{$DOMAIN} {
    # Remove 'tls internal' to use automatic ACME
    # Or explicitly configure:
    tls {
        dns cloudflare {env.CLOUDFLARE_API_TOKEN}  # For DNS-01 challenge
    }
    ...
}
```

**Requirements:**

- Domain must resolve to your server's public IP
- Port 80 must be open for HTTP-01 challenges (or use DNS-01)
- Valid email for Let's Encrypt notifications

**DNS-01 Challenge (for servers behind NAT/firewall):**

```bash
# Add to docker-compose.yml environment
caddy:
  environment:
    CLOUDFLARE_API_TOKEN: your-api-token
```

### 5.3 Custom Certificates (Enterprise CA)

To use certificates from your enterprise CA:

**1. Prepare your certificates:**

```
/path/to/certs/
├── trenova.crt     # Server certificate (with full chain)
└── trenova.key     # Private key
```

**2. Modify docker-compose.yml:**

```yaml
caddy:
  volumes:
    - ./certs/trenova.crt:/etc/caddy/certs/trenova.crt:ro
    - ./certs/trenova.key:/etc/caddy/certs/trenova.key:ro
```

**3. Update Caddyfile:**

```
{$DOMAIN} {
    tls /etc/caddy/certs/trenova.crt /etc/caddy/certs/trenova.key
    ...
}
```

---

## 6. Network & VPN Considerations

### 6.1 Internal Network Only (No VPN)

For deployments accessible only within a corporate LAN:

**DNS Configuration:**

Option A: Internal DNS Server

```
# Add A records to your DNS server
trenova.corp.local          IN  A  192.168.1.100
storage.trenova.corp.local  IN  A  192.168.1.100
```

Option B: Hosts File (per machine)

```
# Windows: C:\Windows\System32\drivers\etc\hosts
# Linux/macOS: /etc/hosts
192.168.1.100  trenova.corp.local
192.168.1.100  storage.trenova.corp.local
```

> **Important:** The `storage.` subdomain is required for file uploads and downloads. Both the main domain and storage subdomain must resolve to the same server.

**Certificate Distribution:**

- Use self-signed certificates (Section 5.1)
- Distribute the root CA to all client machines
- Consider Group Policy for Windows environments

**Firewall Rules:**

```bash
# Allow only internal network
sudo ufw allow from 192.168.1.0/24 to any port 80
sudo ufw allow from 192.168.1.0/24 to any port 443
```

### 6.2 With VPN

**Split-Tunnel VPN:**

- Only traffic to corporate resources goes through VPN
- Ensure the Trenova server IP/subnet is routed through VPN
- DNS must resolve the hostname correctly

**Full-Tunnel VPN:**

- All traffic routes through VPN
- Simpler configuration
- Higher latency for non-corporate traffic

**DNS Resolution:**

- Configure VPN to push DNS settings
- Or use hosts file entries on clients
- Ensure DNS resolves when VPN is connected

**Certificate Trust:**

- If using enterprise CA, certificates are typically already trusted
- For self-signed, distribute the root CA through your standard certificate deployment process

### 6.3 Reverse Proxy / Load Balancer

For deployments behind an existing reverse proxy (nginx, HAProxy, F5, etc.):

**SSL Termination at Load Balancer:**

1. Configure your LB to terminate SSL
2. Modify Caddyfile to accept HTTP:

```
:80 {
    header {
        X-Frame-Options "DENY"
        ...
    }
    ...
}
```

1. Update docker-compose.yml:

```yaml
caddy:
  ports:
    - "8080:80"  # Expose HTTP only, LB handles SSL
```

**Preserving Client IPs:**

Ensure your load balancer sets X-Forwarded-For headers. The Caddyfile is already configured to work with proxied requests.

For nginx:

```nginx
location / {
    proxy_pass http://trenova-backend:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**WebSocket Support (Required for real-time features):**

```nginx
location /ws {
    proxy_pass http://trenova-backend:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

### 6.4 Air-Gapped / Offline Deployment

For environments without internet access:

**Step 1: Save Docker Images (on internet-connected machine)**

```bash
# Pull all required images
docker compose pull

# Save images to a tar file
docker save -o trenova-images.tar \
  $(docker compose config --images | tr '\n' ' ')
```

**Step 2: Transfer to Air-Gapped Machine**

- USB drive
- Secure file transfer
- Physical media

**Step 3: Load Images**

```bash
docker load -i trenova-images.tar
```

**Step 4: Build Application Images**

```bash
# Transfer the entire repository
# Then build locally
docker compose build
```

**Step 5: Deploy**

```bash
./setup.sh
```

**Certificate Considerations:**

- Must use self-signed or enterprise CA certificates
- Cannot use Let's Encrypt (requires internet)
- Pre-generate and include certificates in deployment package

---

## 7. Deployment Checklist

### Pre-Deployment

```
[ ] Hardware meets minimum requirements (4 CPU, 8GB RAM, 50GB SSD)
[ ] Docker Engine 24.0+ installed
[ ] Docker Compose v2 installed
[ ] OpenSSL installed
[ ] Ports 80 and 443 available (netstat -tuln | grep -E ':80|:443')
[ ] Domain/hostname decided
[ ] DNS configured for main domain AND storage subdomain (e.g., trenova.local + storage.trenova.local)
[ ] Firewall rules configured
```

### Deployment

```
[ ] Repository cloned to target machine
[ ] setup.sh executed successfully
[ ] All services healthy: docker compose ps
[ ] No restart loops in logs: docker compose logs --tail=50
[ ] Migrations completed without errors
[ ] Seed data applied
```

### Post-Deployment

```
[ ] Admin login works (admin@trenova.app / admin123!)
[ ] Changed default admin password
[ ] Root CA certificate extracted (if using self-signed)
[ ] Certificate distributed to end users (include storage subdomain cert)
[ ] File upload/download works (test document attachment)
[ ] Application loads correctly in browser
[ ] API health check passes: curl https://your-domain/api/v1/health
```

### Security Hardening

```
[ ] .env file permissions set to 600: chmod 600 .env
[ ] Default passwords changed in .env
[ ] Firewall configured to allow only necessary ports
[ ] Rate limiting verified (try rapid requests)
[ ] HTTPS enforced (HTTP redirects to HTTPS)
[ ] Security headers present (check with securityheaders.com)
```

### Operations

```
[ ] Backup strategy documented and tested
[ ] Log rotation configured
[ ] Monitoring/alerting set up
[ ] Recovery procedure documented and tested
```

---

## 8. Troubleshooting

### Container Fails to Start

**Check container logs:**

```bash
docker compose logs <service-name>
docker compose logs tms-api --tail=100
```

**Check container status:**

```bash
docker compose ps -a
```

**Common causes:**

- Missing environment variables → Check .env file
- Port already in use → `netstat -tuln | grep :80`
- Insufficient memory → Check `docker stats`

### Database Connection Refused

**Symptoms:**

- tms-api fails to start
- Logs show "connection refused" to postgres or pgbouncer

**Solutions:**

```bash
# Check postgres is healthy
docker compose ps postgres

# Check postgres logs
docker compose logs postgres

# Check pgbouncer is healthy
docker compose ps pgbouncer

# Check pgbouncer logs
docker compose logs pgbouncer

# Verify credentials match
grep DATABASE .env
cat config/config.prod.yaml | grep -A5 database:

# Restart database stack
docker compose restart postgres pgbouncer
```

### Certificate Not Trusted

**Symptoms:**

- Browser shows "NET::ERR_CERT_AUTHORITY_INVALID"
- curl shows "SSL certificate problem"

**Solutions:**

1. Extract and install the root CA (Section 5.1)
2. Verify certificate installation:

```bash
# Linux
openssl s_client -connect your-domain:443 -CAfile /etc/ssl/certs/ca-certificates.crt

# macOS
security find-certificate -a -p /Library/Keychains/System.keychain | openssl x509 -text
```

### Session/Login Issues

**Symptoms:**

- Login succeeds but immediately logs out
- "Invalid session" errors

**Solutions:**

1. Check cookie domain in config.prod.yaml matches your domain
2. Ensure `secure: true` is set for HTTPS deployments
3. Check browser cookie settings
4. Verify system clocks are synchronized

```yaml
# config/config.prod.yaml
security:
  session:
    domain: your-actual-domain.com  # Must match
    secure: true                     # Must be true for HTTPS
```

### CORS Errors

**Symptoms:**

- Browser console shows CORS errors
- API requests blocked

**Solutions:**

1. Ensure VITE_API_URL matches the actual domain
2. Verify Caddy is routing correctly
3. Check browser network tab for actual request URLs

```bash
# Rebuild frontend with correct API URL
VITE_API_URL=https://your-domain.com/api/v1 docker compose build caddy
docker compose up -d caddy
```

### Migration Failures

**Symptoms:**

- tms-migrate container exits with error
- Database tables missing

**Solutions:**

```bash
# Check migration logs
docker compose logs tms-migrate

# Manually run migrations
docker compose run --rm tms-api db migrate

# If database is corrupted, reset (WARNING: data loss)
docker compose down -v
docker volume rm deploy_postgres_data
docker compose up -d postgres
# Wait for healthy
docker compose run --rm tms-api db migrate
docker compose run --rm tms-api db seed
```

### Service Dependency Issues

**Symptoms:**

- Services start in wrong order
- "Connection refused" during startup

**Solutions:**

```bash
# Start services in order
docker compose up -d postgres pgbouncer redis minio meilisearch
docker compose up -d temporal
docker compose run --rm tms-api db migrate
docker compose up -d tms-api tms-worker caddy
```

### High Memory Usage

**Symptoms:**

- OOM kills
- Slow performance

**Solutions:**

```bash
# Check memory usage
docker stats

# Reduce limits in docker-compose.yml
# Or increase host memory
```

---

## 9. Maintenance

### Backup Procedures

**Database Backup:**

```bash
# Create backup
docker compose exec postgres pg_dump -U trenova trenova_db > backup_$(date +%Y%m%d_%H%M%S).sql

# Automated daily backup (add to crontab)
0 2 * * * cd /opt/trenova/deploy && docker compose exec -T postgres pg_dump -U trenova trenova_db | gzip > /backups/trenova_$(date +\%Y\%m\%d).sql.gz
```

**Volume Backup:**

```bash
# Stop services first for consistency
docker compose stop

# Backup all volumes
docker run --rm -v deploy_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_data.tar.gz /data
docker run --rm -v deploy_redis_data:/data -v $(pwd):/backup alpine tar czf /backup/redis_data.tar.gz /data
docker run --rm -v deploy_minio_data:/data -v $(pwd):/backup alpine tar czf /backup/minio_data.tar.gz /data
docker run --rm -v deploy_meili_data:/data -v $(pwd):/backup alpine tar czf /backup/meili_data.tar.gz /data

docker compose start
```

**Restore from Backup:**

```bash
# Stop services
docker compose down

# Restore database
docker compose up -d postgres
docker compose exec -T postgres psql -U trenova trenova_db < backup.sql

# Or restore volumes
docker run --rm -v deploy_postgres_data:/data -v $(pwd):/backup alpine tar xzf /backup/postgres_data.tar.gz -C /

docker compose up -d
```

### Update Notifications

Trenova automatically checks for new versions and notifies administrators in the web UI. When an update is available:

1. A banner appears at the top of the admin interface
2. The banner shows the new version and links to release notes
3. Only users with admin permissions see the notification
4. The banner can be dismissed (per version)

**Configuration (config.prod.yaml):**

```yaml
update:
  enabled: true           # Enable/disable update checks
  checkInterval: 1h       # How often to check
  githubOwner: emoss08    # GitHub organization
  githubRepo: trenova     # GitHub repository
  allowPrerelease: false  # Include pre-release versions
  offlineMode: false      # Disable remote checks (air-gapped)
```

**API Endpoints:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/system/version` | GET | Current version info |
| `/api/v1/system/update-status` | GET | Update availability (admin) |
| `/api/v1/system/check-updates` | POST | Force update check (admin) |

### Upgrade Process

Trenova provides CLI commands for managing updates. The recommended method uses pre-built images from GitHub Container Registry.

**Using CLI Commands (Recommended):**

```bash
# Check for available updates
trenova update check

# View current version and update status
trenova update status

# Apply update to latest version
trenova update apply

# Apply update to specific version
trenova update apply 1.2.0

# Rollback to previous version if needed
trenova update rollback
```

**CLI Flags:**

| Flag | Description |
|------|-------------|
| `--skip-backup` | Skip creating backup before update |
| `--skip-migrations` | Skip running database migrations |
| `--force` | Force update even if already on latest |
| `--compose-file` | Specify compose file (default: docker-compose.prod.yml) |

**Manual Upgrade (Alternative):**

```bash
# 1. Backup current installation
./backup.sh  # or manual backup commands above

# 2. Pull latest images from ghcr.io
export TRENOVA_VERSION=1.2.0  # or 'latest'
docker compose -f docker-compose.prod.yml pull

# 3. Stop services
docker compose -f docker-compose.prod.yml down

# 4. Run migrations
docker compose -f docker-compose.prod.yml run --rm tms-migrate

# 5. Start all services
docker compose -f docker-compose.prod.yml up -d

# 6. Verify health
docker compose -f docker-compose.prod.yml ps
curl https://your-domain/api/v1/health
```

### Log Management

**View logs:**

```bash
# All services
docker compose logs

# Specific service with follow
docker compose logs -f tms-api

# Last 100 lines
docker compose logs --tail=100 tms-api
```

**Log rotation (Docker daemon config):**

```json
// /etc/docker/daemon.json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  }
}
```

**External log aggregation:**

```yaml
# docker-compose.yml - add logging config
services:
  tms-api:
    logging:
      driver: syslog
      options:
        syslog-address: "tcp://logserver:514"
        tag: "trenova-api"
```

### Health Monitoring

**Basic health check script:**

```bash
#!/bin/bash
DOMAIN="your-domain.com"

# Check API health
if ! curl -sf "https://${DOMAIN}/api/v1/health" > /dev/null; then
    echo "API health check failed"
    # Send alert
fi

# Check all containers
unhealthy=$(docker compose ps --format json | jq -r 'select(.Health != "healthy" and .Health != "") | .Service')
if [ -n "$unhealthy" ]; then
    echo "Unhealthy services: $unhealthy"
    # Send alert
fi
```

**Prometheus metrics (optional):**

Enable in config.prod.yaml:

```yaml
monitoring:
  metrics:
    enabled: true
    port: 9090
    path: /metrics
```

Expose the metrics port and configure Prometheus to scrape.

---

## 10. Architecture Reference

### Service Dependencies

```
                    ┌─────────────┐
                    │   Caddy     │ :80, :443
                    │  (Reverse   │
                    │   Proxy)    │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │  TMS API │ │ Temporal │ │  MinIO   │
        │   :8080  │ │    UI    │ │ Console  │
        └────┬─────┘ └──────────┘ └──────────┘
             │
    ┌────────┼────────┬────────────┬───────────┐
    ▼        ▼        ▼            ▼           ▼
┌────────┐┌──────┐┌───────┐┌───────────┐┌──────────┐
│Postgres││Redis ││ MinIO ││Meilisearch││ Temporal │
│  :5432 ││:6379 ││ :9000 ││   :7700   ││  :7233   │
└────────┘└──────┘└───────┘└───────────┘└──────────┘
```

### Port Mappings

| Service | Internal Port | External Port | Purpose |
|---------|---------------|---------------|---------|
| Caddy | 80, 443 | 80, 443 | HTTP/HTTPS entry |
| PostgreSQL | 5432 | - | Database |
| Redis | 6379 | - | Cache |
| MinIO | 9000, 9001 | - | Object storage |
| Meilisearch | 7700 | - | Search |
| Temporal | 7233 | - | Workflow engine |
| Temporal UI | 8080 | - | Workflow dashboard |
| TMS API | 8080 | - | Application API |

### Volume Mappings

| Volume | Container Path | Purpose |
|--------|----------------|---------|
| postgres_data | /var/lib/postgresql/data | Database files |
| redis_data | /data | Cache persistence |
| minio_data | /data | Uploaded files |
| meili_data | /meili_data | Search indices |
| caddy_data | /data | Certificates, etc. |
| caddy_config | /config | Caddy configuration |

### Network Topology

All services run on an internal Docker bridge network (`trenova`). Only Caddy exposes ports to the host.

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Host                          │
│  ┌───────────────────────────────────────────────────┐  │
│  │              trenova network (bridge)             │  │
│  │                                                   │  │
│  │  postgres ◄──► pgbouncer ◄──► tms-api ◄──► caddy │  │
│  │                                      └────► :80/:443 │  │
│  │  redis    ◄──┘         │                         │  │
│  │  minio    ◄────────────┤                         │  │
│  │  meili    ◄────────────┤                         │  │
│  │  temporal ◄────────────┘                         │  │
│  │                                                   │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Caddy Documentation](https://caddyserver.com/docs/)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Temporal Documentation](https://docs.temporal.io/)

For support, open an issue at: <https://github.com/emoss08/trenova-2/issues>
