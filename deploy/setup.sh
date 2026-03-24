#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[trenova]${NC} $1"; }
warn() { echo -e "${YELLOW}[trenova]${NC} $1"; }
err() { echo -e "${RED}[trenova]${NC} $1" >&2; }
info() { echo -e "${BLUE}[trenova]${NC} $1"; }

check_prerequisites() {
    local missing=0

    if ! command -v docker &>/dev/null; then
        err "Docker is not installed"
        missing=1
    fi

    if ! docker compose version &>/dev/null; then
        err "Docker Compose v2 is not available"
        missing=1
    fi

    if ! command -v openssl &>/dev/null; then
        err "openssl is not installed"
        missing=1
    fi

    if [ $missing -ne 0 ]; then
        exit 1
    fi

    log "Prerequisites satisfied"
}

generate_secret() {
    openssl rand -base64 "$1" | tr -d '/+=\n' | head -c "$1"
}

ensure_env_var() {
    local key=$1
    local value=$2

    if ! grep -q "^${key}=" .env; then
        if [ -n "$(tail -c1 .env)" ]; then
            echo "" >> .env
        fi
        echo "${key}=${value}" >> .env
        log "Added missing ${key} to .env"
    fi
}

create_env() {
    if [ ! -f .env ]; then
        local domain
        read -rp "Enter your domain (or 'localhost' for local testing): " domain
        domain="${domain:-localhost}"

        cat > .env <<EOF
DOMAIN=${domain}
VITE_API_URL=https://${domain}/api/v1

TRENOVA_DATABASE_NAME=trenova_db
TRENOVA_DATABASE_USER=trenova
TRENOVA_DATABASE_PASSWORD=$(generate_secret 32)
TRENOVA_SECURITY_SESSION_SECRET=$(generate_secret 48)
TRENOVA_SECURITY_ENCRYPTION_KEY=$(generate_secret 48)
TRENOVA_SECURITY_SESSION_DOMAIN=${domain}
TRENOVA_STORAGE_PUBLICENDPOINT=https://storage.${domain}

TRENOVA_STORAGE_ACCESSKEY=trenova-minio
TRENOVA_STORAGE_SECRETKEY=$(generate_secret 32)
TRENOVA_MEILI_MASTER_KEY=$(generate_secret 32)
EOF

        chmod 600 .env
        log "Generated .env with cryptographic secrets"
    else
        warn ".env already exists, validating required variables"
    fi

    ensure_env_var "TRENOVA_GOOGLE_APIKEY" "$(generate_secret 40)"
    ensure_env_var "TRENOVA_SYSTEM_SYSTEMUSERPASSWORD" "$(generate_secret 32)"
}

wait_healthy() {
    local service=$1
    local max_attempts=${2:-60}
    local attempt=0

    log "Waiting for ${service} to become healthy..."
    while [ $attempt -lt $max_attempts ]; do
        if docker compose ps "$service" 2>/dev/null | grep -q "healthy"; then
            log "${service} is healthy"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    err "${service} did not become healthy within $((max_attempts * 2))s"
    docker compose logs --tail=20 "$service"
    return 1
}

is_wsl() {
    grep -qiE "(microsoft|wsl)" /proc/version 2>/dev/null
}

install_cert_linux() {
    local cert_path=$1

    if [ -d /usr/local/share/ca-certificates ]; then
        log "Installing root CA into system trust store..."
        if sudo cp "$cert_path" /usr/local/share/ca-certificates/caddy-root-ca.crt &&
            sudo update-ca-certificates; then
            log "Root CA installed (Debian/Ubuntu)"
        else
            warn "Could not auto-install root CA on Debian/Ubuntu (sudo may require interactive auth)."
            warn "Manually run: sudo cp ${cert_path} /usr/local/share/ca-certificates/caddy-root-ca.crt && sudo update-ca-certificates"
        fi
    elif command -v trust &>/dev/null; then
        log "Installing root CA into system trust store..."
        if sudo trust anchor --store "$cert_path"; then
            log "Root CA installed (Fedora/RHEL)"
        else
            warn "Could not auto-install root CA on Fedora/RHEL (sudo may require interactive auth)."
            warn "Manually run: sudo trust anchor --store ${cert_path}"
        fi
    else
        warn "Could not auto-install. Manually add ${cert_path} to your system trust store."
    fi

    install_cert_nssdb "$cert_path"
}

