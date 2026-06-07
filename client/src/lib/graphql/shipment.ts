import {
  BulkTransferShipmentsToBillingDocument,
  CalculateShipmentDistanceDocument,
  CalculateShipmentLoadingOptimizationDocument,
  CalculateShipmentTotalsDocument,
  CancelShipmentDocument,
  CheckShipmentDuplicateBolDocument,
  CheckShipmentHazmatSegregationDocument,
  CreateShipmentCommentDocument,
  CreateShipmentDocument,
  DeleteShipmentCommentDocument,
  DuplicateShipmentDocument,
  ExceptionShipmentsDocument,
  MapShipmentsDocument,
  RecalculateShipmentDistanceDocument,
  ShipmentBillingReadinessDocument,
  ShipmentCommentCountDocument,
  ShipmentCommentsDocument,
  ShipmentCommandCenterTableDocument,
  ShipmentDetailDocument,
  ShipmentEventsDocument,
  ShipmentPageAnalyticsDocument,
  ShipmentPreviousRatesDocument,
  ShipmentSavedViewCountsDocument,
  ShipmentUiPolicyDocument,
  TransferShipmentOwnershipDocument,
  TransferShipmentToBillingDocument,
  UnassignedShipmentsDocument,
  UncancelShipmentDocument,
  UpdateShipmentCommentDocument,
  UpdateShipmentDocument,
  type BillType,
  type FieldFilterInput,
  type FilterGroupInput,
  type ShipmentAdditionalChargeInput,
  type ShipmentBulkTransferToBillingInput,
  type ShipmentCommentInput,
  type ShipmentCommentUpdateInput,
  type ShipmentCommandCenterTableQueryVariables,
  type ShipmentCommodityInput,
  type ShipmentDuplicateInput,
	type ShipmentInput,
	type ShipmentLoadingOptimizationInput,
	type ShipmentMoveInput,
	type ShipmentPreviousRatesInput,
} from "@/graphql/generated/graphql";
import { requestGraphQL } from "@/lib/graphql";
import { defineDataTableGraphQLConfig } from "@/lib/graphql/data-table";
import type { GraphQLExecutableDocument } from "@/types/graphql";
import type { LoadingOptimizationRequest } from "@/types/loading-optimization";
import type { GenericLimitOffsetResponse } from "@/types/server";
import type {
  BulkTransferToBillingRequest,
  DuplicateShipmentRequest,
  GetPreviousRatesRequest,
  Shipment,
  ShipmentCreateInput,
  ShipmentUpdateInput,
} from "@/types/shipment";
import type { ShipmentComment } from "@/types/shipment-comment";
import type { ShipmentEventList, ShipmentEventType } from "@/types/shipment-event";

type ShipmentConnection = {
  edges?: Array<{ node: unknown }>;
  pageInfo?: {
    hasNextPage?: boolean;
    endCursor?: string | null;
  };
  totalCount?: number | null;
};

type ShipmentPageRequest = {
  limit: number;
  after?: string | null;
  query?: string;
  fieldFilters?: FieldFilterInput[];
  filterGroups?: FilterGroupInput[];
};

type ShipmentGraphQLParams<TVariables> = {
  document: GraphQLExecutableDocument;
  operationName: string;
  variables?: TVariables;
};

function requestShipmentGraphQL<TVariables = Record<string, unknown>>(
  params: ShipmentGraphQLParams<TVariables>,
): Promise<Record<string, any>> {
  return requestGraphQL<Record<string, any>, TVariables>(params);
}

export const shipmentTableGraphQLConfig = defineDataTableGraphQLConfig<
  Shipment,
  ShipmentCommandCenterTableQueryVariables
>({
  document: ShipmentCommandCenterTableDocument,
  operationName: "ShipmentCommandCenterTable",
  connectionKey: "shipments",
  variables: {
    expandShipmentDetails: true,
  },
  mapNode: (node) => node as Shipment,
});

export async function listShipmentsGraphQL(
  req: ShipmentPageRequest,
): Promise<GenericLimitOffsetResponse<Shipment>> {
  const data = await requestShipmentGraphQL({
    document: ShipmentCommandCenterTableDocument,
    operationName: "ShipmentCommandCenterTable",
    variables: {
      first: req.limit,
      after: req.after ?? undefined,
      query: req.query,
      fieldFilters: req.fieldFilters ?? [],
      filterGroups: req.filterGroups ?? [],
      expandShipmentDetails: true,
    },
  });
  return connectionToLimitOffset(data.shipments as ShipmentConnection);
}

