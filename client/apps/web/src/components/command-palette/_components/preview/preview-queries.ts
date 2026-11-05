import { getFileCategory } from "@/components/documents/document-utils";
import { fetchRecordShipments, type ShipmentScope } from "@/lib/graphql/record-shipments";
import { fetchDeskWorker } from "@/lib/graphql/worker-card";
import { ar } from "@/lib/queries/ar";
import { invoice } from "@/lib/queries/invoice";
import { shipment } from "@/lib/queries/shipment";
import { telematics } from "@/lib/queries/telematics";
import { documentService } from "@/services/document";
import {
  CommandPaletteCustomerPreviewDocument,
  type CommandPaletteCustomerPreviewQuery,
  type CommandPaletteCustomerPreviewQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Document } from "@trenova/shared/types/document";
import type { QueryClient } from "@tanstack/react-query";
import type { PaletteRecord } from "../../palette-model";

/**
 * The queries behind each record preview, apart from the previews themselves
 * so the palette can warm them without drawing anything. Each preview reads
 * several small queries rather than one wide one: a part the person may not
 * read (receivables, hours of service) fails alone and its section is
 * hidden, instead of taking the whole preview with it.
 */
export const PREVIEW_STALE_TIME = 30_000;

export const SHIPMENT_EVENTS_SHOWN = 4;
export const OPEN_LOADS_SHOWN = 3;

export type CustomerPreviewData = CommandPaletteCustomerPreviewQuery["customer"];

export function customerPreviewQuery(id: string) {
  return {
    queryKey: ["command-palette", "customer-preview", id] as const,
    queryFn: async ({ signal }: { signal: AbortSignal }) => {
      const data = await requestGraphQL<
        CommandPaletteCustomerPreviewQuery,
        CommandPaletteCustomerPreviewQueryVariables
      >({
        document: CommandPaletteCustomerPreviewDocument,
        operationName: "CommandPaletteCustomerPreview",
        variables: { id },
        signal,
      });
      return data.customer;
    },
    staleTime: PREVIEW_STALE_TIME,
  };
}

export function customerAgingQuery(id: string) {
  return { ...ar.customerProfile(id), staleTime: PREVIEW_STALE_TIME };
}

export function openShipmentsQuery(scope: ShipmentScope, first: number) {
  return {
    queryKey: ["command-palette", "open-shipments", scope, first] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      fetchRecordShipments(scope, { first, after: null, openOnly: true }, { signal }),
    staleTime: PREVIEW_STALE_TIME,
  };
}

export function workerPreviewQuery(id: string) {
  return {
    queryKey: ["command-palette", "worker-preview", id] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchDeskWorker(id, { signal }),
    staleTime: PREVIEW_STALE_TIME,
  };
}

export function workerHosQuery(id: string) {
  return { ...telematics.workerHosState(id), staleTime: PREVIEW_STALE_TIME };
}

export function documentPreviewQuery(id: string) {
  return {
    queryKey: ["command-palette", "document-preview", id] as const,
    queryFn: () => documentService.getById(id),
    staleTime: PREVIEW_STALE_TIME,
  };
}

/** Whether the server draws the file's pages: a PDF or an image, as the page endpoint serves them. */
export function hasPages(document: Pick<Document, "fileType" | "originalName">): boolean {
  const category = getFileCategory(document.fileType, document.originalName);
  return category === "pdf" || category === "image";
}

export function documentExtractionQuery(id: string) {
  return {
    queryKey: ["command-palette", "document-extraction", id] as const,
    queryFn: () => documentService.getShipmentDraft(id),
    staleTime: PREVIEW_STALE_TIME,
  };
}

/** The invoice as the Desk's invoice card reads it, so the two share one cached read. */
export function invoicePreviewQuery(id: string) {
  return { ...invoice.card(id), staleTime: PREVIEW_STALE_TIME };
}

export function shipmentPreviewQuery(id: string) {
  return {
    ...shipment.get(id, { expandShipmentDetails: "true" }),
    staleTime: PREVIEW_STALE_TIME,
  };
}

export function shipmentEventsQuery(id: string) {
  return { ...shipment.events(id, SHIPMENT_EVENTS_SHOWN), staleTime: PREVIEW_STALE_TIME };
}

/**
 * Starts loading a record's preview before it is shown; a cached one is left
 * alone. A document's record is warmed but never its page: the server logs
 * every page it draws as the document being viewed, and a page fetched for a
 * preview nobody opened would put a view in the access log that never happened.
 */
export function prefetchRecordPreview(queryClient: QueryClient, record: PaletteRecord): void {
  switch (record.entityType) {
    case "shipment":
      void queryClient.prefetchQuery(shipmentPreviewQuery(record.id));
      void queryClient.prefetchQuery(shipmentEventsQuery(record.id));
      return;
    case "customer":
      void queryClient.prefetchQuery(customerPreviewQuery(record.id));
      return;
    case "worker":
      void queryClient.prefetchQuery(workerPreviewQuery(record.id));
      return;
    case "invoice":
      void queryClient.prefetchQuery(invoicePreviewQuery(record.id));
      return;
    case "document":
      void queryClient.prefetchQuery(documentPreviewQuery(record.id));
      return;
  }
}
