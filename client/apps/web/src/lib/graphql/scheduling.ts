import type {
  AssignShiftInput,
  AssignWorkerShiftMutation,
  AssignWorkerShiftMutationVariables,
  CreateShiftTemplateMutation,
  CreateShiftTemplateMutationVariables,
  EndWorkerShiftAssignmentMutation,
  EndWorkerShiftAssignmentMutationVariables,
  ProposeShiftSwapInput,
  ProposeShiftSwapMutation,
  ProposeShiftSwapMutationVariables,
  RotaFilterInput,
  RotaQuery,
  RotaQueryVariables,
  SetAvailabilityPreferenceInput,
  SetWorkerAvailabilityPreferenceMutation,
  SetWorkerAvailabilityPreferenceMutationVariables,
  ShiftSwapRequestsQuery,
  ShiftSwapRequestsQueryVariables,
  ShiftTemplateInput,
  ShiftTemplatesQuery,
  ShiftTemplatesQueryVariables,
  TransitionShiftSwapInput,
  TransitionShiftSwapMutation,
  TransitionShiftSwapMutationVariables,
  UpdateShiftTemplateMutation,
  UpdateShiftTemplateMutationVariables,
  WorkerAvailabilityPreferencesQuery,
  WorkerAvailabilityPreferencesQueryVariables,
  WorkerShiftAssignmentsQuery,
  WorkerShiftAssignmentsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  AssignWorkerShiftDocument,
  CreateShiftTemplateDocument,
  EndWorkerShiftAssignmentDocument,
  ProposeShiftSwapDocument,
  RotaDocument,
  SetWorkerAvailabilityPreferenceDocument,
  ShiftSwapRequestsDocument,
  ShiftTemplatesDocument,
  TransitionShiftSwapDocument,
  UpdateShiftTemplateDocument,
  WorkerAvailabilityPreferencesDocument,
  WorkerShiftAssignmentsDocument,
} from "@trenova/graphql/generated/documents/scheduling/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { rootFields, type RootFields } from "@trenova/shared/lib/reference-access";

/*
 * The shift templates list reads each template as its root record, so its pattern is never
 * withheld; the schema marks it nullable for where a template is linked from a worker's
 * assignments (docs/engineering/reference-views.md).
 */
const SHIFT_TEMPLATE_ROOT_FIELDS = ["cycleWeeks", "daysOfWeek"] as const;

export type ShiftTemplateRow = RootFields<
  ShiftTemplatesQuery["shiftTemplates"][number],
  (typeof SHIFT_TEMPLATE_ROOT_FIELDS)[number]
>;
export type RotaBoard = RotaQuery["rota"];
export type RotaBoardRow = RotaBoard["rows"][number];
export type RotaBoardDay = RotaBoardRow["days"][number];
export type ShiftAssignmentRow = WorkerShiftAssignmentsQuery["workerShiftAssignments"][number];
export type AvailabilityPreferenceRow =
  WorkerAvailabilityPreferencesQuery["workerAvailabilityPreferences"][number];
export type ShiftSwapRow = ShiftSwapRequestsQuery["shiftSwapRequests"][number];

export const SHIFT_TEMPLATES_KEY = "shift-templates";
export const ROTA_KEY = "rota";
export const SHIFT_ASSIGNMENTS_KEY = "shift-assignments";
export const AVAILABILITY_PREFERENCES_KEY = "availability-preferences";
export const SHIFT_SWAPS_KEY = "shift-swaps";

export async function fetchShiftTemplates(
  args: { activeOnly?: boolean; limit?: number } = {},
  options?: { signal?: AbortSignal },
): Promise<ShiftTemplateRow[]> {
  const data = await requestGraphQL<ShiftTemplatesQuery, ShiftTemplatesQueryVariables>({
    document: ShiftTemplatesDocument,
    operationName: "ShiftTemplates",
    variables: {
      activeOnly: args.activeOnly ?? null,
      limit: args.limit ?? null,
    },
    signal: options?.signal,
  });
  return data.shiftTemplates.map((template) =>
    rootFields(template, "ShiftTemplate", SHIFT_TEMPLATE_ROOT_FIELDS),
  );
}

export async function fetchRota(
  filter: RotaFilterInput,
  options?: { signal?: AbortSignal },
): Promise<RotaBoard> {
  const data = await requestGraphQL<RotaQuery, RotaQueryVariables>({
    document: RotaDocument,
    operationName: "Rota",
    variables: { filter },
    signal: options?.signal,
  });
  return data.rota;
}

