import type {
  BenefitCostsQuery,
  BenefitCostsQueryVariables,
  BenefitEnrollmentsQuery,
  BenefitEnrollmentsQueryVariables,
  BenefitEnrollmentStatus,
  BenefitPlanInput,
  BenefitPlansQuery,
  BenefitPlansQueryVariables,
  CreateBenefitPlanMutation,
  CreateBenefitPlanMutationVariables,
  EndBenefitEnrollmentInput,
  EndBenefitEnrollmentMutation,
  EndBenefitEnrollmentMutationVariables,
  EnrollBenefitInput,
  EnrollBenefitMutation,
  EnrollBenefitMutationVariables,
  UpdateBenefitPlanInput,
  UpdateBenefitPlanMutation,
  UpdateBenefitPlanMutationVariables,
  WorkerBenefitEnrollmentsQuery,
  WorkerBenefitEnrollmentsQueryVariables,
  WorkerTotalCompensationQuery,
  WorkerTotalCompensationQueryVariables,
} from "@trenova/graphql/generated/graphql";
import {
  BenefitCostsDocument,
  BenefitEnrollmentsDocument,
  BenefitPlansDocument,
  CreateBenefitPlanDocument,
  EndBenefitEnrollmentDocument,
  EnrollBenefitDocument,
  UpdateBenefitPlanDocument,
  WorkerBenefitEnrollmentsDocument,
  WorkerTotalCompensationDocument,
} from "@trenova/graphql/generated/documents/benefits/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { Connection } from "./connection";

export type BenefitPlanRow = BenefitPlansQuery["benefitPlans"][number];
export type BenefitEnrollmentRow =
  WorkerBenefitEnrollmentsQuery["workerBenefitEnrollments"]["edges"][number]["node"];
export type BenefitCostRow = BenefitCostsQuery["benefitCosts"][number];
export type BenefitEnrollmentListRow = BenefitEnrollmentsQuery["benefitEnrollments"][number];
export type TotalCompensationSummary = WorkerTotalCompensationQuery["workerTotalCompensation"];

export const BENEFIT_PLANS_KEY = "benefit-plans";
export const BENEFIT_ENROLLMENTS_KEY = "benefit-enrollments";
export const BENEFIT_COSTS_KEY = "benefit-costs";
export const BENEFIT_ENROLLMENT_LIST_KEY = "benefit-enrollment-list";
export const TOTAL_COMPENSATION_KEY = "total-compensation";

export async function fetchBenefitPlans(
  args: { activeOnly?: boolean; planYear?: number } = {},
  options?: { signal?: AbortSignal },
): Promise<BenefitPlanRow[]> {
  const data = await requestGraphQL<BenefitPlansQuery, BenefitPlansQueryVariables>({
    document: BenefitPlansDocument,
    operationName: "BenefitPlans",
    variables: {
      activeOnly: args.activeOnly ?? null,
      planYear: args.planYear ?? null,
    },
    signal: options?.signal,
  });
  return data.benefitPlans;
}

/** One page of a worker's enrolments as the connection the server returns. */
export async function fetchWorkerBenefitEnrollments(
  args: { workerId: string; openOnly?: boolean; first?: number; after?: string | null },
  options?: { signal?: AbortSignal },
): Promise<Connection<BenefitEnrollmentRow>> {
  const data = await requestGraphQL<
    WorkerBenefitEnrollmentsQuery,
    WorkerBenefitEnrollmentsQueryVariables
  >({
    document: WorkerBenefitEnrollmentsDocument,
    operationName: "WorkerBenefitEnrollments",
    variables: {
      workerId: args.workerId,
      openOnly: args.openOnly ?? false,
      first: args.first,
      after: args.after,
    },
    signal: options?.signal,
  });
  return data.workerBenefitEnrollments;
}

export async function fetchBenefitEnrollments(
  args: {
    planId?: string | null;
    statuses?: BenefitEnrollmentStatus[];
    openOnly?: boolean;
    limit?: number;
  } = {},
  options?: { signal?: AbortSignal },
): Promise<BenefitEnrollmentListRow[]> {
  const data = await requestGraphQL<BenefitEnrollmentsQuery, BenefitEnrollmentsQueryVariables>({
    document: BenefitEnrollmentsDocument,
    operationName: "BenefitEnrollments",
    variables: {
      planId: args.planId ?? null,
      statuses: args.statuses ?? null,
      openOnly: args.openOnly ?? null,
      limit: args.limit ?? null,
    },
    signal: options?.signal,
  });
  return data.benefitEnrollments;
}

export async function fetchBenefitCosts(
  planYear?: number,
  options?: { signal?: AbortSignal },
): Promise<BenefitCostRow[]> {
  const data = await requestGraphQL<BenefitCostsQuery, BenefitCostsQueryVariables>({
    document: BenefitCostsDocument,
    operationName: "BenefitCosts",
    variables: { planYear: planYear ?? null },
    signal: options?.signal,
  });
  return data.benefitCosts;
}

export async function fetchWorkerTotalCompensation(
  workerId: string,
  planYear?: number,
  options?: { signal?: AbortSignal },
): Promise<TotalCompensationSummary> {
  const data = await requestGraphQL<
    WorkerTotalCompensationQuery,
    WorkerTotalCompensationQueryVariables
  >({
    document: WorkerTotalCompensationDocument,
    operationName: "WorkerTotalCompensation",
    variables: { workerId, planYear: planYear ?? null },
    signal: options?.signal,
  });
  return data.workerTotalCompensation;
}

export async function createBenefitPlan(input: BenefitPlanInput) {
  const data = await requestGraphQL<CreateBenefitPlanMutation, CreateBenefitPlanMutationVariables>({
    document: CreateBenefitPlanDocument,
    operationName: "CreateBenefitPlan",
    variables: { input },
  });
  return data.createBenefitPlan;
}

export async function updateBenefitPlan(input: UpdateBenefitPlanInput) {
  const data = await requestGraphQL<UpdateBenefitPlanMutation, UpdateBenefitPlanMutationVariables>({
    document: UpdateBenefitPlanDocument,
    operationName: "UpdateBenefitPlan",
    variables: { input },
  });
  return data.updateBenefitPlan;
}

export async function enrollBenefit(input: EnrollBenefitInput) {
  const data = await requestGraphQL<EnrollBenefitMutation, EnrollBenefitMutationVariables>({
    document: EnrollBenefitDocument,
    operationName: "EnrollBenefit",
    variables: { input },
  });
  return data.enrollBenefit;
}

export async function endBenefitEnrollment(input: EndBenefitEnrollmentInput) {
  const data = await requestGraphQL<
    EndBenefitEnrollmentMutation,
    EndBenefitEnrollmentMutationVariables
  >({
    document: EndBenefitEnrollmentDocument,
    operationName: "EndBenefitEnrollment",
    variables: { input },
  });
  return data.endBenefitEnrollment;
}
