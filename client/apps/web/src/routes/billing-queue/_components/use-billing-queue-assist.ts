import { billingQueueAssistOnPlan } from "@/lib/billing-queue-nova-plan";
import { billingQueue } from "@/lib/queries/billing-queue";
import { useQuery } from "@tanstack/react-query";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { BillingQueueAssist } from "@trenova/shared/types/billing-queue";

/** What Nova does on the queue for a plan without Trenova AI: nothing. */
export const NO_BILLING_QUEUE_ASSIST: BillingQueueAssist = Object.freeze({
  review: false,
  brief: false,
  search: false,
});

/**
 * What Nova does on the billing queue: the server's answer (the billing controls and an AI
 * provider serving the task) narrowed to what the organization's plan includes. Without
 * Trenova AI on the plan the server is never asked, and everything reads as off. Undefined
 * while the server's answer loads.
 */
export function useBillingQueueAssist(): BillingQueueAssist | undefined {
  const onPlan = usePermissionStore((state) => {
    const plan = billingQueueAssistOnPlan(state.manifest);
    return plan.review ? "review" : plan.search ? "search" : "none";
  });
  const { data } = useQuery({ ...billingQueue.assist(), enabled: onPlan !== "none" });

  if (onPlan === "none") return NO_BILLING_QUEUE_ASSIST;
  if (!data) return undefined;
  if (onPlan === "review") return data;
  return { review: false, brief: false, search: data.search };
}
