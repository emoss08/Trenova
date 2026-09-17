import {
  AcknowledgeCarrierIntelEventsDocument,
  ApplyCarrierIntelSuggestionsDocument,
  CarrierIntelEventDocument,
  CarrierIntelEventTableDocument,
  CarrierIntelOverridesDocument,
  CarrierIntelRawPayloadDocument,
  CarrierIntelReviewQueueDocument,
  CarrierIntelSettingsDocument,
  CarrierIntelSnapshotHistoryDocument,
  CarrierIntelSyncPlanDocument,
  CarrierIntelligenceDocument,
  GrantCarrierIntelOverrideDocument,
  MarkCarrierIntelReviewedDocument,
  ResolveCarrierIntelEventDocument,
  RevokeCarrierIntelOverrideDocument,
  SetCarrierMonitoringDocument,
  VetCarrierDocument,
  type ApplyCarrierIntelSuggestionsInput,
  type CarrierIntelDepth,
  type CarrierIntelEventFieldsFragment,
  type CarrierIntelEventStatus,
  type CarrierIntelFieldUpdateFieldsFragment,
  type CarrierIntelFindingFieldsFragment,
  type CarrierIntelInsuranceChangeFieldsFragment,
  type CarrierIntelOverrideFieldsFragment,
  type CarrierIntelProfileFieldsFragment,
  type CarrierIntelSnapshotFieldsFragment,
  type CarrierIntelReviewQueueQuery,
  type CarrierIntelSettingsQuery,
  type CarrierIntelSnapshotSummaryFieldsFragment,
  type CarrierIntelSyncPlanQuery,
  type CarrierIntelligenceQuery,
  type CarrierMonitoringEnrollmentFieldsFragment,
  type GrantCarrierIntelOverrideInput,
  type ResolveCarrierIntelEventInput,
  type VetCarrierMutation,
  CarrierEquipmentVerificationsDocument,
  CustomerBrokerIntelligenceDocument,
  MyCarrierIntelligenceDocument,
  OverrideCarrierEquipmentVerificationDocument,
  VerifyCarrierEquipmentDocument,
  VetCustomerBrokerDocument,
  type CarrierEquipmentVerificationFieldsFragment,
  type CustomerBrokerIntelligenceQuery,
  type MyCarrierIntelligenceQuery,
  type VerifyCarrierEquipmentInput,
  type VetCustomerBrokerMutation,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";

export type CarrierIntelFinding = UnmaskFragments<CarrierIntelFindingFieldsFragment>;
export type CarrierIntelProfile = UnmaskFragments<CarrierIntelProfileFieldsFragment>;
export type CarrierIntelSnapshotSummary =
  UnmaskFragments<CarrierIntelSnapshotSummaryFieldsFragment>;
export type CarrierIntelSnapshot = UnmaskFragments<CarrierIntelSnapshotFieldsFragment>;
export type CarrierIntelEvent = UnmaskFragments<CarrierIntelEventFieldsFragment>;
export type CarrierIntelOverride = UnmaskFragments<CarrierIntelOverrideFieldsFragment>;
export type CarrierMonitoringEnrollment =
  UnmaskFragments<CarrierMonitoringEnrollmentFieldsFragment>;
export type CarrierIntelFieldUpdate = UnmaskFragments<CarrierIntelFieldUpdateFieldsFragment>;
export type CarrierIntelInsuranceChange =
  UnmaskFragments<CarrierIntelInsuranceChangeFieldsFragment>;
export type CarrierIntelSyncPlan = UnmaskFragments<
  CarrierIntelSyncPlanQuery["carrierIntelSyncPlan"]
>;
export type CarrierIntelligenceResult = UnmaskFragments<CarrierIntelligenceQuery>;
export type CarrierIntelligenceCarrier = NonNullable<CarrierIntelligenceResult["carrier"]>;
export type CarrierIntelVetResult = UnmaskFragments<VetCarrierMutation["vetCarrier"]>;
export type CarrierIntelSettings = UnmaskFragments<CarrierIntelSettingsQuery>;
export type CarrierIntelControl = CarrierIntelSettings["carrierIntelControl"];
export type CarrierIntelProviderInfo = CarrierIntelSettings["carrierIntelProvider"];
export type CarrierIntelRuleDefinition = CarrierIntelSettings["carrierIntelRuleCatalog"][number];

export type CarrierIntelEventPage = {
  events: CarrierIntelEvent[];
  totalCount: number | null;
  hasNextPage: boolean;
  endCursor: string | null;
};

export const CARRIER_INTELLIGENCE_KEY = "carrier-intelligence";
export const CARRIER_INTEL_HISTORY_KEY = "carrier-intel-history";
export const CARRIER_INTEL_SYNC_PLAN_KEY = "carrier-intel-sync-plan";
export const CARRIER_INTEL_OVERRIDES_KEY = "carrier-intel-overrides";
export const CARRIER_INTEL_EVENTS_KEY = "carrier-intel-events";
export const CARRIER_INTEL_RAW_PAYLOAD_KEY = "carrier-intel-raw-payload";
export const CARRIER_INTEL_REVIEW_QUEUE_KEY = "carrier-intel-review-queue";

type RequestOptions = { signal?: AbortSignal };

function unmask<T>(value: T): UnmaskFragments<T> {
  return value as unknown as UnmaskFragments<T>;
}

export async function fetchCarrierIntelSettings(
  options?: RequestOptions,
): Promise<CarrierIntelSettings> {
  const data = await requestGraphQL({
    document: CarrierIntelSettingsDocument,
    operationName: "CarrierIntelSettings",
    signal: options?.signal,
  });
  return unmask(data);
}

export type CarrierIntelReviewQueueItem = UnmaskFragments<
  CarrierIntelReviewQueueQuery["carrierIntelReviewQueue"][number]
>;

export async function fetchCarrierIntelReviewQueue(
  limit: number,
  options?: RequestOptions,
): Promise<CarrierIntelReviewQueueItem[]> {
  const data = await requestGraphQL({
    document: CarrierIntelReviewQueueDocument,
    operationName: "CarrierIntelReviewQueue",
    variables: { limit },
    signal: options?.signal,
  });
  return unmask(data.carrierIntelReviewQueue);
}

export async function fetchCarrierIntelligence(
  carrierId: string,
  options?: RequestOptions,
): Promise<CarrierIntelligenceResult> {
  const data = await requestGraphQL({
    document: CarrierIntelligenceDocument,
    operationName: "CarrierIntelligence",
    variables: { carrierId },
    signal: options?.signal,
  });
  return unmask(data);
}

export async function fetchCarrierIntelSnapshotHistory(
  carrierId: string,
  limit: number,
  options?: RequestOptions,
): Promise<CarrierIntelSnapshotSummary[]> {
  const data = await requestGraphQL({
    document: CarrierIntelSnapshotHistoryDocument,
    operationName: "CarrierIntelSnapshotHistory",
    variables: { carrierId, limit },
    signal: options?.signal,
  });
  return unmask(data.carrierIntelSnapshotHistory);
}

export async function fetchCarrierIntelSyncPlan(
  carrierId: string,
  options?: RequestOptions,
): Promise<CarrierIntelSyncPlan> {
  const data = await requestGraphQL({
    document: CarrierIntelSyncPlanDocument,
    operationName: "CarrierIntelSyncPlan",
    variables: { carrierId },
    signal: options?.signal,
  });
  return unmask(data.carrierIntelSyncPlan);
}

export async function fetchCarrierIntelOverrides(
  carrierId: string,
  options?: RequestOptions,
): Promise<CarrierIntelOverride[]> {
  const data = await requestGraphQL({
    document: CarrierIntelOverridesDocument,
    operationName: "CarrierIntelOverrides",
    variables: { carrierId },
    signal: options?.signal,
  });
  return unmask(data.carrierIntelOverrides);
}

export async function fetchCarrierIntelEvent(
  id: string,
  options?: RequestOptions,
): Promise<CarrierIntelEvent | null> {
  const data = await requestGraphQL({
    document: CarrierIntelEventDocument,
    operationName: "CarrierIntelEvent",
    variables: { id },
    signal: options?.signal,
  });
  return data.carrierIntelEvent ? unmask(data.carrierIntelEvent) : null;
}

export type FetchCarrierIntelEventsArgs = {
  carrierId: string;
  statuses?: CarrierIntelEventStatus[];
  first: number;
  after?: string | null;
};

export async function fetchCarrierIntelEvents(
  { carrierId, statuses, first, after }: FetchCarrierIntelEventsArgs,
  options?: RequestOptions,
): Promise<CarrierIntelEventPage> {
  const data = await requestGraphQL({
    document: CarrierIntelEventTableDocument,
    operationName: "CarrierIntelEventTable",
    variables: {
      input: { first, after: after ?? null },
      filter: {
        carrierId,
        statuses: statuses && statuses.length > 0 ? statuses : null,
      },
    },
    signal: options?.signal,
  });
  const connection = unmask(data.carrierIntelEvents);
  return {
    events: connection.edges.map((edge) => edge.node),
    totalCount: connection.totalCount,
    hasNextPage: connection.pageInfo.hasNextPage,
    endCursor: connection.pageInfo.endCursor,
  };
}

export async function fetchCarrierIntelRawPayload(
  carrierId: string,
  snapshotId: string,
  options?: RequestOptions,
): Promise<unknown> {
  const data = await requestGraphQL({
    document: CarrierIntelRawPayloadDocument,
    operationName: "CarrierIntelRawPayload",
    variables: { carrierId, snapshotId },
    signal: options?.signal,
  });
  return data.carrierIntelRawPayload;
}

export type VetCarrierArgs = {
  carrierId: string;
  depth: CarrierIntelDepth | null;
  force: boolean;
};

export async function vetCarrier({
  carrierId,
  depth,
  force,
}: VetCarrierArgs): Promise<CarrierIntelVetResult> {
  const data = await requestGraphQL({
    document: VetCarrierDocument,
    operationName: "VetCarrier",
    variables: { carrierId, depth, force },
  });
  return unmask(data.vetCarrier);
}

export async function setCarrierMonitoring(
  carrierIds: string[],
  enabled: boolean,
): Promise<number> {
  const data = await requestGraphQL({
    document: SetCarrierMonitoringDocument,
    operationName: "SetCarrierMonitoring",
    variables: { carrierIds, enabled },
  });
  return data.setCarrierMonitoring;
}

export async function markCarrierIntelReviewed(carrierId: string, note: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: MarkCarrierIntelReviewedDocument,
    operationName: "MarkCarrierIntelReviewed",
    variables: { carrierId, note },
  });
  return data.markCarrierIntelReviewed;
}