export async function listUnassignedShipmentsGraphQL(req: {
  limit: number;
  after?: string | null;
}): Promise<GenericLimitOffsetResponse<Shipment>> {
  const data = await requestShipmentGraphQL({
    document: UnassignedShipmentsDocument,
    operationName: "UnassignedShipments",
    variables: {
      first: req.limit,
      after: req.after ?? undefined,
    },
  });
  return connectionToLimitOffset(data.unassignedShipments as ShipmentConnection);
}

export async function listExceptionShipmentsGraphQL(req: {
  limit: number;
  after?: string | null;
  fieldFilters: FieldFilterInput[];
}): Promise<GenericLimitOffsetResponse<Shipment>> {
  const data = await requestShipmentGraphQL({
    document: ExceptionShipmentsDocument,
    operationName: "ExceptionShipments",
    variables: {
      first: req.limit,
      after: req.after ?? undefined,
      fieldFilters: req.fieldFilters,
    },
  });
  return connectionToLimitOffset(data.shipments as ShipmentConnection);
}

export async function listMapShipmentsGraphQL(req: {
  limit: number;
  after?: string | null;
  fieldFilters: FieldFilterInput[];
}): Promise<GenericLimitOffsetResponse<Shipment>> {
  const data = await requestShipmentGraphQL({
    document: MapShipmentsDocument,
    operationName: "MapShipments",
    variables: {
      first: req.limit,
      after: req.after ?? undefined,
      fieldFilters: req.fieldFilters,
    },
  });
  return connectionToLimitOffset(data.shipments as ShipmentConnection);
}

export async function getShipmentGraphQL(id: Shipment["id"]): Promise<Shipment> {
  const data = await requestShipmentGraphQL({
    document: ShipmentDetailDocument,
    operationName: "ShipmentDetail",
    variables: {
      id,
      expandShipmentDetails: true,
    },
  });
  if (!data.shipment) {
    throw new Error("Shipment not found");
  }
  return data.shipment as Shipment;
}

export async function getShipmentUIPolicyGraphQL() {
  const data = await requestShipmentGraphQL({
    document: ShipmentUiPolicyDocument,
    operationName: "ShipmentUIPolicy",
  });
  return data.shipmentUIPolicy;
}

export async function getShipmentBillingReadinessGraphQL(shipmentId: Shipment["id"]) {
  const data = await requestShipmentGraphQL({
    document: ShipmentBillingReadinessDocument,
    operationName: "ShipmentBillingReadiness",
    variables: { shipmentId },
  });
  return data.shipmentBillingReadiness;
}

export async function getShipmentSavedViewCountsGraphQL(timezone: string) {
  const data = await requestShipmentGraphQL({
    document: ShipmentSavedViewCountsDocument,
    operationName: "ShipmentSavedViewCounts",
    variables: { timezone },
  });
  return data.shipmentAnalytics.savedViewCounts;
}

export async function getShipmentPageAnalyticsGraphQL(req: {
  include?: string;
  limit?: number;
  offset?: number;
  timezone?: string;
  windowDays?: number;
  startDate?: number;
  endDate?: number;
}) {
  const data = await requestShipmentGraphQL({
    document: ShipmentPageAnalyticsDocument,
    operationName: "ShipmentPageAnalytics",
    variables: req,
  });
  return data.shipmentAnalytics;
}

export async function listShipmentEventsGraphQL(req: {
  shipmentId?: string;
  types?: ShipmentEventType[];
  limit?: number;
  before?: number;
} = {}): Promise<ShipmentEventList> {
  const data = await requestShipmentGraphQL({
    document: ShipmentEventsDocument,
    operationName: "ShipmentEvents",
    variables: {
      shipmentId: req.shipmentId,
      types: req.types,
      limit: req.limit,
      before: req.before,
    },
  });
  return data.shipmentEvents as ShipmentEventList;
}

