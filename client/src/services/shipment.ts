import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import { createLimitOffsetResponse } from "@/types/server";
import {
  duplicateShipmentResponseSchema,
  previousRatesResponseSchema,
  shipmentCreateSchema,
  shipmentSchema,
  shipmentTotalsResponseSchema,
  shipmentUIPolicySchema,
  shipmentUpdateSchema,
  type DuplicateShipmentRequest,
  type DuplicateShipmentResponse,
  type GetPreviousRatesRequest,
  type PreviousRatesResponse,
  type Shipment,
  type ShipmentCreateInput,
  type ShipmentTotalsResponse,
  type ShipmentUIPolicy,
  type ShipmentUpdateInput,
} from "@/types/shipment";

const shipmentListSchema = createLimitOffsetResponse(shipmentSchema);

export class ShipmentService {
  public async list(include?: string) {
    const endpoint = include ? `/shipments/?include=${encodeURIComponent(include)}` : "/shipments/";
    const response = await api.get(endpoint);

    return safeParse(shipmentListSchema, response, "Shipment");
  }

  public async get(id: Shipment["id"], params?: Record<string, string>) {
    const endpoint = params
      ? `/shipments/${id}/?${new URLSearchParams(params).toString()}`
      : `/shipments/${id}/`;
    const response = await api.get<Shipment>(endpoint);

    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async create(payload: ShipmentCreateInput) {
    const response = await api.post<Shipment>("/shipments/", shipmentCreateSchema.parse(payload));
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async update(id: Shipment["id"], payload: ShipmentUpdateInput) {
    const response = await api.put<Shipment>(
      `/shipments/${id}/`,
      shipmentUpdateSchema.parse(payload),
    );
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async duplicate(request: DuplicateShipmentRequest) {
    const response = await api.post<DuplicateShipmentResponse>("/shipments/duplicate/", request);
    return safeParse(duplicateShipmentResponseSchema, response, "Shipment Duplicate");
  }

  public async cancel(shipmentId: string, cancelReason?: string) {
    const response = await api.post<Shipment>(`/shipments/${shipmentId}/cancel/`, {
      shipmentId,
      cancelReason: cancelReason ?? "",
    });
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async uncancel(shipmentId: string) {
    const response = await api.post<Shipment>(`/shipments/${shipmentId}/uncancel/`, { shipmentId });
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async transferOwnership(shipmentId: string, ownerId: string) {
    const response = await api.post<Shipment>(`/shipments/${shipmentId}/transfer-ownership/`, {
      ownerId,
    });
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async calculateTotals(payload: Shipment, signal?: AbortSignal) {
    const response = await api.post<ShipmentTotalsResponse>(
      "/shipments/calculate-totals/",
      payload,
      { signal },
    );
    return safeParse(shipmentTotalsResponseSchema, response, "Shipment Totals");
  }

  public async checkForDuplicateBOLs(bol: string, shipmentId?: string) {
    return api.post<{ valid: boolean }>("/shipments/check-for-duplicate-bols/", {
      bol,
      shipmentId,
    });
  }

  public async checkHazmatSegregation(commodityIds: string[]) {
    return api.post<{ valid: boolean }>("/shipments/check-hazmat-segregation/", {
      commodityIds,
    });
  }

  public async getPreviousRates(request: GetPreviousRatesRequest) {
    const response = await api.post<PreviousRatesResponse>("/shipments/previous-rates/", request);
    return safeParse(previousRatesResponseSchema, response, "Previous Rates");
  }

  public async delayShipments() {
    const response = await api.post<Shipment[]>("/shipments/delay/", {});
    return response;
  }

  public async getDelayedShipments() {
    const response = await api.get<Shipment[]>("/shipments/delayed/");
    return response;
  }

  public async getUIPolicy() {
    const response = await api.get<ShipmentUIPolicy>("/shipments/ui-policy/");
    return safeParse(shipmentUIPolicySchema, response, "Shipment UI Policy");
  }
}