export async function grantCarrierIntelOverride(
  input: GrantCarrierIntelOverrideInput,
): Promise<CarrierIntelOverride> {
  const data = await requestGraphQL({
    document: GrantCarrierIntelOverrideDocument,
    operationName: "GrantCarrierIntelOverride",
    variables: { input },
  });
  return unmask(data.grantCarrierIntelOverride);
}

export async function revokeCarrierIntelOverride(
  id: string,
  reason: string,
): Promise<CarrierIntelOverride> {
  const data = await requestGraphQL({
    document: RevokeCarrierIntelOverrideDocument,
    operationName: "RevokeCarrierIntelOverride",
    variables: { id, reason },
  });
  return unmask(data.revokeCarrierIntelOverride);
}

export async function applyCarrierIntelSuggestions(
  input: ApplyCarrierIntelSuggestionsInput,
): Promise<number> {
  const data = await requestGraphQL({
    document: ApplyCarrierIntelSuggestionsDocument,
    operationName: "ApplyCarrierIntelSuggestions",
    variables: { input },
  });
  return data.applyCarrierIntelSuggestions;
}

export async function acknowledgeCarrierIntelEvents(ids: string[]): Promise<number> {
  const data = await requestGraphQL({
    document: AcknowledgeCarrierIntelEventsDocument,
    operationName: "AcknowledgeCarrierIntelEvents",
    variables: { ids },
  });
  return data.acknowledgeCarrierIntelEvents;
}

