import { pageImage } from "@/lib/queries/document-page";
import { apiService } from "@/services/api";
import {
  billingQueuePacketPageEndpoint,
  billingQueuePacketThumbnailEndpoint,
} from "@/services/billing-queue";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/**
 * The invoice packet an item's preview shows. Kept apart from the billing queue's keys: a write
 * to the queue refreshes those, and a packet's pages never change under their build, so they
 * are refreshed only when the packet is assembled again.
 */
export const billingQueuePacket = createQueryKeys("billingQueuePacket", {
  /** Assembles the item's packet afresh each time a viewer opens it. */
  preview: (itemId: string) => ({
    queryKey: [itemId],
    queryFn: async ({ signal }) =>
      apiService.billingQueueService.getPacketPreview(itemId, { signal }),
  }),
  /** A page of one assembly of the packet: a new build draws its pages afresh. */
  page: (itemId: string, build: string, page: number) => ({
    queryKey: [itemId, build, "page", page],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      pageImage(billingQueuePacketPageEndpoint(itemId, page), signal),
  }),
  /** A page of one assembly of the packet, drawn small for the page rail. */
  thumbnail: (itemId: string, build: string, page: number) => ({
    queryKey: [itemId, build, "thumbnail", page],
    queryFn: ({ signal }: { signal: AbortSignal }) =>
      pageImage(billingQueuePacketThumbnailEndpoint(itemId, page), signal),
  }),
  /** The checks of the assembly the server holds; a build other than the one shown is stale. */
  checks: (itemId: string, build: string) => ({
    queryKey: [itemId, build, "checks"],
    queryFn: async ({ signal }) =>
      apiService.billingQueueService.getPacketChecks(itemId, { signal }),
  }),
});
