import {
  CreateRecurringShipmentDocument,
  GenerateRecurringShipmentDocument,
  MatchRecurringShipmentsDocument,
  RecurringShipmentRunsDocument,
  UpdateRecurringShipmentDocument,
  UpdateRecurringShipmentStatusDocument,
  type MatchRecurringShipmentsInput,
  type RecurringShipmentInput,
  type RecurringShipmentStatus,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export async function listRecurringShipmentRunsGraphQL(
  recurringShipmentId: string,
  first: number,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: RecurringShipmentRunsDocument,
    operationName: "RecurringShipmentRuns",
    variables: { recurringShipmentId, first },
    signal: options?.signal,
  });
  return data.recurringShipmentRuns;
}

export async function matchRecurringShipmentsGraphQL(
  input: MatchRecurringShipmentsInput,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: MatchRecurringShipmentsDocument,
    operationName: "MatchRecurringShipments",
    variables: { input },
    signal: options?.signal,
  });
  return data.matchRecurringShipments;
}

export async function createRecurringShipmentGraphQL(input: RecurringShipmentInput) {
  const data = await requestGraphQL({
    document: CreateRecurringShipmentDocument,
    operationName: "CreateRecurringShipment",
    variables: { input },
  });
  return data.createRecurringShipment;
}

export async function updateRecurringShipmentGraphQL(id: string, input: RecurringShipmentInput) {
  const data = await requestGraphQL({
    document: UpdateRecurringShipmentDocument,
    operationName: "UpdateRecurringShipment",
    variables: { id, input },
  });
  return data.updateRecurringShipment;
}

export async function updateRecurringShipmentStatusGraphQL(
  id: string,
  status: RecurringShipmentStatus,
  version: number,
) {
  const data = await requestGraphQL({
    document: UpdateRecurringShipmentStatusDocument,
    operationName: "UpdateRecurringShipmentStatus",
    variables: { id, status, version },
  });
  return data.updateRecurringShipmentStatus;
}

export async function generateRecurringShipmentGraphQL(id: string, occurrenceAt?: number | null) {
  const data = await requestGraphQL({
    document: GenerateRecurringShipmentDocument,
    operationName: "GenerateRecurringShipment",
    variables: { id, occurrenceAt: occurrenceAt ?? null },
  });
  return data.generateRecurringShipment;
}