export async function createShipmentGraphQL(payload: ShipmentCreateInput): Promise<Shipment> {
  const data = await requestShipmentGraphQL({
    document: CreateShipmentDocument,
    operationName: "CreateShipment",
    variables: {
      input: toShipmentInput(payload),
    },
  });
  return data.createShipment as Shipment;
}

export async function updateShipmentGraphQL(
  id: Shipment["id"],
  payload: ShipmentUpdateInput,
): Promise<Shipment> {
  const data = await requestShipmentGraphQL({
    document: UpdateShipmentDocument,
    operationName: "UpdateShipment",
    variables: {
      id,
      input: toShipmentInput(payload),
    },
  });
  return data.updateShipment as Shipment;
}

export async function cancelShipmentGraphQL(
  id: Shipment["id"],
  cancelReason?: string,
): Promise<Shipment> {
  const data = await requestShipmentGraphQL({
    document: CancelShipmentDocument,
    operationName: "CancelShipment",
    variables: {
      id,
      input: { cancelReason: cancelReason ?? "" },
    },
  });
  return data.cancelShipment as Shipment;
}

export async function uncancelShipmentGraphQL(id: Shipment["id"]): Promise<Shipment> {
  const data = await requestShipmentGraphQL({
    document: UncancelShipmentDocument,
    operationName: "UncancelShipment",
    variables: { id },
  });
  return data.uncancelShipment as Shipment;
}

export async function duplicateShipmentGraphQL(request: DuplicateShipmentRequest) {
  const data = await requestShipmentGraphQL({
    document: DuplicateShipmentDocument,
    operationName: "DuplicateShipment",
    variables: {
      input: {
        shipmentId: request.shipmentId,
        count: request.count,
        overrideDates: request.overrideDates,
      } satisfies ShipmentDuplicateInput,
    },
  });
  return data.duplicateShipment;
}

export async function transferShipmentOwnershipGraphQL(
  id: Shipment["id"],
  ownerId: string,
): Promise<Shipment> {
  const data = await requestShipmentGraphQL({
    document: TransferShipmentOwnershipDocument,
    operationName: "TransferShipmentOwnership",
    variables: {
      id,
      input: { ownerId },
    },
  });
  return data.transferShipmentOwnership as Shipment;
}

export async function transferShipmentToBillingGraphQL(shipmentId: string, billType?: BillType) {
  const data = await requestShipmentGraphQL({
    document: TransferShipmentToBillingDocument,
    operationName: "TransferShipmentToBilling",
    variables: {
      input: {
        shipmentId,
        billType: billType ?? "Invoice",
      },
    },
  });
  return data.transferShipmentToBilling;
}

export async function bulkTransferShipmentsToBillingGraphQL(req: BulkTransferToBillingRequest) {
  const data = await requestShipmentGraphQL({
    document: BulkTransferShipmentsToBillingDocument,
    operationName: "BulkTransferShipmentsToBilling",
    variables: {
      input: {
        shipmentIds: req.shipmentIds,
        billType: req.billType,
      } satisfies ShipmentBulkTransferToBillingInput,
    },
  });
  return data.bulkTransferShipmentsToBilling;
}

export async function calculateShipmentTotalsGraphQL(payload: Shipment) {
  const data = await requestShipmentGraphQL({
    document: CalculateShipmentTotalsDocument,
    operationName: "CalculateShipmentTotals",
    variables: {
      input: toShipmentInput(payload),
    },
  });
  return data.calculateShipmentTotals;
}

export async function calculateShipmentDistanceGraphQL(payload: Shipment) {
  const data = await requestShipmentGraphQL({
    document: CalculateShipmentDistanceDocument,
    operationName: "CalculateShipmentDistance",
    variables: {
      input: toShipmentInput(payload),
    },
  });
  return data.calculateShipmentDistance;
}

export async function recalculateShipmentDistanceGraphQL(shipmentId: Shipment["id"]) {
  const data = await requestShipmentGraphQL({
    document: RecalculateShipmentDistanceDocument,
    operationName: "RecalculateShipmentDistance",
    variables: { shipmentId },
  });
  return data.recalculateShipmentDistance;
}

export async function checkShipmentDuplicateBOLGraphQL(bol: string, shipmentId?: string) {
  const data = await requestShipmentGraphQL({
    document: CheckShipmentDuplicateBolDocument,
    operationName: "CheckShipmentDuplicateBol",
    variables: {
      input: {
        bol,
        shipmentId,
      },
    },
  });
  return data.checkShipmentDuplicateBol;
}

