import { planLimitCopy, type PlanLimitWording } from "@/lib/plan-limit-copy";
import type { PlanLimitCopy } from "@trenova/edition";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PlanLimitNotice } from "@trenova/shared/lib/plan-limit";
import { planCapabilityDefinition, planDisplayName } from "./free-demo";

const PAID_PLANS_COMING = "Paid plans with higher limits are coming soon.";

/**
 * The free demo's wording: a quota says what frees room and that paid plans are coming,
 * a restriction says what the demo leaves out, and the read-only refusal explains that
 * the trial is over.
 */
export const FREE_DEMO_WORDING: PlanLimitWording = {
  planName: (planKey) => planDisplayName(planKey),
  quotaGuidance: (window, t) => {
    if (window === "month") {
      return t("This limit resets at the start of next month. {0}", t(PAID_PLANS_COMING));
    }
    if (window === "item") {
      return t("Upload a smaller file, or split it into parts. {0}", t(PAID_PLANS_COMING));
    }
    return t(
      "The free demo is for trying Trenova, so it holds a small amount of data. Delete records you no longer need to free room. {0}",
      t(PAID_PLANS_COMING),
    );
  },
  readOnly: (t) => ({
    title: t("Your free demo has ended"),
    description: t(
      "The trial period is over, so this workspace is read-only: you can open and export everything, but nothing can be created or changed.",
    ),
    guidance: t(
      "The workspace is deleted when the read-only period ends. Paid plans are coming soon; until then, export anything you want to keep.",
    ),
  }),
  restricted: (capability, planName, t) => {
    const definition = planCapabilityDefinition(capability);
    return {
      title: t("{0} not available", t(definition.label)),
      description: t(definition.explanation),
      guidance: t(
        "{0} is for trying Trenova and leaves this out. Paid plans that include it are coming soon.",
        t(planName),
      ),
    };
  },
};

export function cloudPlanLimitCopy(notice: PlanLimitNotice, t: TranslateFn): PlanLimitCopy {
  return planLimitCopy(notice, t, FREE_DEMO_WORDING);
}
