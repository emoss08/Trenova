import {
  autoRateShipmentGraphQL,
  calculateShipmentDistanceGraphQL,
  calculateShipmentLoadingOptimizationGraphQL,
  calculateShipmentTotalsGraphQL,
  cancelShipmentGraphQL,
  checkShipmentDuplicateBOLGraphQL,
  checkShipmentHazmatSegregationGraphQL,
  createShipmentGraphQL,
  duplicateShipmentGraphQL,
  getShipmentBillingReadinessGraphQL,
  getShipmentEtaGraphQL,
  getShipmentGraphQL,
  getShipmentPreviousRatesGraphQL,
  getShipmentUIPolicyGraphQL,
  listShipmentCommentsGraphQL,
  listShipmentsGraphQL,
  listUnassignedShipmentsGraphQL,
  previewShipmentContractRateGraphQL,
  recalculateShipmentDistanceGraphQL,
  transferShipmentOwnershipGraphQL,
  transferShipmentToBillingGraphQL,
  uncancelShipmentGraphQL,
  updateShipmentGraphQL,
} from "@/lib/graphql/shipment";
import {
  createShipmentPermitGraphQL,
  getShipmentPermitAssessmentGraphQL,
  listShipmentPermitRequirementsGraphQL,
  listShipmentPermitsGraphQL,
  updateShipmentPermitGraphQL,
  waiveShipmentPermitRequirementGraphQL,
} from "@/lib/graphql/shipment-permits";
import type { PermitInput } from "@trenova/graphql/generated/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  loadingOptimizationResultSchema,
  type LoadingOptimizationRequest,
} from "@/types/loading-optimization";
import { createLimitOffsetResponse, type PaginationInfo } from "@trenova/shared/types/server";
import type { BillType } from "@trenova/shared/types/bill-type";
import {
  permitAssessmentSchema,
  permitCreateSchema,
  permitListSchema,
  permitRequirementListSchema,
  permitRequirementSchema,
  permitSchema,
  type PermitCreateInput,
} from "@trenova/shared/types/permit";
import {
  contractRateSchema,
  duplicateShipmentResponseSchema,
  previousRatesResponseSchema,
  shipmentBillingReadinessSchema,
  shipmentCreateSchema,
  shipmentDistanceResponseSchema,
  shipmentEtaSchema,
  shipmentSchema,
  shipmentTotalsResponseSchema,
  shipmentUIPolicySchema,
  shipmentUpdateSchema,
  type DuplicateShipmentRequest,
  type GetPreviousRatesRequest,
  type Shipment,
  type ShipmentCreateInput,
  type ShipmentUpdateInput,
} from "@trenova/shared/types/shipment";
import type { ShipmentCommentListResponse } from "@/types/shipment-comment";

const shipmentListSchema = createLimitOffsetResponse(shipmentSchema);

const UNKNOWN_SHIPMENT_ETA = { verdict: "Unknown", stops: [] } as const;

export function toPermitInput(payload: PermitCreateInput): PermitInput {
  const parsed = permitCreateSchema.parse(payload);
  return {
    stateId: parsed.stateId,
    permitNumber: parsed.permitNumber,
    status: parsed.status,
    issuedAt: parsed.issuedAt ?? null,
    expiresAt: parsed.expiresAt ?? null,
    cost: parsed.cost == null ? null : String(parsed.cost),
    notes: parsed.notes ?? null,
  };
}

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

  public async create(payload: ShipmentCreateInput, options: { idempotencyKey?: string } = {}) {
    const response = await createShipmentGraphQL(shipmentCreateSchema.parse(payload), options);
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

  /**
   * Asks the rate agreements what they would charge for what is on screen.
   *
   * Nothing is written. The panel offers the answer, and the shipment only
   * carries a contract rate once somebody saves the fields it filled in.
   */
  public async previewContractRate(payload: Shipment) {
    const response = await previewShipmentContractRateGraphQL(payload);
    return safeParse(contractRateSchema, response, "Contract Rate");
  }

  /**
   * Prices a saved shipment from its contract again.
   *
   * This overwrites: the rating method, the base rate and every contract
   * accessorial go back to what the agreement says. It is the one action that
   * discards a hand-priced rate, which is why nothing else calls it.
   */
  public async autoRate(shipmentId: string) {
    const response = await autoRateShipmentGraphQL(shipmentId);
    const [shipment, contractRate] = await Promise.all([
      safeParse(shipmentSchema, response.shipment, "Shipment"),
      safeParse(contractRateSchema, response.contractRate, "Contract Rate"),
    ]);

    return { shipment, contractRate };
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

  public async getBillingReadiness(shipmentId: Shipment["id"]) {
    const response = await getShipmentBillingReadinessGraphQL(shipmentId);
    return safeParse(shipmentBillingReadinessSchema, response, "Shipment Billing Readiness");
  }

  public async getEta(
    shipmentId: NonNullable<Shipment["id"]>,
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const eta = await getShipmentEtaGraphQL(shipmentId, { signal });
    return safeParse(shipmentEtaSchema, eta ?? UNKNOWN_SHIPMENT_ETA, "Shipment ETA");
  }

  public async getPermitAssessment(
    shipmentId: NonNullable<Shipment["id"]>,
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await getShipmentPermitAssessmentGraphQL(shipmentId, { signal });
    return safeParse(permitAssessmentSchema, response, "Shipment Permit Assessment");
  }

  public async listPermits(
    shipmentId: NonNullable<Shipment["id"]>,
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await listShipmentPermitsGraphQL(shipmentId, { signal });
    return safeParse(permitListSchema, response, "Shipment Permits");
  }

  public async listPermitRequirements(
    shipmentId: NonNullable<Shipment["id"]>,
    { signal }: { signal?: AbortSignal } = {},
  ) {
    const response = await listShipmentPermitRequirementsGraphQL(shipmentId, { signal });
    return safeParse(permitRequirementListSchema, response, "Permit Requirements");
  }

  public async createPermit(shipmentId: NonNullable<Shipment["id"]>, payload: PermitCreateInput) {
    const response = await createShipmentPermitGraphQL(shipmentId, toPermitInput(payload));
    return safeParse(permitSchema, response, "Permit");
  }

  public async updatePermit(
    shipmentId: NonNullable<Shipment["id"]>,
    permitId: string,
    payload: PermitCreateInput,
  ) {
    const response = await updateShipmentPermitGraphQL(
      shipmentId,
      permitId,
      toPermitInput(payload),
    );
    return safeParse(permitSchema, response, "Permit");
  }

  public async waivePermitRequirement(requirementId: string, reason: string) {
    const response = await waiveShipmentPermitRequirementGraphQL(requirementId, reason);
    return safeParse(permitRequirementSchema, response, "Permit Requirement");
  }
}
