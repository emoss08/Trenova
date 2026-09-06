import {
  ActiveWorkerChecklistTemplatesDocument,
  ArchiveWorkerChecklistTemplateDocument,
  CancelWorkerChecklistDocument,
  CompleteWorkerChecklistItemDocument,
  CreateWorkerChecklistTemplateDocument,
  MarkWorkerChecklistItemNotApplicableDocument,
  ReopenWorkerChecklistItemDocument,
  RestoreWorkerChecklistTemplateDocument,
  SkipWorkerChecklistItemDocument,
  StartWorkerChecklistDocument,
  UpdateWorkerChecklistTemplateDocument,
  WorkerChecklistTemplateTableDocument,
  WorkerChecklistsDocument,
  type ActiveWorkerChecklistTemplatesQuery,
  type ActiveWorkerChecklistTemplatesQueryVariables,
  type ArchiveWorkerChecklistTemplateMutation,
  type ArchiveWorkerChecklistTemplateMutationVariables,
  type CancelWorkerChecklistInput,
  type CancelWorkerChecklistMutation,
  type CancelWorkerChecklistMutationVariables,
  type CompleteWorkerChecklistItemMutation,
  type CompleteWorkerChecklistItemMutationVariables,
  type CreateWorkerChecklistTemplateMutation,
  type CreateWorkerChecklistTemplateMutationVariables,
  type MarkWorkerChecklistItemNotApplicableMutation,
  type MarkWorkerChecklistItemNotApplicableMutationVariables,
  type ReopenWorkerChecklistItemMutation,
  type ReopenWorkerChecklistItemMutationVariables,
  type RestoreWorkerChecklistTemplateMutation,
  type RestoreWorkerChecklistTemplateMutationVariables,
  type SkipWorkerChecklistItemMutation,
  type SkipWorkerChecklistItemMutationVariables,
  type StartWorkerChecklistInput,
  type StartWorkerChecklistMutation,
  type StartWorkerChecklistMutationVariables,
  type UpdateWorkerChecklistTemplateMutation,
  type UpdateWorkerChecklistTemplateMutationVariables,
  type WorkerChecklistFieldsFragment,
  type WorkerChecklistItemActionInput,
  type WorkerChecklistTemplateFieldsFragment,
  type WorkerChecklistTemplateInput,
  type WorkerChecklistsQuery,
  type WorkerChecklistsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type WorkerChecklistTemplateRow = WorkerChecklistTemplateFieldsFragment;
export type WorkerChecklistRow = WorkerChecklistFieldsFragment;
export type WorkerChecklistItemRow = WorkerChecklistRow["items"][number];

export const WORKER_CHECKLIST_TEMPLATE_LIST_KEY = "worker-checklist-template-list";
export const WORKER_CHECKLIST_TEMPLATES_KEY = "worker-checklist-templates";
export const WORKER_CHECKLISTS_KEY = "worker-checklists";

export const workerChecklistTemplateTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: WorkerChecklistTemplateTableDocument,
  operationName: "WorkerChecklistTemplateTable",
  connectionKey: "workerChecklistTemplates",
});

export async function fetchActiveWorkerChecklistTemplates(options?: {
  signal?: AbortSignal;
}): Promise<WorkerChecklistTemplateRow[]> {
  const data = await requestGraphQL<
    ActiveWorkerChecklistTemplatesQuery,
    ActiveWorkerChecklistTemplatesQueryVariables
  >({
    document: ActiveWorkerChecklistTemplatesDocument,
    operationName: "ActiveWorkerChecklistTemplates",
    signal: options?.signal,
  });
  return data.activeWorkerChecklistTemplates as WorkerChecklistTemplateRow[];
}

export async function fetchWorkerChecklists(
  workerId: string,
  includeClosed = true,
  options?: { signal?: AbortSignal },
): Promise<WorkerChecklistRow[]> {
  const data = await requestGraphQL<WorkerChecklistsQuery, WorkerChecklistsQueryVariables>({
    document: WorkerChecklistsDocument,
    operationName: "WorkerChecklists",
    variables: { workerId, includeClosed },
    signal: options?.signal,
  });
  return data.workerChecklists as WorkerChecklistRow[];
}

export async function createWorkerChecklistTemplate(
  input: WorkerChecklistTemplateInput,
): Promise<WorkerChecklistTemplateRow> {
  const data = await requestGraphQL<
    CreateWorkerChecklistTemplateMutation,
    CreateWorkerChecklistTemplateMutationVariables
  >({
    document: CreateWorkerChecklistTemplateDocument,
    operationName: "CreateWorkerChecklistTemplate",
    variables: { input },
  });
  return data.createWorkerChecklistTemplate as WorkerChecklistTemplateRow;
}