export async function checkShipmentHazmatSegregationGraphQL(commodityIds: string[]) {
  const data = await requestShipmentGraphQL({
    document: CheckShipmentHazmatSegregationDocument,
    operationName: "CheckShipmentHazmatSegregation",
    variables: {
      input: {
        commodityIds,
      },
    },
  });
  return data.checkShipmentHazmatSegregation;
}

export async function getShipmentPreviousRatesGraphQL(request: GetPreviousRatesRequest) {
  const data = await requestShipmentGraphQL({
    document: ShipmentPreviousRatesDocument,
    operationName: "ShipmentPreviousRates",
    variables: {
      input: request satisfies ShipmentPreviousRatesInput,
    },
  });
  return data.shipmentPreviousRates;
}

export async function calculateShipmentLoadingOptimizationGraphQL(
  req: LoadingOptimizationRequest,
) {
  const data = await requestShipmentGraphQL({
    document: CalculateShipmentLoadingOptimizationDocument,
    operationName: "CalculateShipmentLoadingOptimization",
    variables: {
      input: req satisfies ShipmentLoadingOptimizationInput,
    },
  });
  return data.calculateShipmentLoadingOptimization;
}

export async function listShipmentCommentsGraphQL(req: {
  shipmentId: Shipment["id"];
  limit: number;
  after?: string | null;
}): Promise<GenericLimitOffsetResponse<ShipmentComment>> {
  const data = await requestShipmentGraphQL({
    document: ShipmentCommentsDocument,
    operationName: "ShipmentComments",
    variables: {
      shipmentId: req.shipmentId,
      first: req.limit,
      after: req.after ?? undefined,
    },
  });
  return connectionToLimitOffset(data.shipmentComments);
}

export async function getShipmentCommentCountGraphQL(shipmentId: Shipment["id"]) {
  const data = await requestShipmentGraphQL({
    document: ShipmentCommentCountDocument,
    operationName: "ShipmentCommentCount",
    variables: { shipmentId },
  });
  return data.shipmentCommentCount;
}

export async function createShipmentCommentGraphQL(
  shipmentId: Shipment["id"],
  input: ShipmentCommentInput,
) {
  const data = await requestShipmentGraphQL({
    document: CreateShipmentCommentDocument,
    operationName: "CreateShipmentComment",
    variables: { shipmentId, input },
  });
  return data.createShipmentComment as ShipmentComment;
}

export async function updateShipmentCommentGraphQL(
  shipmentId: Shipment["id"],
  commentId: ShipmentComment["id"],
  input: ShipmentCommentUpdateInput,
) {
  const data = await requestShipmentGraphQL({
    document: UpdateShipmentCommentDocument,
    operationName: "UpdateShipmentComment",
    variables: { shipmentId, commentId, input },
  });
  return data.updateShipmentComment as ShipmentComment;
}

export async function deleteShipmentCommentGraphQL(
  shipmentId: Shipment["id"],
  commentId: ShipmentComment["id"],
) {
  const data = await requestShipmentGraphQL({
    document: DeleteShipmentCommentDocument,
    operationName: "DeleteShipmentComment",
    variables: { shipmentId, commentId },
  });
  return data.deleteShipmentComment;
}

function connectionToLimitOffset<T>(connection: ShipmentConnection): GenericLimitOffsetResponse<T> {
  const results = (connection.edges ?? []).map((edge) => edge.node as T);
  const totalCount = connection.totalCount ?? results.length;
  const hasNextPage = connection.pageInfo?.hasNextPage ?? false;
  const endCursor = connection.pageInfo?.endCursor ?? null;

  return {
    results,
    count: totalCount,
    next: hasNextPage ? endCursor : null,
    prev: null,
    pageInfo: {
      mode: "cursor",
      hasNextPage,
      endCursor,
      totalCount,
    },
  };
}

