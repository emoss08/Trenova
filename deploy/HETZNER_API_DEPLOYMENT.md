# Hetzner API Deployment

This is the temporary production path for running the Trenova API on a Hetzner CPX31 with Cloudflare in front.

Target shape:

- Public API: `https://api.trenova.app`
- Public ingress: Cloudflare Tunnel only
- Admin access: Tailscale SSH only
- Public SSH: temporary bootstrap only, then removed
- Database: local Postgres container on the Hetzner server
- Object storage: Cloudflare R2
- Frontend: deployed separately

## 1. Create An SSH Key

Run this on your local machine, not on the server.

```bash
ssh-keygen -t ed25519 -C "hetzner-trenova" -f ~/.ssh/hetzner_trenova
```

When it asks for a passphrase, use one if you use a password manager. Otherwise press Enter.

Print the public key:

```bash
cat ~/.ssh/hetzner_trenova.pub
```

Copy the whole output. It starts with `ssh-ed25519`.

## 2. Add The Key To Hetzner

In Hetzner Cloud Console:

1. Open `Security`.
2. Open `SSH Keys`.
3. Click `Add SSH Key`.
4. Paste the public key from `~/.ssh/hetzner_trenova.pub`.
5. Name it `trenova-cpx31`.
6. Save.

When creating the CPX31 server, select this SSH key.

If the server already exists without the key, use the Hetzner web console once, then add the key manually:

```bash
mkdir -p ~/.ssh
chmod 700 ~/.ssh
nano ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

Paste the public key into `authorized_keys`, save, then test SSH from your local machine:

```bash
ssh -i ~/.ssh/hetzner_trenova root@<server-public-ip>
```

## 3. Bootstrap The Server

Public SSH is only for this bootstrap step. After Tailscale SSH works, remove public SSH from the Hetzner firewall.

SSH in:

```bash
ssh -i ~/.ssh/hetzner_trenova root@<server-public-ip>
```

Install base packages:

```bash
apt update && apt upgrade -y
apt install -y ca-certificates curl gnupg ufw git jq openssl
```

Install Docker:

```bash
install -m 0755 -d /etc/apt/keyrings

curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
  | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list

apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
systemctl enable --now docker
```

Install Tailscale:

```bash
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up --ssh
```

Open the login URL printed by Tailscale and approve the server.

From your local machine, verify Tailscale SSH:

```bash
tailscale status
ssh root@<server-tailscale-name-or-ip>
```

Do not continue until this works.

## 4. Lock Down Inbound Traffic

On the server:

```bash
ufw default deny incoming
ufw default allow outgoing
ufw allow in on tailscale0
ufw enable
ufw status verbose
```

In Hetzner Cloud Console, remove the temporary public SSH firewall rule. The final firewall should have no public inbound rules for this server.

## 5. Create The Cloudflare Tunnel

In Cloudflare:

1. Open `Zero Trust`.
2. Go to `Networks` -> `Tunnels`.
3. Create a tunnel named `trenova-api`.
4. Choose `cloudflared`.
5. Copy the tunnel token.

Add a public hostname:

```text
Hostname: api.trenova.app
Service: http://tms-api:8080
```

Cloudflare will create or update the DNS record for `api.trenova.app`.

## 6. Clone Trenova

On the server, over Tailscale SSH:

```bash
mkdir -p /opt
cd /opt
git clone https://github.com/emoss08/Trenova.git trenova
cd /opt/trenova/deploy
cp .env.example .env
```

## 7. Configure Environment

Generate secrets:

```bash
openssl rand -base64 32
openssl rand -base64 32
openssl rand -base64 32
openssl rand -base64 32
```

Edit `.env`:

```bash
nano /opt/trenova/deploy/.env
```

Use this shape:

```env
DOMAIN=api.trenova.app
VITE_API_URL=https://api.trenova.app/api/v1

TRENOVA_DATABASE_NAME=trenova_db
TRENOVA_DATABASE_USER=trenova
TRENOVA_DATABASE_PASSWORD=<generated-secret>

TRENOVA_SECURITY_SESSION_SECRET=<generated-secret>
TRENOVA_SECURITY_ENCRYPTION_KEY=<generated-secret>
TRENOVA_SECURITY_SESSION_DOMAIN=

TRENOVA_STORAGE_PROVIDER=r2
TRENOVA_STORAGE_ENDPOINT=https://<cloudflare-account-id>.r2.cloudflarestorage.com
TRENOVA_STORAGE_BUCKET=<r2-bucket-name>
TRENOVA_STORAGE_ACCESSKEY=<r2-access-key-id>
TRENOVA_STORAGE_SECRETKEY=<r2-secret-access-key>
TRENOVA_STORAGE_REGION=auto
TRENOVA_STORAGE_USESSL=true
TRENOVA_STORAGE_AUTOCREATEBUCKET=false
TRENOVA_STORAGE_PUBLICENDPOINT=

TRENOVA_MEILI_MASTER_KEY=<generated-secret>
TRENOVA_GOOGLE_APIKEY=<google-api-key>
TRENOVA_SYSTEM_SYSTEMUSERPASSWORD=<generated-secret>

CLOUDFLARE_TUNNEL_TOKEN=<cloudflare-tunnel-token>
```

Keep `TRENOVA_SECURITY_SESSION_DOMAIN` empty while the app uses the `__Host-trenova_session` cookie name.

## 8. Start The API Stack

```bash
cd /opt/trenova/deploy
export TRENOVA_VERSION=latest
docker compose -f docker-compose.api.yml pull
docker compose -f docker-compose.api.yml up -d
```

Check status:

```bash
docker compose -f docker-compose.api.yml ps
docker compose -f docker-compose.api.yml logs -f tms-api
```

Check the public API:

```bash
curl -i https://api.trenova.app/api/v1/health
```

If that route does not exist in the current API build, verify that the container is listening:

```bash
docker compose -f docker-compose.api.yml exec tms-api nc -zv 127.0.0.1 8080
```

## 9. Maintenance Commands

Update containers:

```bash
cd /opt/trenova/deploy
export TRENOVA_VERSION=latest
docker compose -f docker-compose.api.yml pull
docker compose -f docker-compose.api.yml up -d
```

View logs:

```bash
docker compose -f docker-compose.api.yml logs -f tms-api
docker compose -f docker-compose.api.yml logs -f tms-worker
docker compose -f docker-compose.api.yml logs -f cloudflared
```

Open a database shell from inside the Docker network:

```bash
cd /opt/trenova/deploy
set -a
. ./.env
set +a

docker compose -f docker-compose.api.yml exec postgres psql \
  -U "$TRENOVA_DATABASE_USER" \
  -d "$TRENOVA_DATABASE_NAME"
```

Create a database backup:

```bash
cd /opt/trenova/deploy
set -a
. ./.env
set +a

mkdir -p /opt/trenova/backups
docker compose -f docker-compose.api.yml exec -T postgres pg_dump \
  -U "$TRENOVA_DATABASE_USER" \
  "$TRENOVA_DATABASE_NAME" \
  > "/opt/trenova/backups/trenova-$(date +%Y%m%d-%H%M%S).sql"
```

## 10. Final Security State

The final state should be:

- Hetzner firewall has no public inbound rules.
- `ufw` denies inbound by default.
- `ufw` allows inbound only on `tailscale0`.
- No Docker service publishes host ports.
- `api.trenova.app` reaches the API only through Cloudflare Tunnel.
- SSH works through Tailscale only.
