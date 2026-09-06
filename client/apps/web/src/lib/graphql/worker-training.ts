import {
  ActiveTrainingCoursesDocument,
  ArchiveTrainingCourseDocument,
  AssignRequiredWorkerTrainingDocument,
  AssignWorkerTrainingDocument,
  BulkAssignTrainingDocument,
  AttachWorkerTrainingDocumentDocument,
  CancelWorkerTrainingDocument,
  CompleteWorkerTrainingDocument,
  CreateTrainingCourseDocument,
  RestoreTrainingCourseDocument,
  TrainingCourseTableDocument,
  UpdateTrainingCourseDocument,
  WaiveWorkerTrainingDocument,
  WorkerTrainingRecordsDocument,
  WorkerTrainingSummaryDocument,
  type ActiveTrainingCoursesQuery,
  type ActiveTrainingCoursesQueryVariables,
  type ArchiveTrainingCourseMutation,
  type ArchiveTrainingCourseMutationVariables,
  type AssignRequiredWorkerTrainingMutation,
  type AssignRequiredWorkerTrainingMutationVariables,
  type AssignWorkerTrainingInput,
  type AssignWorkerTrainingMutation,
  type BulkAssignTrainingInput,
  type BulkAssignTrainingMutation,
  type BulkAssignTrainingMutationVariables,
  type AssignWorkerTrainingMutationVariables,
  type AttachWorkerTrainingDocumentInput,
  type AttachWorkerTrainingDocumentMutation,
  type AttachWorkerTrainingDocumentMutationVariables,
  type CancelWorkerTrainingInput,
  type CancelWorkerTrainingMutation,
  type CancelWorkerTrainingMutationVariables,
  type CompleteWorkerTrainingInput,
  type CompleteWorkerTrainingMutation,
  type CompleteWorkerTrainingMutationVariables,
  type CreateTrainingCourseMutation,
  type CreateTrainingCourseMutationVariables,
  type RestoreTrainingCourseMutation,
  type RestoreTrainingCourseMutationVariables,
  type TrainingCourseFieldsFragment,
  type TrainingCourseInput,
  type UpdateTrainingCourseMutation,
  type UpdateTrainingCourseMutationVariables,
  type WaiveWorkerTrainingInput,
  type WaiveWorkerTrainingMutation,
  type WaiveWorkerTrainingMutationVariables,
  type WorkerTrainingRecordFieldsFragment,
  type WorkerTrainingRecordsQuery,
  type WorkerTrainingRecordsQueryVariables,
  type WorkerTrainingSummaryQuery,
  type WorkerTrainingSummaryQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type TrainingCourseRow = TrainingCourseFieldsFragment;
export type WorkerTrainingRecordRow = WorkerTrainingRecordFieldsFragment;
export type WorkerTrainingSummary = WorkerTrainingSummaryQuery["workerTrainingSummary"];
export type WorkerTrainingSummaryItem = WorkerTrainingSummary["items"][number];

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

export async function fetchWorkerTrainingRecords(
  workerId: string,
  includeClosed = true,
  options?: { signal?: AbortSignal },
): Promise<WorkerTrainingRecordRow[]> {
  const data = await requestGraphQL<
    WorkerTrainingRecordsQuery,
    WorkerTrainingRecordsQueryVariables
  >({
    document: WorkerTrainingRecordsDocument,
    operationName: "WorkerTrainingRecords",
    variables: { workerId, includeClosed },
    signal: options?.signal,
  });
  return data.workerTrainingRecords as WorkerTrainingRecordRow[];
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
  return data.workerTrainingSummary;
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
