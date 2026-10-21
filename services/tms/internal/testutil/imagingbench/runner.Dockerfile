# The container the TMS side of the imaging benchmark runs in: the same Debian release as the
# imaging worker's runtime. The test binary is built on the host with CGO_ENABLED=0 and
# mounted in; see `task imaging:bench`.
FROM debian:trixie-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /bench
