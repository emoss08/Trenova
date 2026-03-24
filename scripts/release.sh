#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_error() { echo -e "${RED}Error: $1${NC}" >&2; }
print_success() { echo -e "${GREEN}$1${NC}"; }
print_warning() { echo -e "${YELLOW}$1${NC}"; }

usage() {
    echo "Usage: $0 [version]"
    echo ""
    echo "Creates a new release for Trenova."
    echo ""
    echo "Arguments:"
    echo "  version    Version to release (e.g., 1.0.0 or v1.0.0)"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -d, --dry-run  Show what would be done without making changes"
    echo ""
    echo "Examples:"
    echo "  $0 1.0.0"
    echo "  $0 v1.2.0"
    echo "  $0 --dry-run 1.0.0"
    exit 0
}

DRY_RUN=false
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            VERSION="$1"
            shift
            ;;
    esac
done

if ! command -v gh &> /dev/null; then
    print_error "GitHub CLI (gh) is not installed. Install it from https://cli.github.com/"
    exit 1
fi

if ! gh auth status &> /dev/null; then
    print_error "Not authenticated with GitHub CLI. Run 'gh auth login' first."
    exit 1
fi

if [ -z "$VERSION" ]; then
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    echo "Latest release: $LATEST_TAG"
    echo ""
    read -p "Enter version to release (e.g., 1.0.0): " VERSION
fi

if [ -z "$VERSION" ]; then
    print_error "Version is required"
    exit 1
fi

if [[ "$VERSION" != v* ]]; then
    VERSION="v$VERSION"
fi

if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    print_error "Invalid version format. Use semantic versioning (e.g., 1.0.0, 1.0.0-beta.1)"
    exit 1
fi

if git rev-parse "$VERSION" &> /dev/null; then
    print_error "Tag $VERSION already exists"
    exit 1
fi

if [ -n "$(git status --porcelain)" ]; then
    print_warning "Warning: You have uncommitted changes"
    git status --short
    echo ""
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ] && [ "$CURRENT_BRANCH" != "master" ]; then
    print_warning "Warning: You are on branch '$CURRENT_BRANCH', not main/master"
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo ""
echo "Release Summary"
echo "==============="
echo "Version:  $VERSION"
echo "Branch:   $CURRENT_BRANCH"
echo "Commit:   $(git rev-parse --short HEAD)"
echo ""

PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -n "$PREV_TAG" ]; then
    echo "Changes since $PREV_TAG:"
    git log --oneline "$PREV_TAG"..HEAD | head -20
    COMMIT_COUNT=$(git rev-list --count "$PREV_TAG"..HEAD)
    if [ "$COMMIT_COUNT" -gt 20 ]; then
        echo "... and $((COMMIT_COUNT - 20)) more commits"
    fi
else
    echo "Changes (first release):"
    git log --oneline | head -20
fi
echo ""

if [ "$DRY_RUN" = true ]; then
    print_warning "DRY RUN - No changes will be made"
    echo ""
    echo "Would execute:"
    echo "  git tag -a $VERSION -m \"Release $VERSION\""
    echo "  git push origin $VERSION"
    echo "  gh release create $VERSION --generate-notes"
    exit 0
fi

read -p "Create release $VERSION? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 1
fi

echo ""
echo "Creating tag $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

echo "Pushing tag to origin..."
git push origin "$VERSION"

echo "Creating GitHub release..."
gh release create "$VERSION" \
    --generate-notes \
    --title "$VERSION"

echo ""
print_success "Release $VERSION created successfully!"
echo ""
echo "GitHub Actions is now building and pushing Docker images to ghcr.io"
echo "Monitor progress at: $(gh repo view --json url -q .url)/actions"
echo ""
echo "Once complete, users can update with:"
echo "  trenova update apply ${VERSION#v}"
