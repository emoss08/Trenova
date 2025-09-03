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
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
CERTS_DIR="$PROJECT_DIR/certs"
DOMAIN="${DOMAIN:-localhost}"

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

# Create certificates directory
create_certs_dir() {
    print_info "Creating certificates directory..."
    mkdir -p "$CERTS_DIR"
}

# Generate self-signed certificates for development/local testing
generate_self_signed_certs() {
    print_info "Generating self-signed certificates for domain: $DOMAIN"
    
    # Generate private key
    openssl genrsa -out "$CERTS_DIR/key.pem" 2048
    
    # Generate certificate signing request
    openssl req -new -key "$CERTS_DIR/key.pem" -out "$CERTS_DIR/cert.csr" \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=$DOMAIN"
    
    # Create extensions file for Subject Alternative Names
    cat > "$CERTS_DIR/cert.ext" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = *.$DOMAIN
DNS.3 = localhost
DNS.4 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

    # Generate self-signed certificate
    openssl x509 -req -in "$CERTS_DIR/cert.csr" -signkey "$CERTS_DIR/key.pem" \
        -out "$CERTS_DIR/cert.pem" -days 365 \
        -extfile "$CERTS_DIR/cert.ext"
    
    # Clean up
    rm "$CERTS_DIR/cert.csr" "$CERTS_DIR/cert.ext"
    
    # Set proper permissions
    chmod 644 "$CERTS_DIR/cert.pem"
    chmod 600 "$CERTS_DIR/key.pem"
    
    print_success "Self-signed certificates generated successfully"
    print_warning "These certificates are for development/testing only!"
    print_warning "For production, use proper SSL certificates from a CA or Let's Encrypt"
}

# Generate DH parameters for better security
generate_dhparam() {
    print_info "Generating DH parameters (this may take a while)..."
    openssl dhparam -out "$CERTS_DIR/dhparam.pem" 2048
    chmod 644 "$CERTS_DIR/dhparam.pem"
    print_success "DH parameters generated"
}

# Setup Let's Encrypt with Certbot (for production use)
setup_letsencrypt() {
    print_info "Setting up Let's Encrypt certificate generation..."
    
    if [ -z "$DOMAIN" ] || [ "$DOMAIN" = "localhost" ]; then
        print_error "Please set a valid domain name in DOMAIN environment variable"
        print_error "Example: DOMAIN=yourdomain.com $0 --letsencrypt"
        exit 1
    fi
    
    print_info "Creating directories for Certbot..."
    mkdir -p "$PROJECT_DIR/certbot/www"
    mkdir -p "$PROJECT_DIR/certbot/conf"
    
    print_info "Requesting SSL certificate for $DOMAIN..."
    
    # Use Docker to run Certbot
    docker run -it --rm \
        -v "$PROJECT_DIR/certbot/conf:/etc/letsencrypt" \
        -v "$PROJECT_DIR/certbot/www:/var/www/certbot" \
        certbot/certbot certonly \
        --webroot \
        --webroot-path=/var/www/certbot \
        --email "${ACME_EMAIL:-admin@$DOMAIN}" \
        --agree-tos \
        --no-eff-email \
        -d "$DOMAIN" \
        -d "www.$DOMAIN"
    
    if [ $? -eq 0 ]; then
        # Copy certificates to our certs directory
        cp "$PROJECT_DIR/certbot/conf/live/$DOMAIN/fullchain.pem" "$CERTS_DIR/cert.pem"
        cp "$PROJECT_DIR/certbot/conf/live/$DOMAIN/privkey.pem" "$CERTS_DIR/key.pem"
        
        print_success "Let's Encrypt certificates obtained successfully"
        print_info "Certificates are located in: $CERTS_DIR"
    else
        print_error "Failed to obtain Let's Encrypt certificates"
        exit 1
    fi
}