export async function resolveCarrierIntelEvent(
  input: ResolveCarrierIntelEventInput,
): Promise<CarrierIntelEvent> {
  const data = await requestGraphQL({
    document: ResolveCarrierIntelEventDocument,
    operationName: "ResolveCarrierIntelEvent",
    variables: { input },
  });
  return unmask(data.resolveCarrierIntelEvent);
}

export type CarrierEquipmentVerification =
  UnmaskFragments<CarrierEquipmentVerificationFieldsFragment>;
export type MyCarrierIntelligence = UnmaskFragments<
  MyCarrierIntelligenceQuery["myCarrierIntelligence"]
>;
export type CustomerBrokerIntelligence = UnmaskFragments<
  NonNullable<CustomerBrokerIntelligenceQuery["customer"]>
>;
export type CustomerBrokerVetResult = UnmaskFragments<
  VetCustomerBrokerMutation["vetCustomerBroker"]
>;

export const CARRIER_EQUIPMENT_VERIFICATIONS_KEY = "carrier-equipment-verifications";
export const MY_CARRIER_INTELLIGENCE_KEY = "my-carrier-intelligence";
export const CUSTOMER_BROKER_INTELLIGENCE_KEY = "customer-broker-intelligence";

export async function fetchCarrierEquipmentVerifications(
  carrierAssignmentId: string,
  options?: { signal?: AbortSignal },
): Promise<CarrierEquipmentVerification[]> {
  const data = await requestGraphQL({
    document: CarrierEquipmentVerificationsDocument,
    operationName: "CarrierEquipmentVerifications",
    variables: { carrierAssignmentId },
    signal: options?.signal,
  });
  return unmask(data.carrierEquipmentVerifications);
}

