import type {
  ActiveWorkerChecklistTemplatesQuery,
  ActiveWorkerChecklistTemplatesQueryVariables,
  ArchiveWorkerChecklistTemplateMutation,
  ArchiveWorkerChecklistTemplateMutationVariables,
  CancelWorkerChecklistInput,
  CancelWorkerChecklistMutation,
  CancelWorkerChecklistMutationVariables,
  CompleteWorkerChecklistItemMutation,
  CompleteWorkerChecklistItemMutationVariables,
  CreateWorkerChecklistTemplateMutation,
  CreateWorkerChecklistTemplateMutationVariables,
  MarkWorkerChecklistItemNotApplicableMutation,
  MarkWorkerChecklistItemNotApplicableMutationVariables,
  ReopenWorkerChecklistItemMutation,
  ReopenWorkerChecklistItemMutationVariables,
  RestoreWorkerChecklistTemplateMutation,
  RestoreWorkerChecklistTemplateMutationVariables,
  SkipWorkerChecklistItemMutation,
  SkipWorkerChecklistItemMutationVariables,
  StartWorkerChecklistInput,
  StartWorkerChecklistMutation,
  StartWorkerChecklistMutationVariables,
  UpdateWorkerChecklistTemplateMutation,
  UpdateWorkerChecklistTemplateMutationVariables,
  WorkerChecklistFieldsFragment,
  WorkerChecklistItemActionInput,
  WorkerChecklistTemplateFieldsFragment,
  WorkerChecklistTemplateInput,
  WorkerChecklistsQuery,
  WorkerChecklistsQueryVariables,
} from "@trenova/graphql/generated/graphql";
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
} from "@trenova/graphql/generated/documents/worker-checklist/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { RootFields } from "@trenova/shared/lib/reference-access";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type WorkerChecklistTemplateRow = WorkerChecklistTemplateFieldsFragment;
/**
 * A checklist read for itself (the worker's checklists, and each checklist a mutation returns):
 * the root record, so its progress is never withheld. The schema marks progress nullable for
 * where a checklist is linked from the worker's overview (docs/engineering/reference-views.md).
 */
export type WorkerChecklistRow = RootFields<WorkerChecklistFieldsFragment, "progress">;
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

/**
 * A worker's checklists, open first then closed newest first, as one list the server caps at
 * the newest 100. `includeClosed: false` reads only the open ones.
 */
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
