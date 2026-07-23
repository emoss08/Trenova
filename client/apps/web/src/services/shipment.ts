import {
  bulkTransferShipmentsToBillingGraphQL,
  calculateShipmentDistanceGraphQL,
  calculateShipmentLoadingOptimizationGraphQL,
  calculateShipmentTotalsGraphQL,
  cancelShipmentGraphQL,
  checkShipmentDuplicateBOLGraphQL,
  checkShipmentHazmatSegregationGraphQL,
  createShipmentGraphQL,
  duplicateShipmentGraphQL,
  getShipmentBillingReadinessGraphQL,
  getShipmentGraphQL,
  getShipmentPreviousRatesGraphQL,
  getShipmentUIPolicyGraphQL,
  listShipmentCommentsGraphQL,
  listShipmentsGraphQL,
  listUnassignedShipmentsGraphQL,
  recalculateShipmentDistanceGraphQL,
  transferShipmentOwnershipGraphQL,
  transferShipmentToBillingGraphQL,
  uncancelShipmentGraphQL,
  updateShipmentGraphQL,
} from "@/lib/graphql/shipment";
import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  loadingOptimizationResultSchema,
  type LoadingOptimizationRequest,
} from "@/types/loading-optimization";
import { createLimitOffsetResponse, type PaginationInfo } from "@trenova/shared/types/server";
import type { BillType } from "@trenova/shared/types/bill-type";
import {
  bulkTransferToBillingResponseSchema,
  duplicateShipmentResponseSchema,
  previousRatesResponseSchema,
  shipmentBillingReadinessSchema,
  shipmentCreateSchema,
  shipmentDistanceResponseSchema,
  shipmentSchema,
  shipmentTotalsResponseSchema,
  shipmentUIPolicySchema,
  shipmentUpdateSchema,
  type BulkTransferToBillingRequest,
  type DuplicateShipmentRequest,
  type GetPreviousRatesRequest,
  type Shipment,
  type ShipmentCreateInput,
  type ShipmentUpdateInput,
} from "@trenova/shared/types/shipment";
import type { ShipmentCommentListResponse } from "@/types/shipment-comment";

const shipmentListSchema = createLimitOffsetResponse(shipmentSchema);

export class ShipmentService {
  public async list(_include?: string) {
    const response = await listShipmentsGraphQL({
      limit: 20,
    });
    return safeParse(shipmentListSchema, response, "Shipment");
  }

  public async listUnassigned(req: { limit: number; after?: string | null }) {
    const response = await listUnassignedShipmentsGraphQL(req);

    return safeParse(shipmentListSchema, response, "Unassigned Shipments");
  }

  public async get(id: Shipment["id"], _params?: Record<string, string>) {
    const response = await getShipmentGraphQL(id);

    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async create(payload: ShipmentCreateInput) {
    const response = await createShipmentGraphQL(shipmentCreateSchema.parse(payload));
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async update(id: Shipment["id"], payload: ShipmentUpdateInput) {
    const response = await updateShipmentGraphQL(id, shipmentUpdateSchema.parse(payload));
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async duplicate(request: DuplicateShipmentRequest) {
    const response = await duplicateShipmentGraphQL(request);
    return safeParse(duplicateShipmentResponseSchema, response, "Shipment Duplicate");
  }

  public async getComments(req: PaginationInfo & { shipmentId: Shipment["id"] }) {
    const response = await listShipmentCommentsGraphQL({
      shipmentId: req.shipmentId,
      limit: req.limit ?? 20,
      after: null,
    });

    return response as ShipmentCommentListResponse;
  }

  public async cancel(shipmentId: string, cancelReason?: string) {
    const response = await cancelShipmentGraphQL(shipmentId, cancelReason);
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async uncancel(shipmentId: string) {
    const response = await uncancelShipmentGraphQL(shipmentId);
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async transferOwnership(shipmentId: string, ownerId: string) {
    const response = await transferShipmentOwnershipGraphQL(shipmentId, ownerId);
    return safeParse(shipmentSchema, response, "Shipment");
  }

  public async calculateTotals(payload: Shipment, _signal?: AbortSignal) {
    const response = await calculateShipmentTotalsGraphQL(payload);
    return safeParse(shipmentTotalsResponseSchema, response, "Shipment Totals");
  }

  public async calculateDistance(payload: Shipment, _signal?: AbortSignal) {
    const response = await calculateShipmentDistanceGraphQL(payload);
    return safeParse(shipmentDistanceResponseSchema, response, "Shipment Distance");
  }

  public async recalculateDistance(shipmentId: Shipment["id"]) {
    const response = await recalculateShipmentDistanceGraphQL(shipmentId);
    return safeParse(shipmentDistanceResponseSchema, response, "Shipment Distance");
  }

  public async checkForDuplicateBOLs(bol: string, shipmentId?: string) {
    return checkShipmentDuplicateBOLGraphQL(bol, shipmentId);
  }

  public async checkHazmatSegregation(commodityIds: string[]) {
    return checkShipmentHazmatSegregationGraphQL(commodityIds);
  }

  public async getPreviousRates(request: GetPreviousRatesRequest) {
    const response = await getShipmentPreviousRatesGraphQL(request);
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
    const response = await getShipmentUIPolicyGraphQL();
    return safeParse(shipmentUIPolicySchema, response, "Shipment UI Policy");
  }

  public async calculateLoadingOptimization(req: LoadingOptimizationRequest) {
    const response = await calculateShipmentLoadingOptimizationGraphQL(req);
    return safeParse(loadingOptimizationResultSchema, response, "Loading Optimization");
  }

  public async transferToBilling(shipmentId: string, billType?: BillType) {
    return transferShipmentToBillingGraphQL(shipmentId, billType);
  }

  public async bulkTransferToBilling(req: BulkTransferToBillingRequest) {
    const response = await bulkTransferShipmentsToBillingGraphQL(req);

    return safeParse(bulkTransferToBillingResponseSchema, response, "Bulk Transfer to Billing");
  }

  public async getBillingReadiness(shipmentId: Shipment["id"]) {
    const response = await getShipmentBillingReadinessGraphQL(shipmentId);
    return safeParse(shipmentBillingReadinessSchema, response, "Shipment Billing Readiness");
  }
}
