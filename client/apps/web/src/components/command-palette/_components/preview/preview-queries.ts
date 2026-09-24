import { fetchWorkerOverview, WORKER_OVERVIEW_KEY } from "@/lib/graphql/worker-overview";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  CommandPaletteCustomerPreviewDocument,
  type CommandPaletteCustomerPreviewQuery,
  type CommandPaletteCustomerPreviewQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { QueryClient } from "@tanstack/react-query";
import type { PaletteRecord } from "../../palette-model";

/**
 * The queries behind each record preview, apart from the previews themselves
 * so the palette can warm them without pulling in a preview's chunk (the
 * shipment preview carries the map).
 */
export const PREVIEW_STALE_TIME = 30_000;

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

export function workerPreviewQuery(id: string) {
  return {
    queryKey: [WORKER_OVERVIEW_KEY, id] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchWorkerOverview(id, { signal }),
    staleTime: PREVIEW_STALE_TIME,
  };
}

export function documentPreviewQuery(id: string) {
  return {
    queryKey: ["command-palette", "document-preview", id] as const,
    queryFn: () => apiService.documentService.getById(id),
    staleTime: PREVIEW_STALE_TIME,
  };
}

export function shipmentPreviewQuery(id: string) {
  return {
    ...queries.shipment.get(id, { expandShipmentDetails: "true" }),
    staleTime: PREVIEW_STALE_TIME,
  };
}

/** Starts loading a record's preview before it is shown; a cached one is left alone. */
export function prefetchRecordPreview(queryClient: QueryClient, record: PaletteRecord): void {
  switch (record.entityType) {
    case "shipment":
      void queryClient.prefetchQuery(shipmentPreviewQuery(record.id));
      return;
    case "customer":
      void queryClient.prefetchQuery(customerPreviewQuery(record.id));
      return;
    case "worker":
      void queryClient.prefetchQuery(workerPreviewQuery(record.id));
      return;
    case "document":
      void queryClient.prefetchQuery(documentPreviewQuery(record.id));
      return;
  }
}
