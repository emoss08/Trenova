import type {
  ActiveTrainingCoursesQuery,
  ActiveTrainingCoursesQueryVariables,
  ArchiveTrainingCourseMutation,
  ArchiveTrainingCourseMutationVariables,
  AssignRequiredWorkerTrainingMutation,
  AssignRequiredWorkerTrainingMutationVariables,
  AssignWorkerTrainingInput,
  AssignWorkerTrainingMutation,
  BulkAssignTrainingInput,
  BulkAssignTrainingMutation,
  BulkAssignTrainingMutationVariables,
  AssignWorkerTrainingMutationVariables,
  AttachWorkerTrainingDocumentInput,
  AttachWorkerTrainingDocumentMutation,
  AttachWorkerTrainingDocumentMutationVariables,
  CancelWorkerTrainingInput,
  CancelWorkerTrainingMutation,
  CancelWorkerTrainingMutationVariables,
  CompleteWorkerTrainingInput,
  CompleteWorkerTrainingMutation,
  CompleteWorkerTrainingMutationVariables,
  CreateTrainingCourseMutation,
  CreateTrainingCourseMutationVariables,
  RestoreTrainingCourseMutation,
  RestoreTrainingCourseMutationVariables,
  TrainingCourseFieldsFragment,
  TrainingCourseInput,
  UpdateTrainingCourseMutation,
  UpdateTrainingCourseMutationVariables,
  WaiveWorkerTrainingInput,
  WaiveWorkerTrainingMutation,
  WaiveWorkerTrainingMutationVariables,
  WorkerTrainingRecordFieldsFragment,
  WorkerTrainingRecordQuery,
  WorkerTrainingRecordQueryVariables,
  WorkerTrainingRecordsQuery,
  WorkerTrainingRecordsQueryVariables,
  WorkerTrainingSummaryQuery,
  WorkerTrainingSummaryQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  ActiveTrainingCoursesDocument,
  ArchiveTrainingCourseDocument,
  AssignRequiredWorkerTrainingDocument,
  AssignWorkerTrainingDocument,
  AttachWorkerTrainingDocumentDocument,
  BulkAssignTrainingDocument,
  CancelWorkerTrainingDocument,
  CompleteWorkerTrainingDocument,
  CreateTrainingCourseDocument,
  RestoreTrainingCourseDocument,
  TrainingCourseTableDocument,
  UpdateTrainingCourseDocument,
  WaiveWorkerTrainingDocument,
  WorkerTrainingRecordDocument,
  WorkerTrainingRecordFieldsFragmentDoc,
  WorkerTrainingRecordsDocument,
  WorkerTrainingSummaryDocument,
} from "@trenova/graphql/generated/documents/worker-training/operations";
import { unmaskConnection, type Connection } from "./connection";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type TrainingCourseRow = TrainingCourseFieldsFragment;
export type WorkerTrainingRecordRow = WorkerTrainingRecordFieldsFragment;
type RawTrainingSummary = WorkerTrainingSummaryQuery["workerTrainingSummary"];
type RawTrainingSummaryItem = RawTrainingSummary["items"][number];
/**
 * A training slot. The server answers its course null only when the organization's plan leaves
 * out Training and performance, and a slot with no course is no slot, so it is left out.
 */
export type WorkerTrainingSummaryItem = Omit<RawTrainingSummaryItem, "course"> & {
  course: NonNullable<RawTrainingSummaryItem["course"]>;
};
export type WorkerTrainingSummary = Omit<RawTrainingSummary, "items"> & {
  items: WorkerTrainingSummaryItem[];
};

export const TRAINING_COURSE_LIST_KEY = "training-course-list";
export const TRAINING_COURSES_KEY = "training-courses";
export const WORKER_TRAINING_KEY = "worker-training";
export const WORKER_TRAINING_SUMMARY_KEY = "worker-training-summary";

