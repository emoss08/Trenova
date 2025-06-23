#!/bin/bash

set -e # Exit immediately if a command exits with a non-zero status

# Function to handle errors and cleanup
cleanup() {
	local exit_code=$?
	if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
		rm -rf "$TMP_DIR"
	fi

	# If we failed during installation and moved the original Go
	if [ $exit_code -ne 0 ] && [ -d "/usr/local/go_backup" ]; then
		echo "Error occurred. Restoring original Go installation..."
		sudo rm -rf /usr/local/go 2>/dev/null || true
		sudo mv /usr/local/go_backup /usr/local/go
		echo "Original Go installation restored."
	fi

	exit $exit_code
}

# Register the cleanup function for script exit
trap cleanup EXIT

# Get current Go version
if command -v go >/dev/null 2>&1; then
	CURRENT_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
	echo "Current Go version: $CURRENT_VERSION"
else
	echo "Go is not currently installed or not in PATH."
	CURRENT_VERSION="0.0.0"
fi

# Get latest Go version using the official download page with more robust parsing
echo "Fetching latest Go version..."
LATEST_VERSION=$(curl -s https://golang.org/dl/ | grep -o 'go[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1 | sed 's/go//')

# Verify we got a valid version
if [[ ! $LATEST_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "Failed to get latest version. Got: '$LATEST_VERSION'"
	echo "Using alternate method..."

	# Alternate method using the Go website API
	LATEST_VERSION=$(curl -s https://go.dev/VERSION?m=text | head -1 | sed 's/go//')

	if [[ ! $LATEST_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		echo "Still failed to get a valid version. Please check manually."
		exit 1
	fi
fi

echo "Latest Go version: $LATEST_VERSION"

# Compare versions (using version comparison function)
function version_lt() {
	[ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$1" ] && [ "$1" != "$2" ]
}

if version_lt "$CURRENT_VERSION" "$LATEST_VERSION"; then
	echo "Updating Go from $CURRENT_VERSION to $LATEST_VERSION..."

	# Define OS and architecture
	OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
	ARCH="$(uname -m)"
	if [ "$ARCH" = "x86_64" ]; then
		ARCH="amd64"
	elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
		ARCH="arm64"
	fi

	# Download URL
	DOWNLOAD_URL="https://go.dev/dl/go${LATEST_VERSION}.${OS}-${ARCH}.tar.gz"
	echo "Downloading from: $DOWNLOAD_URL"

	# Create temporary directory
	TMP_DIR=$(mktemp -d)

	# Download the latest version with progress bar
	echo "Downloading Go ${LATEST_VERSION}..."
	curl -L --progress-bar "$DOWNLOAD_URL" -o "$TMP_DIR/go.tar.gz"

	# Verify the download
	if [ ! -s "$TMP_DIR/go.tar.gz" ]; then
		echo "Download failed or file is empty."
		exit 1
	fi

	# Check if it's a valid gzip file
	if ! file "$TMP_DIR/go.tar.gz" | grep -q "gzip compressed data"; then
		echo "Downloaded file is not a valid gzip archive."
		exit 1
	fi

	# Stop if go binary is in use
	if command -v go >/dev/null 2>&1 && pgrep -f "$(which go)" >/dev/null; then
		echo "Go processes are running. Please close them and try again."
		exit 1
	fi

	# Backup current installation if it exists
	if [ -d "/usr/local/go" ]; then
		echo "Backing up current Go installation..."
		sudo mv /usr/local/go /usr/local/go_backup
	fi

	# Extract to /usr/local (requires sudo)
	echo "Installing new version..."
	sudo tar -C /usr/local -xzf "$TMP_DIR/go.tar.gz"

	# Make sure the binary is executable
	sudo chmod +x /usr/local/go/bin/go

	# Verify the installation
	if [ -f "/usr/local/go/bin/go" ]; then
		NEW_VERSION=$(/usr/local/go/bin/go version | awk '{print $3}' | sed 's/go//')
		if [[ "$NEW_VERSION" == "$LATEST_VERSION" ]]; then
			echo "Go successfully updated to version $NEW_VERSION"
			# Remove backup after successful update
			sudo rm -rf "/usr/local/go_backup"

			# Remind about PATH
			if ! command -v go >/dev/null 2>&1; then
				echo -e "\nNOTE: Go binary is not in your PATH."
				echo "Add the following to your shell profile:"
				echo "export PATH=\$PATH:/usr/local/go/bin"
			fi
		else
			echo "Version mismatch after installation. Got $NEW_VERSION, expected $LATEST_VERSION."
			exit 1
		fi
	else
		echo "Go binary not found after installation."
		exit 1
	fi
else
	echo "You already have the latest version of Go ($CURRENT_VERSION)"
fi

echo "Done!"
