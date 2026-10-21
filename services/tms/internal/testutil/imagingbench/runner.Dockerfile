# The container the Go imaging benchmark runs in: the same Debian release as the imaging
# worker's runtime. The test binaries are built on the host with CGO_ENABLED=0 and mounted
# in; see `task imaging:bench-go`.
FROM debian:trixie-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /bench