export async function fetchWorkerShiftAssignments(
  workerId: string,
  activeOnly = false,
  options?: { signal?: AbortSignal },
): Promise<ShiftAssignmentRow[]> {
  const data = await requestGraphQL<
    WorkerShiftAssignmentsQuery,
    WorkerShiftAssignmentsQueryVariables
  >({
    document: WorkerShiftAssignmentsDocument,
    operationName: "WorkerShiftAssignments",
    variables: { workerId, activeOnly },
    signal: options?.signal,
  });
  return data.workerShiftAssignments;
}

export async function fetchWorkerAvailabilityPreferences(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<AvailabilityPreferenceRow[]> {
  const data = await requestGraphQL<
    WorkerAvailabilityPreferencesQuery,
    WorkerAvailabilityPreferencesQueryVariables
  >({
    document: WorkerAvailabilityPreferencesDocument,
    operationName: "WorkerAvailabilityPreferences",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerAvailabilityPreferences;
}

export async function fetchShiftSwapRequests(
  args: { workerId?: string; openOnly?: boolean; since?: number; limit?: number } = {},
  options?: { signal?: AbortSignal },
): Promise<ShiftSwapRow[]> {
  const data = await requestGraphQL<ShiftSwapRequestsQuery, ShiftSwapRequestsQueryVariables>({
    document: ShiftSwapRequestsDocument,
    operationName: "ShiftSwapRequests",
    variables: {
      workerId: args.workerId ?? null,
      openOnly: args.openOnly ?? null,
      since: args.since ?? null,
      limit: args.limit ?? null,
    },
    signal: options?.signal,
  });
  return data.shiftSwapRequests;
}

export async function createShiftTemplate(input: ShiftTemplateInput) {
  const data = await requestGraphQL<
    CreateShiftTemplateMutation,
    CreateShiftTemplateMutationVariables
  >({
    document: CreateShiftTemplateDocument,
    operationName: "CreateShiftTemplate",
    variables: { input },
  });
  return data.createShiftTemplate;
}

export async function updateShiftTemplate(id: string, input: ShiftTemplateInput) {
  const data = await requestGraphQL<
    UpdateShiftTemplateMutation,
    UpdateShiftTemplateMutationVariables
  >({
    document: UpdateShiftTemplateDocument,
    operationName: "UpdateShiftTemplate",
    variables: { id, input },
  });
  return data.updateShiftTemplate;
}

export async function assignWorkerShift(input: AssignShiftInput) {
  const data = await requestGraphQL<AssignWorkerShiftMutation, AssignWorkerShiftMutationVariables>({
    document: AssignWorkerShiftDocument,
    operationName: "AssignWorkerShift",
    variables: { input },
  });
  return data.assignWorkerShift;
}

export async function endWorkerShiftAssignment(id: string, effectiveTo: number) {
  const data = await requestGraphQL<
    EndWorkerShiftAssignmentMutation,
    EndWorkerShiftAssignmentMutationVariables
  >({
    document: EndWorkerShiftAssignmentDocument,
    operationName: "EndWorkerShiftAssignment",
    variables: { id, effectiveTo },
  });
  return data.endWorkerShiftAssignment;
}

export async function setWorkerAvailabilityPreference(input: SetAvailabilityPreferenceInput) {
  const data = await requestGraphQL<
    SetWorkerAvailabilityPreferenceMutation,
    SetWorkerAvailabilityPreferenceMutationVariables
  >({
    document: SetWorkerAvailabilityPreferenceDocument,
    operationName: "SetWorkerAvailabilityPreference",
    variables: { input },
  });
  return data.setWorkerAvailabilityPreference;
}

export async function proposeShiftSwap(input: ProposeShiftSwapInput) {
  const data = await requestGraphQL<ProposeShiftSwapMutation, ProposeShiftSwapMutationVariables>({
    document: ProposeShiftSwapDocument,
    operationName: "ProposeShiftSwap",
    variables: { input },
  });
  return data.proposeShiftSwap;
}

export async function transitionShiftSwap(input: TransitionShiftSwapInput) {
  const data = await requestGraphQL<
    TransitionShiftSwapMutation,
    TransitionShiftSwapMutationVariables
  >({
    document: TransitionShiftSwapDocument,
    operationName: "TransitionShiftSwap",
    variables: { input },
  });
  return data.transitionShiftSwap;
}
