import { safeParse } from "@/lib/parse";
import {
  createShipmentCommentGraphQL,
  deleteShipmentCommentGraphQL,
  getShipmentCommentCountGraphQL,
  listShipmentCommentsGraphQL,
  updateShipmentCommentGraphQL,
} from "@/lib/graphql/shipment";
import type { Shipment } from "@/types/shipment";
import {
  shipmentCommentCountResponseSchema,
  shipmentCommentListResponseSchema,
  shipmentCommentSchema,
  type ShipmentCommentCreateInput,
  type ShipmentCommentUpdateInput,
} from "@/types/shipment-comment";

export class ShipmentCommentService {
  public async getCount(shipmentId: string) {
    const response = await getShipmentCommentCountGraphQL(shipmentId);
    return safeParse(shipmentCommentCountResponseSchema, response, "Shipment Comment Count");
  }

  public async list(shipmentId: Shipment["id"], params?: { limit?: number; after?: string | null }) {
    const response = await listShipmentCommentsGraphQL({
      shipmentId,
      limit: params?.limit ?? 20,
      after: params?.after ?? null,
    });
    return safeParse(shipmentCommentListResponseSchema, response, "Shipment Comments");
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
}