# Create certificate renewal script
create_renewal_script() {
    print_info "Creating certificate renewal script..."
    
    cat > "$PROJECT_DIR/scripts/renew-certs.sh" << 'EOF'
#!/bin/bash

# Renew Let's Encrypt certificates
docker run --rm \
    -v "$(pwd)/certbot/conf:/etc/letsencrypt" \
    -v "$(pwd)/certbot/www:/var/www/certbot" \
    certbot/certbot renew

# Reload nginx/traefik after renewal
if [ "$1" = "nginx" ]; then
    docker-compose -f docker-compose-prod.yml -f docker-compose-prod.nginx.yml exec tren-nginx nginx -s reload
elif [ "$1" = "traefik" ]; then
    docker-compose -f docker-compose-prod.yml -f docker-compose-prod.traefik.yml restart tren-traefik
fi

echo "Certificate renewal completed"
EOF

    chmod +x "$PROJECT_DIR/scripts/renew-certs.sh"
    
    print_success "Certificate renewal script created at scripts/renew-certs.sh"
    print_info "Add to crontab for automatic renewal:"
    print_info "0 12 * * * $PROJECT_DIR/scripts/renew-certs.sh nginx"
}

# Verify certificates
verify_certificates() {
    if [ ! -f "$CERTS_DIR/cert.pem" ] || [ ! -f "$CERTS_DIR/key.pem" ]; then
        print_error "Certificate files not found in $CERTS_DIR"
        return 1
    fi
    
    print_info "Verifying certificates..."
    
    # Check certificate validity
    if openssl x509 -in "$CERTS_DIR/cert.pem" -text -noout > /dev/null 2>&1; then
        print_success "Certificate is valid"
        
        # Show certificate details
        print_info "Certificate details:"
        openssl x509 -in "$CERTS_DIR/cert.pem" -text -noout | grep -E "(Subject:|Issuer:|Not Before:|Not After:|DNS:|IP Address:)"
    else
        print_error "Certificate is invalid"
        return 1
    fi
    
    # Check private key
    if openssl rsa -in "$CERTS_DIR/key.pem" -check > /dev/null 2>&1; then
        print_success "Private key is valid"
    else
        print_error "Private key is invalid"
        return 1
    fi
    
    # Verify key matches certificate
    cert_hash=$(openssl x509 -noout -modulus -in "$CERTS_DIR/cert.pem" | openssl md5)
    key_hash=$(openssl rsa -noout -modulus -in "$CERTS_DIR/key.pem" | openssl md5)
    
    if [ "$cert_hash" = "$key_hash" ]; then
        print_success "Certificate and private key match"
    else
        print_error "Certificate and private key do not match"
        return 1
    fi
}

# Main function
main() {
    case "${1:-}" in
        --letsencrypt|--le)
            create_certs_dir
            setup_letsencrypt
            create_renewal_script
            verify_certificates
            ;;
        --verify)
            verify_certificates
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Generate SSL certificates for Trenova TMS production deployment"
            echo ""
            echo "Options:"
            echo "  --letsencrypt, --le    Generate Let's Encrypt certificates (requires valid domain)"
            echo "  --verify              Verify existing certificates"
            echo "  --help, -h            Show this help message"
            echo ""
            echo "Default (no options):   Generate self-signed certificates for development"
            echo ""
            echo "Environment Variables:"
            echo "  DOMAIN                Domain name for certificates (required for Let's Encrypt)"
            echo "  ACME_EMAIL           Email for Let's Encrypt registration"
            echo ""
            echo "Examples:"
            echo "  $0                                    # Generate self-signed certs"
            echo "  DOMAIN=example.com $0 --letsencrypt  # Generate Let's Encrypt certs"
            echo "  $0 --verify                          # Verify existing certificates"
            ;;
        *)
            create_certs_dir
            generate_self_signed_certs
            generate_dhparam
            verify_certificates
            ;;
    esac
}

# Run main function
main "$@"