install_cert_nssdb() {
    local cert_path=$1

    if ! command -v certutil &>/dev/null; then
        if command -v apt-get &>/dev/null; then
            log "Installing libnss3-tools (required for Chrome/Firefox cert trust)..."
            sudo apt-get install -y libnss3-tools >/dev/null 2>&1 || true
        elif command -v dnf &>/dev/null; then
            log "Installing nss-tools (required for Chrome/Firefox cert trust)..."
            sudo dnf install -y nss-tools >/dev/null 2>&1 || true
        fi
    fi

    if ! command -v certutil &>/dev/null; then
        warn "certutil not available. Chrome/Firefox may not trust the certificate."
        warn "Install libnss3-tools (Debian/Ubuntu) or nss-tools (Fedora/RHEL) and run:"
        echo "  certutil -d sql:\$HOME/.pki/nssdb -A -t \"C,,\" -n \"Caddy Root CA\" -i ${cert_path}"
        return
    fi

    if [ -d "$HOME/.pki/nssdb" ]; then
        certutil -d "sql:$HOME/.pki/nssdb" -D -n "Caddy Root CA" 2>/dev/null || true
        certutil -d "sql:$HOME/.pki/nssdb" -A -t "C,," -n "Caddy Root CA" -i "$cert_path"
        log "Root CA installed into Chrome/Firefox NSS database"
    else
        mkdir -p "$HOME/.pki/nssdb"
        certutil -d "sql:$HOME/.pki/nssdb" -N --empty-password
        certutil -d "sql:$HOME/.pki/nssdb" -A -t "C,," -n "Caddy Root CA" -i "$cert_path"
        log "Created NSS database and installed root CA for Chrome/Firefox"
    fi
}

install_cert_wsl() {
    local cert_path=$1
    local wsl_distro
    local wsl_cert_path

    wsl_distro="${WSL_DISTRO_NAME:-}"
    if [ -z "$wsl_distro" ]; then
        wsl_distro=$(grep "^ID=" /etc/os-release 2>/dev/null | cut -d= -f2 | tr -d '"')
        wsl_distro="${wsl_distro^}"
    fi
    wsl_cert_path=$(wslpath -w "$cert_path" 2>/dev/null) || wsl_cert_path="\\\\wsl\$\\${wsl_distro}${cert_path//\//\\}"

    echo ""
    warn "IMPORTANT: You must install the root CA on Windows for browsers to trust it."
    echo ""
    echo "  Run this in PowerShell as Administrator:"
    echo ""
    echo "    Import-Certificate -FilePath \"${wsl_cert_path}\" -CertStoreLocation Cert:\\LocalMachine\\Root"
    echo ""
    echo "  Then fully quit Chrome (right-click system tray icon -> Exit) and reopen it."
    echo ""
}

install_cert_macos() {
    local cert_path=$1
    log "Installing root CA into macOS system keychain..."
    if sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$cert_path"; then
        log "Root CA installed (macOS)"
    else
        warn "Could not auto-install root CA on macOS (sudo may require interactive auth)."
        warn "Manually run: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ${cert_path}"
    fi
}

