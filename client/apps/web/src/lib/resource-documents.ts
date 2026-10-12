import { apiService } from "@/services/api";
import type { Document } from "@trenova/shared/types/document";
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
      apiService.documentService.getByResource(resourceType, resourceId, search, {
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
