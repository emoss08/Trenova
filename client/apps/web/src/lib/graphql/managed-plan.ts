import type { ManagedPlanQuery } from "@trenova/graphql/generated/graphql";
import {
  ManagedPlanDocument,
} from "@trenova/graphql/generated/documents/managed-plan/managed-plan";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type ManagedPlan = ManagedPlanQuery["managedPlan"];
export type ManagedPlanAddOn = ManagedPlan["included"][number];
export type ManagedPlanChange = ManagedPlan["changes"][number];
export type ManagedPlanUsage = NonNullable<ManagedPlan["usage"]>;
export type ManagedPlanMeter = ManagedPlanUsage["meters"][number];

/**
 * What this instance's plan includes, the add-ons it does not, its recent changes, its usage this
 * billing period and the link to its page in the customer's Trenova account.
 */
export async function fetchManagedPlan(options?: { signal?: AbortSignal }): Promise<ManagedPlan> {
  const data = await requestGraphQL({
    document: ManagedPlanDocument,
    operationName: "ManagedPlan",
    signal: options?.signal,
  });
  return data.managedPlan;
}
