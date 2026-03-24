import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  shipmentCommentCountResponseSchema,
  shipmentCommentListResponseSchema,
  shipmentCommentSchema,
  type ShipmentComment,
  type ShipmentCommentCountResponse,
  type ShipmentCommentCreateInput,
  type ShipmentCommentListResponse,
  type ShipmentCommentUpdateInput,
} from "@/types/shipment-comment";

export class ShipmentCommentService {
  public async getCount(shipmentId: string) {
    const response = await api.get<ShipmentCommentCountResponse>(
      `/shipments/${shipmentId}/comments/count/`,
    );
    return safeParse(shipmentCommentCountResponseSchema, response, "Shipment Comment Count");
  }

  public async list(shipmentId: string, params?: { limit?: number; offset?: number }) {
    const searchParams = new URLSearchParams();
    if (params?.limit != null) searchParams.set("limit", String(params.limit));
    if (params?.offset != null) searchParams.set("offset", String(params.offset));
    const qs = searchParams.toString();
    const response = await api.get<ShipmentCommentListResponse>(
      `/shipments/${shipmentId}/comments/${qs ? `?${qs}` : ""}`,
    );
    return safeParse(shipmentCommentListResponseSchema, response, "Shipment Comments");
  }

  public async create(shipmentId: string, data: ShipmentCommentCreateInput) {
    const response = await api.post<ShipmentComment>(
      `/shipments/${shipmentId}/comments/`,
      data,
    );
    return safeParse(shipmentCommentSchema, response, "Shipment Comment");
  }

  public async update(shipmentId: string, commentId: string, data: ShipmentCommentUpdateInput) {
    const response = await api.put<ShipmentComment>(
      `/shipments/${shipmentId}/comments/${commentId}/`,
      data,
    );
    return safeParse(shipmentCommentSchema, response, "Shipment Comment");
  }

  public async delete(shipmentId: string, commentId: string) {
    await api.delete(`/shipments/${shipmentId}/comments/${commentId}/`);
  }
}
