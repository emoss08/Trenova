#!/bin/bash
set -euo pipefail

# * Docker Hub repository name
DOCKER_REPO="wolfredstep/trenova"

# * Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# * Function to print colored output
print_status() {
    echo -e "${2}${1}${NC}"
}

# * Function to build and push image
build_and_push() {
    local dockerfile=$1
    local context=$2
    local image_name=$3
    local tag=${4:-latest}
    
    print_status "Building $image_name:$tag..." "$YELLOW"
    
    # * Build for multiple platforms
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --file "$dockerfile" \
        --tag "${DOCKER_REPO}-${image_name}:${tag}" \
        --tag "${DOCKER_REPO}-${image_name}:latest" \
        --push \
        "$context"
    
    if [ $? -eq 0 ]; then
        print_status "Successfully pushed ${DOCKER_REPO}-${image_name}:${tag}" "$GREEN"
    else
        print_status "Failed to build/push ${DOCKER_REPO}-${image_name}:${tag}" "$RED"
        exit 1
    fi
}

# * Main script
main() {
    print_status "Starting Docker Hub push process..." "$GREEN"
    
    # * Ensure buildx is available and create builder if needed
    if ! docker buildx ls | grep -q "trenova-builder"; then
        print_status "Creating buildx builder..." "$YELLOW"
        docker buildx create --name trenova-builder --use
        docker buildx inspect --bootstrap
    else
        docker buildx use trenova-builder
    fi
    
    # * Get version tag from git or use provided tag
    VERSION_TAG=${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "latest")}
    print_status "Using version tag: $VERSION_TAG" "$YELLOW"
    
    # * Build and push API image
    print_status "Building API image..." "$YELLOW"
    build_and_push "Dockerfile" "." "api" "$VERSION_TAG"
    
    # * Build and push UI image
    print_status "Building UI image..." "$YELLOW"
    build_and_push "ui/Dockerfile" "./ui" "ui" "$VERSION_TAG"
    
    # * Build and push Caddy image
    print_status "Building Caddy image..." "$YELLOW"
    build_and_push "Dockerfile.caddy" "." "caddy" "$VERSION_TAG"
    
    print_status "All images successfully pushed to Docker Hub!" "$GREEN"
    print_status "Images available at:" "$GREEN"
    print_status "  - ${DOCKER_REPO}-api:${VERSION_TAG}" "$GREEN"
    print_status "  - ${DOCKER_REPO}-ui:${VERSION_TAG}" "$GREEN"
    print_status "  - ${DOCKER_REPO}-caddy:${VERSION_TAG}" "$GREEN"
}

# * Run main function
main "$@"