export async function updateWorkerChecklistTemplate(
  id: string,
  input: WorkerChecklistTemplateInput,
): Promise<WorkerChecklistTemplateRow> {
  const data = await requestGraphQL<
    UpdateWorkerChecklistTemplateMutation,
    UpdateWorkerChecklistTemplateMutationVariables
  >({
    document: UpdateWorkerChecklistTemplateDocument,
    operationName: "UpdateWorkerChecklistTemplate",
    variables: { id, input },
  });
  return data.updateWorkerChecklistTemplate as WorkerChecklistTemplateRow;
}

export async function archiveWorkerChecklistTemplate(
  id: string,
  version?: number,
): Promise<WorkerChecklistTemplateRow> {
  const data = await requestGraphQL<
    ArchiveWorkerChecklistTemplateMutation,
    ArchiveWorkerChecklistTemplateMutationVariables
  >({
    document: ArchiveWorkerChecklistTemplateDocument,
    operationName: "ArchiveWorkerChecklistTemplate",
    variables: { id, version },
  });
  return data.archiveWorkerChecklistTemplate as WorkerChecklistTemplateRow;
}

export async function restoreWorkerChecklistTemplate(
  id: string,
  version?: number,
): Promise<WorkerChecklistTemplateRow> {
  const data = await requestGraphQL<
    RestoreWorkerChecklistTemplateMutation,
    RestoreWorkerChecklistTemplateMutationVariables
  >({
    document: RestoreWorkerChecklistTemplateDocument,
    operationName: "RestoreWorkerChecklistTemplate",
    variables: { id, version },
  });
  return data.restoreWorkerChecklistTemplate as WorkerChecklistTemplateRow;
}

export async function startWorkerChecklist(
  input: StartWorkerChecklistInput,
): Promise<WorkerChecklistRow> {
  const data = await requestGraphQL<
    StartWorkerChecklistMutation,
    StartWorkerChecklistMutationVariables
  >({
    document: StartWorkerChecklistDocument,
    operationName: "StartWorkerChecklist",
    variables: { input },
  });
  return data.startWorkerChecklist as WorkerChecklistRow;
}

export async function completeWorkerChecklistItem(
  input: WorkerChecklistItemActionInput,
): Promise<WorkerChecklistRow> {
  const data = await requestGraphQL<
    CompleteWorkerChecklistItemMutation,
    CompleteWorkerChecklistItemMutationVariables
  >({
    document: CompleteWorkerChecklistItemDocument,
    operationName: "CompleteWorkerChecklistItem",
    variables: { input },
  });
  return data.completeWorkerChecklistItem as WorkerChecklistRow;
}

export async function skipWorkerChecklistItem(
  input: WorkerChecklistItemActionInput,
): Promise<WorkerChecklistRow> {
  const data = await requestGraphQL<
    SkipWorkerChecklistItemMutation,
    SkipWorkerChecklistItemMutationVariables
  >({
    document: SkipWorkerChecklistItemDocument,
    operationName: "SkipWorkerChecklistItem",
    variables: { input },
  });
  return data.skipWorkerChecklistItem as WorkerChecklistRow;
}

export async function markWorkerChecklistItemNotApplicable(
  input: WorkerChecklistItemActionInput,
): Promise<WorkerChecklistRow> {
  const data = await requestGraphQL<
    MarkWorkerChecklistItemNotApplicableMutation,
    MarkWorkerChecklistItemNotApplicableMutationVariables
  >({
    document: MarkWorkerChecklistItemNotApplicableDocument,
    operationName: "MarkWorkerChecklistItemNotApplicable",
    variables: { input },
  });
  return data.markWorkerChecklistItemNotApplicable as WorkerChecklistRow;
}

export async function reopenWorkerChecklistItem(
  id: string,
  version?: number,
): Promise<WorkerChecklistRow> {
  const data = await requestGraphQL<
    ReopenWorkerChecklistItemMutation,
    ReopenWorkerChecklistItemMutationVariables
  >({
    document: ReopenWorkerChecklistItemDocument,
    operationName: "ReopenWorkerChecklistItem",
    variables: { id, version },
  });
  return data.reopenWorkerChecklistItem as WorkerChecklistRow;
}

export async function cancelWorkerChecklist(
  input: CancelWorkerChecklistInput,
): Promise<WorkerChecklistRow> {
  const data = await requestGraphQL<
    CancelWorkerChecklistMutation,
    CancelWorkerChecklistMutationVariables
  >({
    document: CancelWorkerChecklistDocument,
    operationName: "CancelWorkerChecklist",
    variables: { input },
  });
  return data.cancelWorkerChecklist as WorkerChecklistRow;
}
