#!/bin/bash
##
## Copyright 2023-2025 Eric Moss
## Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
## Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md##

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_DIR/.env.production"
REVERSE_PROXY="${REVERSE_PROXY:-nginx}"  # nginx or traefik

print_banner() {
    echo -e "${PURPLE}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║                                                      ║"
    echo "║                 TRENOVA TMS                          ║"
    echo "║              Production Deployment                   ║"
    echo "║                                                      ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${PURPLE}[STEP]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."
    
    # Check if Docker is installed and running
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    # Check if Docker Compose is available
    if ! docker compose version &> /dev/null && ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not available. Please install Docker Compose."
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Setup environment file
setup_environment() {
    print_step "Setting up environment configuration..."
    
    if [ ! -f "$ENV_FILE" ]; then
        if [ -f "$PROJECT_DIR/.env.production.example" ]; then
            print_info "Creating .env.production from example file..."
            cp "$PROJECT_DIR/.env.production.example" "$ENV_FILE"
            
            print_warning "Please edit .env.production with your configuration:"
            print_info "  - Set your domain name (DOMAIN)"
            print_info "  - Configure database passwords"
            print_info "  - Set Redis password"
            print_info "  - Configure MinIO credentials"
            print_info "  - Set API keys for AI services (optional)"
            print_info "  - Configure email settings"
            
            read -p "Press Enter after you have configured .env.production..."
        else
            print_error ".env.production.example not found. Please ensure all files are in place."
            exit 1
        fi
    else
        print_info "Using existing .env.production file"
    fi
    
    # Validate critical environment variables
    if ! grep -q "^DOMAIN=" "$ENV_FILE" || grep -q "^DOMAIN=yourdomain.com" "$ENV_FILE"; then
        print_error "Please set a valid DOMAIN in .env.production"
        exit 1
    fi
    
    print_success "Environment configuration ready"
}

# Generate SSL certificates
setup_ssl() {
    print_step "Setting up SSL certificates..."
    
    # Source environment file to get DOMAIN
    export $(grep -v '^#' "$ENV_FILE" | xargs)
    
    if [ ! -f "$PROJECT_DIR/certs/cert.pem" ]; then
        print_info "SSL certificates not found. Generating certificates..."
        
        if [ "$DOMAIN" = "localhost" ] || [[ "$DOMAIN" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            print_info "Generating self-signed certificates for development/testing..."
            "$SCRIPT_DIR/generate-certs.sh"
        else
            print_info "Domain detected: $DOMAIN"
            read -p "Do you want to generate Let's Encrypt certificates? (y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                "$SCRIPT_DIR/generate-certs.sh" --letsencrypt
            else
                print_info "Generating self-signed certificates..."
                "$SCRIPT_DIR/generate-certs.sh"
            fi
        fi
    else
        print_info "SSL certificates already exist"
        "$SCRIPT_DIR/generate-certs.sh" --verify
    fi
    
    print_success "SSL certificates ready"
}

# Build Docker images
build_images() {
    print_step "Building Docker images..."
    
    cd "$PROJECT_DIR"
    
    # Determine compose files to use
    COMPOSE_FILES="-f docker-compose-prod.yml"
    
    if [ "$REVERSE_PROXY" = "nginx" ]; then
        COMPOSE_FILES="$COMPOSE_FILES -f docker-compose-prod.nginx.yml"
        print_info "Building images for Nginx deployment..."
    elif [ "$REVERSE_PROXY" = "traefik" ]; then
        COMPOSE_FILES="$COMPOSE_FILES -f docker-compose-prod.traefik.yml"
        print_info "Building images for Traefik deployment..."
    fi
    
    # Load environment variables
    export $(grep -v '^#' "$ENV_FILE" | xargs)
    
    # Build all images using docker-compose
    print_info "Building Docker images..."
    docker-compose $COMPOSE_FILES build --parallel
    
    print_success "Docker images built successfully"
}

# Create necessary directories
create_directories() {
    print_step "Creating necessary directories..."
    
    mkdir -p "$PROJECT_DIR/logs/nginx" 2>/dev/null || true
    mkdir -p "$PROJECT_DIR/logs/traefik" 2>/dev/null || true
    mkdir -p "$PROJECT_DIR/logs/api" 2>/dev/null || true
    mkdir -p "$PROJECT_DIR/certbot/www" 2>/dev/null || true
    mkdir -p "$PROJECT_DIR/certbot/conf" 2>/dev/null || true
    mkdir -p "$PROJECT_DIR/backups" 2>/dev/null || true
    
    print_success "Directories created"
}

# Deploy the application
deploy_application() {
    print_step "Deploying Trenova TMS..."
    
    cd "$PROJECT_DIR"
    
    # Determine compose files to use
    COMPOSE_FILES="-f docker-compose-prod.yml"
    
    if [ "$REVERSE_PROXY" = "nginx" ]; then
        COMPOSE_FILES="$COMPOSE_FILES -f docker-compose-prod.nginx.yml"
        print_info "Using Nginx as reverse proxy"
    elif [ "$REVERSE_PROXY" = "traefik" ]; then
        COMPOSE_FILES="$COMPOSE_FILES -f docker-compose-prod.traefik.yml"
        print_info "Using Traefik as reverse proxy"
    else
        print_error "Invalid reverse proxy choice: $REVERSE_PROXY"
        print_info "Valid options: nginx, traefik"
        exit 1
    fi
    
    # Load environment variables
    export $(grep -v '^#' "$ENV_FILE" | xargs)
    
    # Stop any existing containers
    print_info "Stopping existing containers..."
    docker compose $COMPOSE_FILES down --remove-orphans 2>/dev/null || true
    
    # Start the services
    print_info "Starting services..."
    docker compose $COMPOSE_FILES up -d
    
    print_success "Application deployed successfully"
}

# Wait for services to be healthy
wait_for_services() {
    print_step "Waiting for services to be healthy..."
    
    local max_attempts=60
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        print_info "Health check attempt $attempt/$max_attempts..."
        
        # Check if all services are healthy
        if docker compose -f docker-compose-prod.yml ps --format json | jq -r '.[].Health' | grep -q "unhealthy"; then
            print_info "Some services are still starting up..."
            sleep 10
            ((attempt++))
        else
            print_success "All services are healthy!"
            return 0
        fi
    done
    
    print_warning "Some services may not be fully ready yet. Check logs if needed."
}

# Display deployment information
show_deployment_info() {
    print_step "Deployment Information"
    
    # Load environment variables
    export $(grep -v '^#' "$ENV_FILE" | xargs)
    
    echo
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                 DEPLOYMENT COMPLETE                  ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo
    
    if [ "$REVERSE_PROXY" = "nginx" ]; then
        echo -e "${BLUE}Application URL:${NC} https://$DOMAIN"
        echo -e "${BLUE}HTTP Port:${NC} ${HTTP_PORT:-80}"
        echo -e "${BLUE}HTTPS Port:${NC} ${HTTPS_PORT:-443}"
    elif [ "$REVERSE_PROXY" = "traefik" ]; then
        echo -e "${BLUE}Application URL:${NC} https://$DOMAIN"
        echo -e "${BLUE}Traefik Dashboard:${NC} https://traefik.$DOMAIN"
        echo -e "${BLUE}HTTP Port:${NC} ${HTTP_PORT:-80}"
        echo -e "${BLUE}HTTPS Port:${NC} ${HTTPS_PORT:-443}"
        echo -e "${BLUE}Metrics Port:${NC} ${METRICS_PORT:-8082}"
    fi
    
    echo -e "${BLUE}API Port:${NC} ${API_PORT:-13001}"
    echo -e "${BLUE}UI Port:${NC} ${UI_PORT:-15173}"
    echo
    
    echo -e "${YELLOW}Next Steps:${NC}"
    echo -e "${YELLOW}1.${NC} Configure your DNS to point to this server"
    echo -e "${YELLOW}2.${NC} Ensure firewall allows traffic on ports 80 and 443"
    echo -e "${YELLOW}3.${NC} Monitor logs: docker compose logs -f"
    echo -e "${YELLOW}4.${NC} Access the application at https://$DOMAIN"
    echo
    
    echo -e "${YELLOW}Management Commands:${NC}"
    echo -e "${YELLOW}•${NC} View logs: docker compose $COMPOSE_FILES logs -f [service_name]"
    echo -e "${YELLOW}•${NC} Scale services: docker compose $COMPOSE_FILES up -d --scale tren-api=2"
    echo -e "${YELLOW}•${NC} Update: docker compose $COMPOSE_FILES pull && docker compose $COMPOSE_FILES up -d"
    echo -e "${YELLOW}•${NC} Backup database: ./scripts/backup-db.sh"
    echo -e "${YELLOW}•${NC} Stop services: docker compose $COMPOSE_FILES down"
    echo
}

# Cleanup function for script interruption
cleanup() {
    print_warning "Deployment interrupted. Cleaning up..."
    cd "$PROJECT_DIR" 2>/dev/null || true
    docker compose -f docker-compose-prod.yml -f docker-compose-prod.nginx.yml down --remove-orphans 2>/dev/null || true
    docker compose -f docker-compose-prod.yml -f docker-compose-prod.traefik.yml down --remove-orphans 2>/dev/null || true
    exit 1
}

# Set trap for cleanup on script interruption
trap cleanup SIGINT SIGTERM

# Main deployment function
main() {
    print_banner
    
    # Change to project directory
    cd "$PROJECT_DIR"
    
    case "${1:-}" in
        --nginx)
            REVERSE_PROXY="nginx"
            ;;
        --traefik)
            REVERSE_PROXY="traefik"
            ;;
        --help|-h)
            echo "Trenova TMS Production Deployment Script"
            echo
            echo "Usage: $0 [OPTIONS]"
            echo
            echo "Options:"
            echo "  --nginx     Deploy with Nginx as reverse proxy (default)"
            echo "  --traefik   Deploy with Traefik as reverse proxy"
            echo "  --help, -h  Show this help message"
            echo
            echo "Environment Variables:"
            echo "  REVERSE_PROXY    nginx|traefik (default: nginx)"
            echo
            echo "Examples:"
            echo "  $0 --nginx     # Deploy with Nginx"
            echo "  $0 --traefik   # Deploy with Traefik"
            echo
            exit 0
            ;;
    esac
    
    print_info "Using $REVERSE_PROXY as reverse proxy"
    
    # Execute deployment steps
    check_prerequisites
    setup_environment
    create_directories
    setup_ssl
    build_images
    deploy_application
    wait_for_services
    show_deployment_info
    
    print_success "Trenova TMS deployment completed successfully!"
}

# Run main function with all arguments
main "$@"