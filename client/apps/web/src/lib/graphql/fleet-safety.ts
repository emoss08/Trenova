import {
  DeleteSafetyViolationDocument,
  FleetSafetyDocument,
  RecordSafetyViolationDocument,
  UpdateSafetyViolationDocument,
  WorkerSafetyViolationsDocument,
  type DeleteSafetyViolationMutation,
  type DeleteSafetyViolationMutationVariables,
  type FleetSafetyInput,
  type FleetSafetyQuery,
  type FleetSafetyQueryVariables,
  type RecordSafetyViolationInput,
  type RecordSafetyViolationMutation,
  type RecordSafetyViolationMutationVariables,
  type UpdateSafetyViolationInput,
  type UpdateSafetyViolationMutation,
  type UpdateSafetyViolationMutationVariables,
  type WorkerSafetyViolationsQuery,
  type WorkerSafetyViolationsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type FleetSafetySummary = FleetSafetyQuery["fleetSafety"];
export type FleetSafetyBasicRow = FleetSafetySummary["basics"][number];
export type FleetSafetyKindRow = FleetSafetySummary["kinds"][number];
export type FleetSafetyTerminalRow = FleetSafetySummary["terminals"][number];
export type FleetSafetyTrendPoint = FleetSafetySummary["trend"][number];
export type FleetSafetyRankRow = FleetSafetySummary["worst"][number];
export type SafetyViolationRow = WorkerSafetyViolationsQuery["workerSafetyViolations"][number];

export const FLEET_SAFETY_KEY = "fleet-safety";
export const SAFETY_VIOLATIONS_KEY = "safety-violations";

export async function fetchFleetSafety(
  input: FleetSafetyInput,
  options?: { signal?: AbortSignal },
): Promise<FleetSafetySummary> {
  const data = await requestGraphQL<FleetSafetyQuery, FleetSafetyQueryVariables>({
    document: FleetSafetyDocument,
    operationName: "FleetSafety",
    variables: { input },
    signal: options?.signal,
  });
  return data.fleetSafety;
}

export async function fetchSafetyViolations(
  args: {
    safetyEventId?: string;
    workerId?: string;
  },
  options?: { signal?: AbortSignal },
): Promise<SafetyViolationRow[]> {
  const data = await requestGraphQL<
    WorkerSafetyViolationsQuery,
    WorkerSafetyViolationsQueryVariables
  >({
    document: WorkerSafetyViolationsDocument,
    operationName: "WorkerSafetyViolations",
    variables: {
      safetyEventId: args.safetyEventId ?? null,
      workerId: args.workerId ?? null,
    },
    signal: options?.signal,
  });
  return data.workerSafetyViolations;
}

export async function recordSafetyViolation(input: RecordSafetyViolationInput) {
  const data = await requestGraphQL<
    RecordSafetyViolationMutation,
    RecordSafetyViolationMutationVariables
  >({
    document: RecordSafetyViolationDocument,
    operationName: "RecordSafetyViolation",
    variables: { input },
  });
  return data.recordSafetyViolation;
}

export async function updateSafetyViolation(input: UpdateSafetyViolationInput) {
  const data = await requestGraphQL<
    UpdateSafetyViolationMutation,
    UpdateSafetyViolationMutationVariables
  >({
    document: UpdateSafetyViolationDocument,
    operationName: "UpdateSafetyViolation",
    variables: { input },
  });
  return data.updateSafetyViolation;
}

export async function deleteSafetyViolation(id: string): Promise<boolean> {
  const data = await requestGraphQL<
    DeleteSafetyViolationMutation,
    DeleteSafetyViolationMutationVariables
  >({
    document: DeleteSafetyViolationDocument,
    operationName: "DeleteSafetyViolation",
    variables: { id },
  });
  return data.deleteSafetyViolation;
}
