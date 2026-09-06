import {
  ActivePerformanceReviewTemplatesDocument,
  ArchivePerformanceReviewTemplateDocument,
  ClosePerformanceReviewDocument,
  CreatePerformanceReviewDocument,
  CreatePerformanceReviewTemplateDocument,
  DeletePerformanceReviewDocument,
  PerformanceReviewTemplateTableDocument,
  ReopenPerformanceReviewDocument,
  RestorePerformanceReviewTemplateDocument,
  SubmitPerformanceReviewDocument,
  UpdatePerformanceReviewDocument,
  UpdatePerformanceReviewTemplateDocument,
  WorkerPerformanceReviewsDocument,
  type ActivePerformanceReviewTemplatesQuery,
  type ActivePerformanceReviewTemplatesQueryVariables,
  type ArchivePerformanceReviewTemplateMutation,
  type ArchivePerformanceReviewTemplateMutationVariables,
  type ClosePerformanceReviewMutation,
  type ClosePerformanceReviewMutationVariables,
  type CreatePerformanceReviewInput,
  type CreatePerformanceReviewMutation,
  type CreatePerformanceReviewMutationVariables,
  type CreatePerformanceReviewTemplateMutation,
  type CreatePerformanceReviewTemplateMutationVariables,
  type DeletePerformanceReviewMutation,
  type DeletePerformanceReviewMutationVariables,
  type PerformanceReviewFieldsFragment,
  type PerformanceReviewStatusInput,
  type PerformanceReviewTemplateFieldsFragment,
  type PerformanceReviewTemplateInput,
  type ReopenPerformanceReviewMutation,
  type ReopenPerformanceReviewMutationVariables,
  type RestorePerformanceReviewTemplateMutation,
  type RestorePerformanceReviewTemplateMutationVariables,
  type SubmitPerformanceReviewMutation,
  type SubmitPerformanceReviewMutationVariables,
  type UpdatePerformanceReviewInput,
  type UpdatePerformanceReviewMutation,
  type UpdatePerformanceReviewMutationVariables,
  type UpdatePerformanceReviewTemplateMutation,
  type UpdatePerformanceReviewTemplateMutationVariables,
  type WorkerPerformanceReviewsQuery,
  type WorkerPerformanceReviewsQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type ReviewTemplateRow = PerformanceReviewTemplateFieldsFragment;
export type PerformanceReviewRow = PerformanceReviewFieldsFragment;

export const REVIEW_TEMPLATE_LIST_KEY = "review-template-list";
export const REVIEW_TEMPLATES_KEY = "review-templates";
export const WORKER_REVIEWS_KEY = "worker-reviews";

export const reviewTemplateTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: PerformanceReviewTemplateTableDocument,
  operationName: "PerformanceReviewTemplateTable",
  connectionKey: "performanceReviewTemplates",
});

export async function fetchActivePerformanceReviewTemplates(options?: {
  signal?: AbortSignal;
}): Promise<ReviewTemplateRow[]> {
  const data = await requestGraphQL<
    ActivePerformanceReviewTemplatesQuery,
    ActivePerformanceReviewTemplatesQueryVariables
  >({
    document: ActivePerformanceReviewTemplatesDocument,
    operationName: "ActivePerformanceReviewTemplates",
    signal: options?.signal,
  });
  return data.activePerformanceReviewTemplates as ReviewTemplateRow[];
}

