import type { ManagedPlanQuery } from "@trenova/graphql/generated/graphql";
import {
  ManagedPlanDocument,
} from "@trenova/graphql/generated/documents/managed-plan/managed-plan";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type ManagedPlan = ManagedPlanQuery["managedPlan"];
export type ManagedPlanAddOn = ManagedPlan["included"][number];
export type ManagedPlanChange = ManagedPlan["changes"][number];

/** What this instance's plan includes, the add-ons it does not, and its recent changes. */
export async function fetchManagedPlan(options?: { signal?: AbortSignal }): Promise<ManagedPlan> {
  const data = await requestGraphQL({
    document: ManagedPlanDocument,
    operationName: "ManagedPlan",
    signal: options?.signal,
  });
  return data.managedPlan;
}
