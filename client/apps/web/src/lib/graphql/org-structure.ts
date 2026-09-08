import {
  ApprovalDelegationsDocument,
  AssignUserPositionDocument,
  AssignWorkerPositionDocument,
  CreateJobPositionDocument,
  DelegateApprovalDocument,
  HeadcountDocument,
  JobPositionHoldersDocument,
  JobPositionsDocument,
  MyTeamDocument,
  RevokeApprovalDelegationDocument,
  UpdateJobPositionDocument,
  type ApprovalDelegationsQuery,
  type ApprovalDelegationsQueryVariables,
  type AssignUserPositionMutation,
  type AssignUserPositionMutationVariables,
  type AssignWorkerPositionMutation,
  type AssignWorkerPositionMutationVariables,
  type CreateJobPositionMutation,
  type CreateJobPositionMutationVariables,
  type DelegateApprovalInput,
  type DelegateApprovalMutation,
  type DelegateApprovalMutationVariables,
  type HeadcountQuery,
  type HeadcountQueryVariables,
  type JobPositionHoldersQuery,
  type JobPositionHoldersQueryVariables,
  type JobPositionInput,
  type JobPositionsQuery,
  type JobPositionsQueryVariables,
  type MyTeamQuery,
  type MyTeamQueryVariables,
  type RevokeApprovalDelegationMutation,
  type RevokeApprovalDelegationMutationVariables,
  type UpdateJobPositionInput,
  type UpdateJobPositionMutation,
  type UpdateJobPositionMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type JobPositionRow = JobPositionsQuery["jobPositions"][number];
export type HeadcountSummary = HeadcountQuery["headcount"];
export type HeadcountRow = HeadcountSummary["byFleet"][number];
export type TeamMemberRow = MyTeamQuery["myTeam"][number];
export type ApprovalDelegationRow = ApprovalDelegationsQuery["approvalDelegations"][number];
export type PositionHolderRow = JobPositionHoldersQuery["jobPositionHolders"][number];

export const JOB_POSITIONS_KEY = "job-positions";
export const HEADCOUNT_KEY = "headcount";
export const MY_TEAM_KEY = "my-team";
export const APPROVAL_DELEGATIONS_KEY = "approval-delegations";
export const POSITION_HOLDERS_KEY = "position-holders";

export async function fetchJobPositions(
  args: {
    activeOnly?: boolean;
    drivingOnly?: boolean;
  } = {},
  options?: { signal?: AbortSignal },
): Promise<JobPositionRow[]> {
  const data = await requestGraphQL<JobPositionsQuery, JobPositionsQueryVariables>({
    document: JobPositionsDocument,
    operationName: "JobPositions",
    variables: {
      activeOnly: args.activeOnly ?? null,
      drivingOnly: args.drivingOnly ?? null,
    },
    signal: options?.signal,
  });
  return data.jobPositions;
}

export async function fetchHeadcount(options?: {
  signal?: AbortSignal;
}): Promise<HeadcountSummary> {
  const data = await requestGraphQL<HeadcountQuery, HeadcountQueryVariables>({
    document: HeadcountDocument,
    operationName: "Headcount",
    variables: {},
    signal: options?.signal,
  });
  return data.headcount;
}

export async function fetchMyTeam(
  includeInactive = false,
  options?: { signal?: AbortSignal },
): Promise<TeamMemberRow[]> {
  const data = await requestGraphQL<MyTeamQuery, MyTeamQueryVariables>({
    document: MyTeamDocument,
    operationName: "MyTeam",
    variables: { includeInactive },
    signal: options?.signal,
  });
  return data.myTeam;
}

export async function fetchApprovalDelegations(
  args: {
    delegatorId?: string;
    delegateId?: string;
    activeOnly?: boolean;
  } = {},
  options?: { signal?: AbortSignal },
): Promise<ApprovalDelegationRow[]> {
  const data = await requestGraphQL<ApprovalDelegationsQuery, ApprovalDelegationsQueryVariables>({
    document: ApprovalDelegationsDocument,
    operationName: "ApprovalDelegations",
    variables: {
      delegatorId: args.delegatorId ?? null,
      delegateId: args.delegateId ?? null,
      activeOnly: args.activeOnly ?? null,
    },
    signal: options?.signal,
  });
  return data.approvalDelegations;
}

export async function fetchPositionHolders(
  positionId: string,
  options?: { signal?: AbortSignal },
): Promise<PositionHolderRow[]> {
  const data = await requestGraphQL<JobPositionHoldersQuery, JobPositionHoldersQueryVariables>({
    document: JobPositionHoldersDocument,
    operationName: "JobPositionHolders",
    variables: { id: positionId },
    signal: options?.signal,
  });
  return data.jobPositionHolders;
}

export async function assignWorkerPosition(workerId: string, positionId: string | null) {
  const data = await requestGraphQL<
    AssignWorkerPositionMutation,
    AssignWorkerPositionMutationVariables
  >({
    document: AssignWorkerPositionDocument,
    operationName: "AssignWorkerPosition",
    variables: { workerId, positionId },
  });
  return data.assignWorkerPosition;
}

export async function assignUserPosition(userId: string, positionId: string | null) {
  const data = await requestGraphQL<
    AssignUserPositionMutation,
    AssignUserPositionMutationVariables
  >({
    document: AssignUserPositionDocument,
    operationName: "AssignUserPosition",
    variables: { userId, positionId },
  });
  return data.assignUserPosition;
}

export async function createJobPosition(input: JobPositionInput) {
  const data = await requestGraphQL<CreateJobPositionMutation, CreateJobPositionMutationVariables>({
    document: CreateJobPositionDocument,
    operationName: "CreateJobPosition",
    variables: { input },
  });
  return data.createJobPosition;
}

export async function updateJobPosition(input: UpdateJobPositionInput) {
  const data = await requestGraphQL<UpdateJobPositionMutation, UpdateJobPositionMutationVariables>({
    document: UpdateJobPositionDocument,
    operationName: "UpdateJobPosition",
    variables: { input },
  });
  return data.updateJobPosition;
}

export async function delegateApproval(input: DelegateApprovalInput) {
  const data = await requestGraphQL<DelegateApprovalMutation, DelegateApprovalMutationVariables>({
    document: DelegateApprovalDocument,
    operationName: "DelegateApproval",
    variables: { input },
  });
  return data.delegateApproval;
}

export async function revokeApprovalDelegation(id: string) {
  const data = await requestGraphQL<
    RevokeApprovalDelegationMutation,
    RevokeApprovalDelegationMutationVariables
  >({
    document: RevokeApprovalDelegationDocument,
    operationName: "RevokeApprovalDelegation",
    variables: { id },
  });
  return data.revokeApprovalDelegation;
}
