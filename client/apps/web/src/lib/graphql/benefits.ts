import {
  BenefitCostsDocument,
  BenefitPlansDocument,
  CreateBenefitPlanDocument,
  EndBenefitEnrollmentDocument,
  EnrollBenefitDocument,
  UpdateBenefitPlanDocument,
  WorkerBenefitEnrollmentsDocument,
  WorkerTotalCompensationDocument,
  type BenefitCostsQuery,
  type BenefitCostsQueryVariables,
  type BenefitPlanInput,
  type BenefitPlansQuery,
  type BenefitPlansQueryVariables,
  type CreateBenefitPlanMutation,
  type CreateBenefitPlanMutationVariables,
  type EndBenefitEnrollmentInput,
  type EndBenefitEnrollmentMutation,
  type EndBenefitEnrollmentMutationVariables,
  type EnrollBenefitInput,
  type EnrollBenefitMutation,
  type EnrollBenefitMutationVariables,
  type UpdateBenefitPlanInput,
  type UpdateBenefitPlanMutation,
  type UpdateBenefitPlanMutationVariables,
  type WorkerBenefitEnrollmentsQuery,
  type WorkerBenefitEnrollmentsQueryVariables,
  type WorkerTotalCompensationQuery,
  type WorkerTotalCompensationQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type BenefitPlanRow = BenefitPlansQuery["benefitPlans"][number];
export type BenefitEnrollmentRow =
  WorkerBenefitEnrollmentsQuery["workerBenefitEnrollments"][number];
export type BenefitCostRow = BenefitCostsQuery["benefitCosts"][number];
export type TotalCompensationSummary = WorkerTotalCompensationQuery["workerTotalCompensation"];

export const BENEFIT_PLANS_KEY = "benefit-plans";
export const BENEFIT_ENROLLMENTS_KEY = "benefit-enrollments";
export const BENEFIT_COSTS_KEY = "benefit-costs";
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

export async function fetchWorkerBenefitEnrollments(
  workerId: string,
  openOnly = false,
  options?: { signal?: AbortSignal },
): Promise<BenefitEnrollmentRow[]> {
  const data = await requestGraphQL<
    WorkerBenefitEnrollmentsQuery,
    WorkerBenefitEnrollmentsQueryVariables
  >({
    document: WorkerBenefitEnrollmentsDocument,
    operationName: "WorkerBenefitEnrollments",
    variables: { workerId, openOnly },
    signal: options?.signal,
  });
  return data.workerBenefitEnrollments;
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
