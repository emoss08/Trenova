import { safeParse } from "@trenova/shared/lib/parse";
import {
  acknowledgeShipmentCommentGraphQL,
  createShipmentCommentGraphQL,
  deleteShipmentCommentGraphQL,
  getShipmentCommentCountGraphQL,
  listShipmentCommentRepliesGraphQL,
  listShipmentCommentsGraphQL,
  pinShipmentCommentGraphQL,
  resolveShipmentCommentGraphQL,
  unpinShipmentCommentGraphQL,
  unresolveShipmentCommentGraphQL,
  updateShipmentCommentGraphQL,
} from "@/lib/graphql/shipment";
import type { Shipment } from "@trenova/shared/types/shipment";
import {
  shipmentCommentCountResponseSchema,
  shipmentCommentListResponseSchema,
  shipmentCommentSchema,
  type ShipmentCommentCreateInput,
  type ShipmentCommentFilter,
  type ShipmentCommentUpdateInput,
} from "@/types/shipment-comment";

export class ShipmentCommentService {
  public async getCount(shipmentId: string) {
    const response = await getShipmentCommentCountGraphQL(shipmentId);
    return safeParse(shipmentCommentCountResponseSchema, response, "Shipment Comment Count");
  }

  public async list(
    shipmentId: Shipment["id"],
    params?: { limit?: number; after?: string | null; filter?: ShipmentCommentFilter | null },
  ) {
    const response = await listShipmentCommentsGraphQL({
      shipmentId,
      limit: params?.limit ?? 20,
      after: params?.after ?? null,
      filter: params?.filter ?? null,
    });
    return safeParse(shipmentCommentListResponseSchema, response, "Shipment Comments");
  }

  public async listReplies(
    shipmentId: Shipment["id"],
    commentId: string,
    params?: { limit?: number; after?: string | null },
  ) {
    const response = await listShipmentCommentRepliesGraphQL({
      shipmentId,
      commentId,
      limit: params?.limit ?? 10,
      after: params?.after ?? null,
    });
    return safeParse(shipmentCommentListResponseSchema, response, "Shipment Comment Replies");
  }

  public async create(shipmentId: string, data: ShipmentCommentCreateInput) {
    const response = await createShipmentCommentGraphQL(shipmentId, data);
    return safeParse(shipmentCommentSchema, response, "Shipment Comment");
  }

  public async update(shipmentId: string, commentId: string, data: ShipmentCommentUpdateInput) {
    const response = await updateShipmentCommentGraphQL(shipmentId, commentId, data);
    return safeParse(shipmentCommentSchema, response, "Shipment Comment");
  }

  public async delete(shipmentId: string, commentId: string) {
    await deleteShipmentCommentGraphQL(shipmentId, commentId);
  }

  public async setPinned(shipmentId: string, commentId: string, pinned: boolean) {
    const response = pinned
      ? await pinShipmentCommentGraphQL(shipmentId, commentId)
      : await unpinShipmentCommentGraphQL(shipmentId, commentId);
    return safeParse(shipmentCommentSchema, response, "Shipment Comment");
  }

  public async setResolved(shipmentId: string, commentId: string, resolved: boolean) {
    const response = resolved
      ? await resolveShipmentCommentGraphQL(shipmentId, commentId)
      : await unresolveShipmentCommentGraphQL(shipmentId, commentId);
    return safeParse(shipmentCommentSchema, response, "Shipment Comment");
  }

  public async acknowledge(shipmentId: string, commentId: string) {
    const response = await acknowledgeShipmentCommentGraphQL(shipmentId, commentId);
    return safeParse(shipmentCommentSchema, response, "Shipment Comment");
  }
}