export async function verifyCarrierEquipment(
  input: VerifyCarrierEquipmentInput,
): Promise<CarrierEquipmentVerification> {
  const data = await requestGraphQL({
    document: VerifyCarrierEquipmentDocument,
    operationName: "VerifyCarrierEquipment",
    variables: { input },
  });
  return unmask(data.verifyCarrierEquipment);
}

export async function overrideCarrierEquipmentVerification(
  id: string,
  reason: string,
): Promise<CarrierEquipmentVerification> {
  const data = await requestGraphQL({
    document: OverrideCarrierEquipmentVerificationDocument,
    operationName: "OverrideCarrierEquipmentVerification",
    variables: { id, reason },
  });
  return unmask(data.overrideCarrierEquipmentVerification);
}

export async function fetchMyCarrierIntelligence(
  refresh: boolean,
  options?: { signal?: AbortSignal },
): Promise<MyCarrierIntelligence> {
  const data = await requestGraphQL({
    document: MyCarrierIntelligenceDocument,
    operationName: "MyCarrierIntelligence",
    variables: { refresh },
    signal: options?.signal,
  });
  return unmask(data.myCarrierIntelligence);
}

export async function fetchCustomerBrokerIntelligence(
  customerId: string,
  options?: { signal?: AbortSignal },
): Promise<CustomerBrokerIntelligence | null> {
  const data = await requestGraphQL({
    document: CustomerBrokerIntelligenceDocument,
    operationName: "CustomerBrokerIntelligence",
    variables: { customerId },
    signal: options?.signal,
  });
  return unmask(data.customer);
}

export async function vetCustomerBroker(
  customerId: string,
  force: boolean,
): Promise<CustomerBrokerVetResult> {
  const data = await requestGraphQL({
    document: VetCustomerBrokerDocument,
    operationName: "VetCustomerBroker",
    variables: { customerId, force },
  });
  return unmask(data.vetCustomerBroker);
}
