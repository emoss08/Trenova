import type {
  CreateWorkerPolicyMutation,
  CreateWorkerPolicyMutationVariables,
  DecideProfileChangeInput,
  DecideProfileChangeMutation,
  DecideProfileChangeMutationVariables,
  ProfileChangeFilterInput,
  ProfileChangeRequestsQuery,
  ProfileChangeRequestsQueryVariables,
  UpdateWorkerPolicyMutation,
  UpdateWorkerPolicyMutationVariables,
  WorkerPoliciesQuery,
  WorkerPoliciesQueryVariables,
  WorkerPolicyAcknowledgementsQuery,
  WorkerPolicyAcknowledgementsQueryVariables,
  WorkerPolicyComplianceQuery,
  WorkerPolicyComplianceQueryVariables,
  WorkerPolicyInput,
} from "@trenova/graphql/generated/graphql";
import {
  CreateWorkerPolicyDocument,
  DecideProfileChangeDocument,
  ProfileChangeRequestsDocument,
  UpdateWorkerPolicyDocument,
  WorkerPoliciesDocument,
  WorkerPolicyAcknowledgementsDocument,
  WorkerPolicyComplianceDocument,
} from "@trenova/graphql/generated/documents/self-service/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Connection } from "./connection";

export type WorkerPolicyRow = WorkerPoliciesQuery["workerPolicies"][number];
export type PolicyComplianceView = WorkerPolicyComplianceQuery["workerPolicyCompliance"];
export type PolicyAcknowledgementRow =
  WorkerPolicyAcknowledgementsQuery["workerPolicyAcknowledgements"]["edges"][number]["node"];
export type ProfileChangeRequestRow = ProfileChangeRequestsQuery["profileChangeRequests"][number];

export const WORKER_POLICIES_KEY = "worker-policies";
export const POLICY_COMPLIANCE_KEY = "worker-policy-compliance";
export const POLICY_ACKNOWLEDGEMENTS_KEY = "worker-policy-acknowledgements";
export const PROFILE_CHANGE_REQUESTS_KEY = "profile-change-requests";

export async function fetchWorkerPolicies(
  args: { activeOnly?: boolean; limit?: number } = {},
  options?: { signal?: AbortSignal },
): Promise<WorkerPolicyRow[]> {
  const data = await requestGraphQL<WorkerPoliciesQuery, WorkerPoliciesQueryVariables>({
    document: WorkerPoliciesDocument,
    operationName: "WorkerPolicies",
    variables: { activeOnly: args.activeOnly ?? null, limit: args.limit ?? null },
    signal: options?.signal,
  });
  return data.workerPolicies;
}

export async function fetchWorkerPolicyCompliance(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<PolicyComplianceView> {
  const data = await requestGraphQL<
    WorkerPolicyComplianceQuery,
    WorkerPolicyComplianceQueryVariables
  >({
    document: WorkerPolicyComplianceDocument,
    operationName: "WorkerPolicyCompliance",
    variables: { id },
    signal: options?.signal,
  });
  return data.workerPolicyCompliance;
}

/** One page of the policies a worker signed, newest first, as the connection the server returns. */
export async function fetchWorkerPolicyAcknowledgements(
  args: { workerId: string; first?: number; after?: string | null },
  options?: { signal?: AbortSignal },
): Promise<Connection<PolicyAcknowledgementRow>> {
  const data = await requestGraphQL<
    WorkerPolicyAcknowledgementsQuery,
    WorkerPolicyAcknowledgementsQueryVariables
  >({
    document: WorkerPolicyAcknowledgementsDocument,
    operationName: "WorkerPolicyAcknowledgements",
    variables: {
      workerId: args.workerId,
      first: args.first,
      after: args.after,
      includeTotalCount: !args.after,
    },
    signal: options?.signal,
  });
  return data.workerPolicyAcknowledgements;
}

export async function fetchProfileChangeRequests(
  filter: ProfileChangeFilterInput = {},
  options?: { signal?: AbortSignal },
): Promise<ProfileChangeRequestRow[]> {
  const data = await requestGraphQL<
    ProfileChangeRequestsQuery,
    ProfileChangeRequestsQueryVariables
  >({
    document: ProfileChangeRequestsDocument,
    operationName: "ProfileChangeRequests",
    variables: { filter },
    signal: options?.signal,
  });
  return data.profileChangeRequests;
}

export async function createWorkerPolicy(input: WorkerPolicyInput) {
  const data = await requestGraphQL<
    CreateWorkerPolicyMutation,
    CreateWorkerPolicyMutationVariables
  >({
    document: CreateWorkerPolicyDocument,
    operationName: "CreateWorkerPolicy",
    variables: { input },
  });
  return data.createWorkerPolicy;
}

export async function updateWorkerPolicy(id: string, input: WorkerPolicyInput) {
  const data = await requestGraphQL<
    UpdateWorkerPolicyMutation,
    UpdateWorkerPolicyMutationVariables
  >({
    document: UpdateWorkerPolicyDocument,
    operationName: "UpdateWorkerPolicy",
    variables: { id, input },
  });
  return data.updateWorkerPolicy;
}

export async function decideProfileChange(input: DecideProfileChangeInput) {
  const data = await requestGraphQL<
    DecideProfileChangeMutation,
    DecideProfileChangeMutationVariables
  >({
    document: DecideProfileChangeDocument,
    operationName: "DecideProfileChange",
    variables: { input },
  });
  return data.decideProfileChange;
}