export const trainingCourseTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: TrainingCourseTableDocument,
  operationName: "TrainingCourseTable",
  connectionKey: "trainingCourses",
});

export async function fetchActiveTrainingCourses(options?: {
  signal?: AbortSignal;
}): Promise<TrainingCourseRow[]> {
  const data = await requestGraphQL<
    ActiveTrainingCoursesQuery,
    ActiveTrainingCoursesQueryVariables
  >({
    document: ActiveTrainingCoursesDocument,
    operationName: "ActiveTrainingCourses",
    signal: options?.signal,
  });
  return data.activeTrainingCourses as TrainingCourseRow[];
}

/**
 * A page of a worker's training records, newest first, as the server's connection. Open
 * assignments and, with `includeClosed`, the closed ones too.
 */
export async function fetchWorkerTrainingRecords(
  variables: { workerId: string; includeClosed?: boolean; first?: number; after?: string },
  options?: { signal?: AbortSignal },
): Promise<Connection<WorkerTrainingRecordRow>> {
  const data = await requestGraphQL<
    WorkerTrainingRecordsQuery,
    WorkerTrainingRecordsQueryVariables
  >({
    document: WorkerTrainingRecordsDocument,
    operationName: "WorkerTrainingRecords",
    variables,
    signal: options?.signal,
  });
  return unmaskConnection(WorkerTrainingRecordFieldsFragmentDoc, data.workerTrainingRecords);
}

