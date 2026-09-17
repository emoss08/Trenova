import {
  CarrierIntelCostEstimateDocument,
  CarrierIntelMonitoringStatusDocument,
  CarrierIntelUsageDocument,
  ResumeCarrierIntelMonitoringDocument,
  SwitchCarrierIntelProviderDocument,
  UpdateCarrierIntelControlDocument,
  type CarrierIntelControlPatchInput,
  type CarrierIntelCostEstimateQuery,
  type CarrierIntelCostEstimateQueryVariables,
  type CarrierIntelMonitoringStatusQuery,
  type CarrierIntelUsageQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";
import type { CarrierIntelControl, CarrierIntelRuleDefinition } from "./carrier-intelligence";

export type CarrierIntelRuleParamDefinition = CarrierIntelRuleDefinition["params"][number];
export type CarrierIntelCostEstimate = CarrierIntelCostEstimateQuery["carrierIntelCostEstimate"];
export type CarrierIntelCostEstimateParams = CarrierIntelCostEstimateQueryVariables;
export type CarrierIntelUsageSummary = CarrierIntelUsageQuery["carrierIntelUsage"];
export type CarrierIntelMonitoringStatus = UnmaskFragments<
  CarrierIntelMonitoringStatusQuery["carrierIntelMonitoringStatus"]
>;
export type CarrierIntelFeedState = CarrierIntelMonitoringStatus["feeds"][number];

type RequestOptions = { signal?: AbortSignal };

export async function fetchCarrierIntelCostEstimate(
  variables: CarrierIntelCostEstimateParams,
  options?: RequestOptions,
): Promise<CarrierIntelCostEstimate> {
  const data = await requestGraphQL({
    document: CarrierIntelCostEstimateDocument,
    operationName: "CarrierIntelCostEstimate",
    variables,
    signal: options?.signal,
  });
  return data.carrierIntelCostEstimate;
}

export async function fetchCarrierIntelUsage(
  month?: number,
  options?: RequestOptions,
): Promise<CarrierIntelUsageSummary> {
  const data = await requestGraphQL({
    document: CarrierIntelUsageDocument,
    operationName: "CarrierIntelUsage",
    variables: { month },
    signal: options?.signal,
  });
  return data.carrierIntelUsage;
}

export async function fetchCarrierIntelMonitoringStatus(
  options?: RequestOptions,
): Promise<CarrierIntelMonitoringStatus> {
  const data = await requestGraphQL({
    document: CarrierIntelMonitoringStatusDocument,
    operationName: "CarrierIntelMonitoringStatus",
    signal: options?.signal,
  });
  return data.carrierIntelMonitoringStatus as unknown as CarrierIntelMonitoringStatus;
}

export async function updateCarrierIntelControl(
  input: CarrierIntelControlPatchInput,
): Promise<CarrierIntelControl> {
  const data = await requestGraphQL({
    document: UpdateCarrierIntelControlDocument,
    operationName: "UpdateCarrierIntelControl",
    variables: { input },
  });
  return data.updateCarrierIntelControl as unknown as CarrierIntelControl;
}

export async function switchCarrierIntelProvider(provider: string): Promise<CarrierIntelControl> {
  const data = await requestGraphQL({
    document: SwitchCarrierIntelProviderDocument,
    operationName: "SwitchCarrierIntelProvider",
    variables: { provider },
  });
  return data.switchCarrierIntelProvider as unknown as CarrierIntelControl;
}

export async function resumeCarrierIntelMonitoring(): Promise<boolean> {
  const data = await requestGraphQL({
    document: ResumeCarrierIntelMonitoringDocument,
    operationName: "ResumeCarrierIntelMonitoring",
  });
  return data.resumeCarrierIntelMonitoring;
}
