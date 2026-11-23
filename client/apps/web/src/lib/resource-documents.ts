import { documentService } from "@/services/document";
import type { Document, DocumentPreviewStatus } from "@trenova/shared/types/document";
import { queryOptions } from "@tanstack/react-query";

/** How often a record's documents are read again while one of them is still being processed. */
export const PENDING_DOCUMENTS_REFETCH_MS = 3_000;

/**
 * The documents attached to one record, with their types, as the documents
 * tab and the Desk's cards read them. One key for both, so an upload, a
 * review or a delete in the tab refreshes the card too: everything that
 * changes a record's documents invalidates ["documents", type, id].
 */
export function resourceDocumentsQuery(resourceType: string, resourceId: string, search = "") {
  return queryOptions({
    queryKey: ["documents", resourceType, resourceId, search] as const,
    queryFn: () =>
      documentService.getByResource(resourceType, resourceId, search, {
        includeDocumentType: "true",
      }),
  });
}

/**
 * Whether the server is still working on a document: drawing its preview, or,
 * for one uploaded to be read (a rate confirmation), reading it and drafting
 * the shipment from it. Its readers poll until it settles.
 */
export function hasPendingDocumentWork(document: Document): boolean {
  if (document.previewStatus === "Pending") return true;
  if (document.processingProfile !== "rate_confirmation_import") return false;

  return (
    document.contentStatus === "Pending" ||
    document.contentStatus === "Extracting" ||
    document.shipmentDraftStatus === "Pending"
  );
}

/** A document a record's read carries, as far as waiting on its preview needs it. */
export type PreviewedDocument = { previewStatus?: DocumentPreviewStatus | null } | null | undefined;

/** How many reads a pending preview gets before its read stops polling for it. */
export const PREVIEW_PENDING_MAX_POLLS = 40;

const pendingSince = new WeakMap<object, number>();

/**
 * The refetch interval of a read whose rows carry documents (a credential's scan, a training
 * certificate, a safety event's report): on while any of their previews is still being drawn and
 * off once none is, so a preview appears without a reload. Bounded: once a pending preview has
 * been waited on for `PREVIEW_PENDING_MAX_POLLS` intervals (a preview the worker never finished),
 * the read stops polling until its previews settle or a write reads it again. `query` is the
 * TanStack query the interval belongs to (its identity holds when the wait started). TanStack
 * stops it while the browser tab is hidden.
 */
export function previewPendingRefetchMs(
  query: object,
  documents: Iterable<PreviewedDocument>,
  now = Date.now(),
): number | false {
  let pending = false;
  for (const file of documents) {
    if (file?.previewStatus === "Pending") {
      pending = true;
      break;
    }
  }
  if (!pending) {
    pendingSince.delete(query);
    return false;
  }
  const since = pendingSince.get(query);
  if (since === undefined) {
    pendingSince.set(query, now);
    return PENDING_DOCUMENTS_REFETCH_MS;
  }
  return now - since < PENDING_DOCUMENTS_REFETCH_MS * PREVIEW_PENDING_MAX_POLLS
    ? PENDING_DOCUMENTS_REFETCH_MS
    : false;
}