/** One training record by id: a deep link to a record outside the pages already read. */
export async function fetchWorkerTrainingRecord(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerTrainingRecordRow> {
  const data = await requestGraphQL<WorkerTrainingRecordQuery, WorkerTrainingRecordQueryVariables>({
    document: WorkerTrainingRecordDocument,
    operationName: "WorkerTrainingRecord",
    variables: { id },
    signal: options?.signal,
  });
  return data.workerTrainingRecord as WorkerTrainingRecordRow;
}

export async function fetchWorkerTrainingSummary(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerTrainingSummary> {
  const data = await requestGraphQL<
    WorkerTrainingSummaryQuery,
    WorkerTrainingSummaryQueryVariables
  >({
    document: WorkerTrainingSummaryDocument,
    operationName: "WorkerTrainingSummary",
    variables: { workerId },
    signal: options?.signal,
  });
  const summary = data.workerTrainingSummary;
  return {
    ...summary,
    items: summary.items.filter((item): item is WorkerTrainingSummaryItem => item.course != null),
  };
}

export async function createTrainingCourse(input: TrainingCourseInput): Promise<TrainingCourseRow> {
  const data = await requestGraphQL<
    CreateTrainingCourseMutation,
    CreateTrainingCourseMutationVariables
  >({
    document: CreateTrainingCourseDocument,
    operationName: "CreateTrainingCourse",
    variables: { input },
  });
  return data.createTrainingCourse as TrainingCourseRow;
}

export async function updateTrainingCourse(
  id: string,
  input: TrainingCourseInput,
): Promise<TrainingCourseRow> {
  const data = await requestGraphQL<
    UpdateTrainingCourseMutation,
    UpdateTrainingCourseMutationVariables
  >({
    document: UpdateTrainingCourseDocument,
    operationName: "UpdateTrainingCourse",
    variables: { id, input },
  });
  return data.updateTrainingCourse as TrainingCourseRow;
}

export async function archiveTrainingCourse(
  id: string,
  version?: number,
): Promise<TrainingCourseRow> {
  const data = await requestGraphQL<
    ArchiveTrainingCourseMutation,
    ArchiveTrainingCourseMutationVariables
  >({
    document: ArchiveTrainingCourseDocument,
    operationName: "ArchiveTrainingCourse",
    variables: { id, version },
  });
  return data.archiveTrainingCourse as TrainingCourseRow;
}

export async function restoreTrainingCourse(
  id: string,
  version?: number,
): Promise<TrainingCourseRow> {
  const data = await requestGraphQL<
    RestoreTrainingCourseMutation,
    RestoreTrainingCourseMutationVariables
  >({
    document: RestoreTrainingCourseDocument,
    operationName: "RestoreTrainingCourse",
    variables: { id, version },
  });
  return data.restoreTrainingCourse as TrainingCourseRow;
}

export async function assignWorkerTraining(
  input: AssignWorkerTrainingInput,
): Promise<WorkerTrainingRecordRow> {
  const data = await requestGraphQL<
    AssignWorkerTrainingMutation,
    AssignWorkerTrainingMutationVariables
  >({
    document: AssignWorkerTrainingDocument,
    operationName: "AssignWorkerTraining",
    variables: { input },
  });
  return data.assignWorkerTraining as WorkerTrainingRecordRow;
}

export type BulkAssignTrainingResult = BulkAssignTrainingMutation["bulkAssignTraining"];

/**
 * Opens one or more courses for a set of workers. Workers who already have a
 * course open come back as skipped rather than failed, so the caller can say
 * how many were actually enrolled.
 */
export async function bulkAssignTraining(
  input: BulkAssignTrainingInput,
): Promise<BulkAssignTrainingResult> {
  const data = await requestGraphQL<
    BulkAssignTrainingMutation,
    BulkAssignTrainingMutationVariables
  >({
    document: BulkAssignTrainingDocument,
    operationName: "BulkAssignTraining",
    variables: { input },
  });
  return data.bulkAssignTraining;
}

export async function assignRequiredWorkerTraining(
  workerId: string,
): Promise<WorkerTrainingRecordRow[]> {
  const data = await requestGraphQL<
    AssignRequiredWorkerTrainingMutation,
    AssignRequiredWorkerTrainingMutationVariables
  >({
    document: AssignRequiredWorkerTrainingDocument,
    operationName: "AssignRequiredWorkerTraining",
    variables: { workerId },
  });
  return data.assignRequiredWorkerTraining as WorkerTrainingRecordRow[];
}

export async function completeWorkerTraining(
  input: CompleteWorkerTrainingInput,
): Promise<WorkerTrainingRecordRow> {
  const data = await requestGraphQL<
    CompleteWorkerTrainingMutation,
    CompleteWorkerTrainingMutationVariables
  >({
    document: CompleteWorkerTrainingDocument,
    operationName: "CompleteWorkerTraining",
    variables: { input },
  });
  return data.completeWorkerTraining as WorkerTrainingRecordRow;
}

export async function waiveWorkerTraining(
  input: WaiveWorkerTrainingInput,
): Promise<WorkerTrainingRecordRow> {
  const data = await requestGraphQL<
    WaiveWorkerTrainingMutation,
    WaiveWorkerTrainingMutationVariables
  >({
    document: WaiveWorkerTrainingDocument,
    operationName: "WaiveWorkerTraining",
    variables: { input },
  });
  return data.waiveWorkerTraining as WorkerTrainingRecordRow;
}

export async function cancelWorkerTraining(
  input: CancelWorkerTrainingInput,
): Promise<WorkerTrainingRecordRow> {
  const data = await requestGraphQL<
    CancelWorkerTrainingMutation,
    CancelWorkerTrainingMutationVariables
  >({
    document: CancelWorkerTrainingDocument,
    operationName: "CancelWorkerTraining",
    variables: { input },
  });
  return data.cancelWorkerTraining as WorkerTrainingRecordRow;
}

export async function attachWorkerTrainingDocument(
  input: AttachWorkerTrainingDocumentInput,
): Promise<WorkerTrainingRecordRow> {
  const data = await requestGraphQL<
    AttachWorkerTrainingDocumentMutation,
    AttachWorkerTrainingDocumentMutationVariables
  >({
    document: AttachWorkerTrainingDocumentDocument,
    operationName: "AttachWorkerTrainingDocument",
    variables: { input },
  });
  return data.attachWorkerTrainingDocument as WorkerTrainingRecordRow;
}