print_org_distribution_instructions() {
    echo ""
    info "=== Distributing the Root CA to Other Machines ==="
    echo ""
    echo "  Trenova uses a self-signed TLS certificate. Every machine that"
    echo "  accesses Trenova must trust the root CA: ${SCRIPT_DIR}/caddy-root-ca.crt"
    echo ""
    echo "  Windows (single machine, PowerShell as Administrator):"
    echo "    Import-Certificate -FilePath \"caddy-root-ca.crt\" -CertStoreLocation Cert:\\LocalMachine\\Root"
    echo ""
    echo "  Windows (Group Policy - recommended for organizations):"
    echo "    1. Open Group Policy Management (gpmc.msc)"
    echo "    2. Edit a GPO linked to the target OU"
    echo "    3. Computer Configuration -> Policies -> Windows Settings"
    echo "       -> Security Settings -> Public Key Policies"
    echo "       -> Trusted Root Certification Authorities"
    echo "    4. Right-click -> Import -> select caddy-root-ca.crt"
    echo "    5. Machines will pick it up on next gpupdate"
    echo ""
    echo "  macOS (MDM/Profile):"
    echo "    Deploy the .crt as a configuration profile via MDM (Jamf, Mosyle, etc.)"
    echo "    Or manually: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain caddy-root-ca.crt"
    echo ""
    echo "  Linux (Debian/Ubuntu):"
    echo "    sudo cp caddy-root-ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates"
    echo "    sudo apt-get install -y libnss3-tools"
    echo "    certutil -d sql:\$HOME/.pki/nssdb -A -t \"C,,\" -n \"Caddy Root CA\" -i caddy-root-ca.crt"
    echo ""
    echo "  Linux (Fedora/RHEL):"
    echo "    sudo trust anchor --store caddy-root-ca.crt"
    echo "    sudo dnf install -y nss-tools"
    echo "    certutil -d sql:\$HOME/.pki/nssdb -A -t \"C,,\" -n \"Caddy Root CA\" -i caddy-root-ca.crt"
    echo ""
    echo "  NOTE: Chrome and Firefox on Linux use their own certificate database (NSS)."
    echo "  The certutil commands above are required for browsers to trust the certificate."
    echo ""
}

extract_and_install_cert() {
    local max_attempts=15
    local attempt=0

    log "Extracting Caddy root CA certificate..."
    while [ $attempt -lt $max_attempts ]; do
        if docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-root-ca.crt 2>/dev/null; then
            log "Caddy root CA saved to ${SCRIPT_DIR}/caddy-root-ca.crt"

            if is_wsl; then
                install_cert_wsl "${SCRIPT_DIR}/caddy-root-ca.crt"
                install_cert_linux "${SCRIPT_DIR}/caddy-root-ca.crt"
            elif [ "$(uname)" = "Darwin" ]; then
                install_cert_macos "${SCRIPT_DIR}/caddy-root-ca.crt"
            else
                install_cert_linux "${SCRIPT_DIR}/caddy-root-ca.crt"
            fi

            print_org_distribution_instructions

            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    warn "Could not extract Caddy root CA certificate (Caddy may not have generated it yet)"
    warn "You can manually extract it later:"
    echo "  docker compose cp caddy:/data/caddy/pki/authorities/local/root.crt ./caddy-root-ca.crt"
}

create_minio_bucket() {
    log "Creating MinIO bucket..."
    docker compose exec minio mc alias set local http://localhost:9000 "$TRENOVA_STORAGE_ACCESSKEY" "$TRENOVA_STORAGE_SECRETKEY" 2>/dev/null || true
    docker compose exec minio mc mb local/trenova --ignore-existing 2>/dev/null || true
}

main() {
    if [ "${1:-}" = "--reset" ]; then
        warn "Resetting deployment (destroying all data)..."
        docker compose down -v 2>/dev/null || true
        rm -f .env
        log "Reset complete, re-running setup..."
        echo ""
    fi

    log "Trenova Self-Hosted Setup"
    echo ""

    check_prerequisites
    create_env

    source .env

    log "Building Docker images..."
    docker compose build

    log "Starting infrastructure services..."
    docker compose up -d postgres pgbouncer redis minio meilisearch

    wait_healthy postgres
    wait_healthy pgbouncer
    wait_healthy redis
    wait_healthy minio
    wait_healthy meilisearch

    log "Starting Temporal..."
    docker compose up -d temporal
    wait_healthy temporal 90

    log "Starting Temporal UI..."
    docker compose up -d temporal-ui

    log "Running database migrations and seeding..."
    docker compose up tms-migrate

    create_minio_bucket

    log "Starting application services..."
    docker compose up -d tms-api tms-worker caddy

    wait_healthy tms-api

    extract_and_install_cert

    echo ""
    log "Trenova is running!"
    echo ""

    log "Frontend:    https://${DOMAIN}"
    log "API:         https://${DOMAIN}/api/v1/health"
    log "Temporal UI: https://${DOMAIN}/temporal-ui"
    log "MinIO:       https://${DOMAIN}/minio"

    echo ""
    log "Credentials are stored in: ${SCRIPT_DIR}/.env"
}

main "$@"