export async function fetchWorkerPerformanceReviews(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<PerformanceReviewRow[]> {
  const data = await requestGraphQL<
    WorkerPerformanceReviewsQuery,
    WorkerPerformanceReviewsQueryVariables
  >({
    document: WorkerPerformanceReviewsDocument,
    operationName: "WorkerPerformanceReviews",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerPerformanceReviews as PerformanceReviewRow[];
}

export async function createPerformanceReviewTemplate(
  input: PerformanceReviewTemplateInput,
): Promise<ReviewTemplateRow> {
  const data = await requestGraphQL<
    CreatePerformanceReviewTemplateMutation,
    CreatePerformanceReviewTemplateMutationVariables
  >({
    document: CreatePerformanceReviewTemplateDocument,
    operationName: "CreatePerformanceReviewTemplate",
    variables: { input },
  });
  return data.createPerformanceReviewTemplate as ReviewTemplateRow;
}

export async function updatePerformanceReviewTemplate(
  id: string,
  input: PerformanceReviewTemplateInput,
): Promise<ReviewTemplateRow> {
  const data = await requestGraphQL<
    UpdatePerformanceReviewTemplateMutation,
    UpdatePerformanceReviewTemplateMutationVariables
  >({
    document: UpdatePerformanceReviewTemplateDocument,
    operationName: "UpdatePerformanceReviewTemplate",
    variables: { id, input },
  });
  return data.updatePerformanceReviewTemplate as ReviewTemplateRow;
}

export async function archivePerformanceReviewTemplate(
  id: string,
  version?: number,
): Promise<ReviewTemplateRow> {
  const data = await requestGraphQL<
    ArchivePerformanceReviewTemplateMutation,
    ArchivePerformanceReviewTemplateMutationVariables
  >({
    document: ArchivePerformanceReviewTemplateDocument,
    operationName: "ArchivePerformanceReviewTemplate",
    variables: { id, version },
  });
  return data.archivePerformanceReviewTemplate as ReviewTemplateRow;
}

export async function restorePerformanceReviewTemplate(
  id: string,
  version?: number,
): Promise<ReviewTemplateRow> {
  const data = await requestGraphQL<
    RestorePerformanceReviewTemplateMutation,
    RestorePerformanceReviewTemplateMutationVariables
  >({
    document: RestorePerformanceReviewTemplateDocument,
    operationName: "RestorePerformanceReviewTemplate",
    variables: { id, version },
  });
  return data.restorePerformanceReviewTemplate as ReviewTemplateRow;
}

export async function createPerformanceReview(
  input: CreatePerformanceReviewInput,
): Promise<PerformanceReviewRow> {
  const data = await requestGraphQL<
    CreatePerformanceReviewMutation,
    CreatePerformanceReviewMutationVariables
  >({
    document: CreatePerformanceReviewDocument,
    operationName: "CreatePerformanceReview",
    variables: { input },
  });
  return data.createPerformanceReview as PerformanceReviewRow;
}

export async function updatePerformanceReview(
  input: UpdatePerformanceReviewInput,
): Promise<PerformanceReviewRow> {
  const data = await requestGraphQL<
    UpdatePerformanceReviewMutation,
    UpdatePerformanceReviewMutationVariables
  >({
    document: UpdatePerformanceReviewDocument,
    operationName: "UpdatePerformanceReview",
    variables: { input },
  });
  return data.updatePerformanceReview as PerformanceReviewRow;
}

export async function submitPerformanceReview(
  input: PerformanceReviewStatusInput,
): Promise<PerformanceReviewRow> {
  const data = await requestGraphQL<
    SubmitPerformanceReviewMutation,
    SubmitPerformanceReviewMutationVariables
  >({
    document: SubmitPerformanceReviewDocument,
    operationName: "SubmitPerformanceReview",
    variables: { input },
  });
  return data.submitPerformanceReview as PerformanceReviewRow;
}

export async function reopenPerformanceReview(
  input: PerformanceReviewStatusInput,
): Promise<PerformanceReviewRow> {
  const data = await requestGraphQL<
    ReopenPerformanceReviewMutation,
    ReopenPerformanceReviewMutationVariables
  >({
    document: ReopenPerformanceReviewDocument,
    operationName: "ReopenPerformanceReview",
    variables: { input },
  });
  return data.reopenPerformanceReview as PerformanceReviewRow;
}

export async function closePerformanceReview(
  input: PerformanceReviewStatusInput,
): Promise<PerformanceReviewRow> {
  const data = await requestGraphQL<
    ClosePerformanceReviewMutation,
    ClosePerformanceReviewMutationVariables
  >({
    document: ClosePerformanceReviewDocument,
    operationName: "ClosePerformanceReview",
    variables: { input },
  });
  return data.closePerformanceReview as PerformanceReviewRow;
}

export async function deletePerformanceReview(id: string): Promise<boolean> {
  const data = await requestGraphQL<
    DeletePerformanceReviewMutation,
    DeletePerformanceReviewMutationVariables
  >({
    document: DeletePerformanceReviewDocument,
    operationName: "DeletePerformanceReview",
    variables: { id },
  });
  return data.deletePerformanceReview;
}