function toShipmentInput(payload: Shipment | ShipmentCreateInput | ShipmentUpdateInput): ShipmentInput {
  return {
    sourceDocumentId: payload.sourceDocumentId,
    serviceTypeId: payload.serviceTypeId,
    shipmentTypeId: payload.shipmentTypeId,
    customerId: payload.customerId,
    tractorTypeId: payload.tractorTypeId,
    trailerTypeId: payload.trailerTypeId,
    ownerId: payload.ownerId,
    enteredById: payload.enteredById,
    canceledById: payload.canceledById,
    formulaTemplateId: payload.formulaTemplateId,
    consolidationGroupId: payload.consolidationGroupId,
    status: payload.status,
    tenderStatus: payload.tenderStatus ?? undefined,
    entryMethod: payload.entryMethod,
    proNumber: payload.proNumber,
    bol: payload.bol,
    cancelReason: payload.cancelReason,
    otherChargeAmount: String(payload.otherChargeAmount ?? "0"),
    freightChargeAmount: String(payload.freightChargeAmount ?? "0"),
    baseRate: String(payload.baseRate ?? "0"),
    totalChargeAmount: String(payload.totalChargeAmount ?? "0"),
    pieces: payload.pieces,
    weight: payload.weight,
    temperatureMin: payload.temperatureMin,
    temperatureMax: payload.temperatureMax,
    actualDeliveryDate: payload.actualDeliveryDate,
    actualShipDate: payload.actualShipDate,
    canceledAt: payload.canceledAt,
    billingTransferStatus: payload.billingTransferStatus,
    transferredToBillingAt: payload.transferredToBillingAt,
    markedReadyToBillAt: payload.markedReadyToBillAt,
    billedAt: payload.billedAt,
    ratingUnit: payload.ratingUnit,
    ratingDetail: payload.ratingDetail,
    version: "version" in payload ? payload.version : undefined,
    moves: payload.moves?.map(toShipmentMoveInput) ?? [],
    additionalCharges: payload.additionalCharges?.map(toAdditionalChargeInput) ?? [],
    commodities: payload.commodities?.map(toCommodityInput) ?? [],
  };
}

function toShipmentMoveInput(move: Shipment["moves"][number]): ShipmentMoveInput {
  return {
    id: move.id,
    shipmentId: move.shipmentId,
    status: move.status,
    loaded: move.loaded,
    sequence: move.sequence,
    distance: move.distance,
    distanceSource: move.distanceSource,
    distanceProvider: move.distanceProvider,
    distanceCalculatedAt: move.distanceCalculatedAt,
    distanceRouteSignature: move.distanceRouteSignature,
    distanceDataVersion: move.distanceDataVersion,
    distanceRoutingType: move.distanceRoutingType,
    distanceUnits: move.distanceUnits,
    distanceMetadata: move.distanceMetadata,
    version: move.version,
    stops: move.stops?.map((stop) => ({
      id: stop.id,
      shipmentMoveId: stop.shipmentMoveId,
      locationId: stop.locationId,
      status: stop.status,
      type: stop.type,
      scheduleType: stop.scheduleType,
      sequence: stop.sequence,
      pieces: stop.pieces,
      weight: stop.weight,
      scheduledWindowStart: stop.scheduledWindowStart,
      scheduledWindowEnd: stop.scheduledWindowEnd,
      actualArrival: stop.actualArrival,
      actualDeparture: stop.actualDeparture,
      countLateOverride: stop.countLateOverride,
      countDetentionOverride: stop.countDetentionOverride,
      addressLine: stop.addressLine,
      version: stop.version,
    })) ?? [],
  };
}

function toAdditionalChargeInput(
  charge: Shipment["additionalCharges"][number],
): ShipmentAdditionalChargeInput {
  return {
    id: charge.id,
    shipmentId: "shipmentId" in charge ? (charge.shipmentId as string | undefined) : undefined,
    accessorialChargeId: charge.accessorialChargeId,
    isSystemGenerated: charge.isSystemGenerated,
    method: charge.method,
    amount: String(charge.amount ?? "0"),
    unit: charge.unit,
    version: charge.version,
  };
}

function toCommodityInput(commodity: Shipment["commodities"][number]): ShipmentCommodityInput {
  return {
    id: commodity.id,
    shipmentId:
      "shipmentId" in commodity ? (commodity.shipmentId as string | undefined) : undefined,
    commodityId: commodity.commodityId,
    pieces: commodity.pieces,
    weight: commodity.weight,
    version: commodity.version,
  };
}
