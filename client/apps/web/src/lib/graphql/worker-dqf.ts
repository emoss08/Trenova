import {
  DeleteEmploymentVerificationDocument,
  DqfRetentionCandidatesDocument,
  DriverQualificationFileDocument,
  MarkEmploymentVerificationRequestedDocument,
  OutstandingEmploymentVerificationsDocument,
  RecordEmploymentVerificationDocument,
  RecordEmploymentVerificationFollowUpDocument,
  UpdateEmploymentVerificationDocument,
  type DeleteEmploymentVerificationMutation,
  type DeleteEmploymentVerificationMutationVariables,
  type DqfRetentionCandidatesQuery,
  type DqfRetentionCandidatesQueryVariables,
  type DriverQualificationFileQuery,
  type DriverQualificationFileQueryVariables,
  type MarkEmploymentVerificationRequestedMutation,
  type MarkEmploymentVerificationRequestedMutationVariables,
  type OutstandingEmploymentVerificationsQuery,
  type OutstandingEmploymentVerificationsQueryVariables,
  type RecordEmploymentVerificationFollowUpMutation,
  type RecordEmploymentVerificationFollowUpMutationVariables,
  type RecordEmploymentVerificationInput,
  type RecordEmploymentVerificationMutation,
  type RecordEmploymentVerificationMutationVariables,
  type UpdateEmploymentVerificationInput,
  type UpdateEmploymentVerificationMutation,
  type UpdateEmploymentVerificationMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DQFFile = DriverQualificationFileQuery["driverQualificationFile"];
export type DQFItem = DQFFile["items"][number];
export type EmploymentVerificationRow = DQFFile["verifications"][number];
export type OutstandingVerificationRow =
  OutstandingEmploymentVerificationsQuery["outstandingEmploymentVerifications"][number];
export type DQFRetentionCandidate = DqfRetentionCandidatesQuery["dqfRetentionCandidates"][number];

export const DRIVER_QUALIFICATION_FILE_KEY = "driver-qualification-file";
export const OUTSTANDING_VERIFICATIONS_KEY = "outstanding-employment-verifications";
export const DQF_RETENTION_CANDIDATES_KEY = "dqf-retention-candidates";

export async function fetchDriverQualificationFile(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<DQFFile> {
  const data = await requestGraphQL<
    DriverQualificationFileQuery,
    DriverQualificationFileQueryVariables
  >({
    document: DriverQualificationFileDocument,
    operationName: "DriverQualificationFile",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.driverQualificationFile;
}

export async function fetchOutstandingVerifications(options?: {
  signal?: AbortSignal;
}): Promise<OutstandingVerificationRow[]> {
  const data = await requestGraphQL<
    OutstandingEmploymentVerificationsQuery,
    OutstandingEmploymentVerificationsQueryVariables
  >({
    document: OutstandingEmploymentVerificationsDocument,
    operationName: "OutstandingEmploymentVerifications",
    variables: {},
    signal: options?.signal,
  });
  return data.outstandingEmploymentVerifications;
}

export async function fetchDqfRetentionCandidates(options?: {
  signal?: AbortSignal;
}): Promise<DQFRetentionCandidate[]> {
  const data = await requestGraphQL<
    DqfRetentionCandidatesQuery,
    DqfRetentionCandidatesQueryVariables
  >({
    document: DqfRetentionCandidatesDocument,
    operationName: "DqfRetentionCandidates",
    variables: {},
    signal: options?.signal,
  });
  return data.dqfRetentionCandidates;
}

export async function recordEmploymentVerification(input: RecordEmploymentVerificationInput) {
  const data = await requestGraphQL<
    RecordEmploymentVerificationMutation,
    RecordEmploymentVerificationMutationVariables
  >({
    document: RecordEmploymentVerificationDocument,
    operationName: "RecordEmploymentVerification",
    variables: { input },
  });
  return data.recordEmploymentVerification;
}

export async function updateEmploymentVerification(input: UpdateEmploymentVerificationInput) {
  const data = await requestGraphQL<
    UpdateEmploymentVerificationMutation,
    UpdateEmploymentVerificationMutationVariables
  >({
    document: UpdateEmploymentVerificationDocument,
    operationName: "UpdateEmploymentVerification",
    variables: { input },
  });
  return data.updateEmploymentVerification;
}

export async function markEmploymentVerificationRequested(id: string) {
  const data = await requestGraphQL<
    MarkEmploymentVerificationRequestedMutation,
    MarkEmploymentVerificationRequestedMutationVariables
  >({
    document: MarkEmploymentVerificationRequestedDocument,
    operationName: "MarkEmploymentVerificationRequested",
    variables: { id },
  });
  return data.markEmploymentVerificationRequested;
}

export async function recordEmploymentVerificationFollowUp(id: string) {
  const data = await requestGraphQL<
    RecordEmploymentVerificationFollowUpMutation,
    RecordEmploymentVerificationFollowUpMutationVariables
  >({
    document: RecordEmploymentVerificationFollowUpDocument,
    operationName: "RecordEmploymentVerificationFollowUp",
    variables: { id },
  });
  return data.recordEmploymentVerificationFollowUp;
}

export async function deleteEmploymentVerification(id: string): Promise<boolean> {
  const data = await requestGraphQL<
    DeleteEmploymentVerificationMutation,
    DeleteEmploymentVerificationMutationVariables
  >({
    document: DeleteEmploymentVerificationDocument,
    operationName: "DeleteEmploymentVerification",
    variables: { id },
  });
  return data.deleteEmploymentVerification;
}
