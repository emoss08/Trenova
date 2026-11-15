import { PlanFeature } from "@/lib/plan-feature";
import { planFeatureOf } from "@trenova/shared/hooks/use-plan-feature";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";

/**
 * Whether the plan includes what Nova does on the billing queue. The review, and the brief
 * written from it, are an agent's work (agent automation) that Nova presents (the Desk); a
 * search read as a sentence is Nova's alone. With neither on the plan nothing about Nova is
 * read or drawn on the queue or offered in its controls.
 */
export function billingQueueAssistOnPlan(
  manifest: Pick<PermissionManifest, "unavailableFeatures"> | null | undefined,
): { review: boolean; search: boolean } {
  const nova = planFeatureOf(manifest, PlanFeature.AIDesk).available;
  const agents = planFeatureOf(manifest, PlanFeature.AgentAutomation).available;
  return { review: nova && agents, search: nova };
}

/**
 * Whether the plan includes Nova's review of queue items. What an earlier review left on an
 * item (its verdict, its summary, what it read, the issues it found) is shown, and the review
 * offered in the billing controls, only while it does.
 */
export function useBillingQueueReviewOnPlan(): boolean {
  return usePermissionStore((state) => billingQueueAssistOnPlan(state.manifest).review);
}
