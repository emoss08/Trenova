# Document Pipeline

This document describes the current document pipeline in `services/tms`, the runtime dependencies it relies on, and where `GTC` still matters today.

## Scope

This covers:

- document upload
- upload-session lifecycle
- thumbnail generation
- OCR and document intelligence
- extracted-content search
- Meilisearch indexing

## Current architecture

The document system is split into a few separate stages.

### 1. Upload and persistence

The upload path is owned by TMS.

- The browser creates an upload session through TMS.
- Large files upload directly to object storage using multipart upload.
- TMS finalizes the upload and creates the `documents` row.
- `document_upload_sessions` tracks in-progress and completed upload state.

Primary dependencies:

- Postgres
- MinIO / S3-compatible storage
- Redis
- Temporal worker

`GTC` is not required for upload correctness.

### 2. Thumbnail generation

Thumbnail generation is asynchronous.

- TMS creates the document row.
- A Temporal workflow is started for thumbnail generation when the file type supports previews.
- The preview is written back to storage and the document preview fields are updated.

Primary dependencies:

- Postgres
- MinIO / S3-compatible storage
- Temporal worker

`GTC` is not required for thumbnail generation.

### 3. OCR and document intelligence

Document intelligence is also asynchronous.

- After upload finalization, TMS starts intelligence extraction workflows.
- Native text extraction runs first.
- OCR is used for scanned PDFs and image uploads.
- Extracted content is stored in `document_contents`.
- Page-level extraction and OCR metadata are stored in `document_content_pages`.
- Structured extraction and shipment-draft candidates are derived from the canonical intelligence result.
- Document intelligence runs on its own Temporal task queue so OCR-heavy work does not compete with the shared system queue.
- OCR preprocessing runs before `tesseract` when enabled, but falls back to raw-image OCR if preprocessing fails.

Primary dependencies:

- Postgres
- MinIO / S3-compatible storage
- Temporal worker
- `tesseract` for OCR-capable environments

`GTC` is not required for OCR or document intelligence correctness.

### 4. Search indexing

This is the only part of the current document system that still depends on `GTC`.

Current flow:

- TMS writes canonical search data into `document_search_projections`.
- `GTC` tails Postgres changes and projects `document_search_projections` into Meilisearch.
- TMS search endpoints use Meilisearch first and can fall back to Postgres FTS when needed.

Primary dependencies:

- Postgres
- Meilisearch
- `GTC`

If `GTC` is down:

- document uploads still work
- thumbnail generation still works
- OCR and extraction still work
- `document_contents` still updates in Postgres
- Meilisearch document search becomes stale until GTC catches up or a backfill runs

## What depends on GTC today

`GTC` matters today only for keeping Meilisearch in sync from `document_search_projections`.

It does not control:

- upload completion
- upload-session recovery
- storage writes
- thumbnail generation
- OCR
- canonical extracted text persistence
- document classification
- shipment-draft generation

## Required runtime components

For the full current document feature set:

- `Postgres`
  Source of truth for documents, upload sessions, extracted content, and search projections.
- `MinIO` or other S3-compatible storage
  Stores original files and thumbnails.
- `Temporal server`
  Runs thumbnail, upload reconciliation, and document intelligence workflows.
- `Temporal worker`
  Required for workflow execution. If the worker is down, uploads may finalize partially and async jobs will stall.
- `Document intelligence worker`
  Dedicated lower-concurrency worker queue for OCR and extraction so CPU-heavy OCR does not saturate shared workers.
- `Redis`
  Used for upload coordination and lease-style workflow guards.
- `tesseract`
  Required only for OCR on scanned PDFs and images.
- OCR preprocessing
  Runs in-process inside the document-intelligence worker before `tesseract`; no separate service is required today.
- `Meilisearch`
  Used for fast document search.
- `GTC`
  Required only for Meilisearch sync in the current design.

## Failure modes

### Temporal worker down

Symptoms:

- previews stay pending
- upload sessions may remain in intermediate async states
- document intelligence does not progress

Core uploads can still complete, but async processing stalls.

### Tesseract missing

Symptoms:

- scanned/image documents fail extraction
- native-text documents still extract normally

### OCR saturation

Symptoms:

- document intelligence backs up while uploads and thumbnails remain healthy
- OCR-heavy scanned PDFs process more slowly than native-text documents

Current protection:

- document intelligence runs on a dedicated queue
- OCR subprocesses are bounded by document-intelligence worker concurrency
- each OCR subprocess has a hard timeout
- page-level OCR metadata is persisted so low-quality pages can be inspected without rerunning the entire document

### GTC down

Symptoms:

- search results in Meilisearch become stale
- document search may rely more heavily on Postgres fallback paths

Uploads and intelligence continue to run.

### Meilisearch down

Symptoms:

- search endpoints that prefer Meilisearch degrade or fall back
- uploads, previews, and intelligence continue

## If you replace GTC with Kafka + Debezium

That is a valid replacement, but it only replaces the search-index projection leg.

What would change:

- `document_search_projections` can remain the canonical projection table in Postgres.
- Debezium can publish change events from Postgres.
- A Kafka consumer can build the Meilisearch indexing sink.
- TMS does not need to change its upload, thumbnail, OCR, or intelligence orchestration for that swap.

What should remain the same:

- Postgres stays the canonical source of truth.
- TMS keeps writing `document_search_projections`.
- Meilisearch remains a projection/read model, not the source of truth.

## Operational checks

When document search looks wrong, check in this order:

1. `documents` row exists.
2. `document_contents` is populated as expected.
3. `document_search_projections` row exists and is current.
4. `GTC` is healthy and has no DLQ/backfill lag.
5. Meilisearch contains the expected document.

When upload or preview looks wrong, do not start with `GTC`; check:

1. `document_upload_sessions`
2. Temporal worker health
3. thumbnail workflow state
4. `documents.preview_status`
5. object storage paths
