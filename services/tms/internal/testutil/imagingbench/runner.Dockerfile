# The container the Go imaging benchmark runs in: the same Debian release as the imaging
# worker's runtime, with the tesseract CLI the legacy OCR pipeline shells out to. The test
# binaries are built on the host with CGO_ENABLED=0 and mounted in; see `task imaging:bench-go`.
FROM debian:trixie-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends tesseract-ocr tesseract-ocr-eng ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /bench
