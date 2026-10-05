import {
  formatPlanMeterValue,
  planCapabilityLabel,
  planDisplayName,
  planMeterDefinition,
  type PlanMeterWindow,
} from "@/lib/plan-meters";
import type { PlanLimitCopy } from "@trenova/edition";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  SUBSCRIPTION_READ_ONLY_REASON,
  type PlanLimitNotice,
} from "@trenova/shared/lib/plan-limit";

type Lines = Pick<PlanLimitCopy, "title" | "description" | "guidance">;

/**
 * What a plan-limit refusal says beyond its figures. The host's wording names no plan
 * and promises nothing; an edition with plans of its own passes its own.
 */
export type PlanLimitWording = {
  planName: (planKey: string) => string;
  quotaGuidance: (window: PlanMeterWindow, t: TranslateFn) => string;
  readOnly: (t: TranslateFn) => Lines;
  restricted: (capability: string, planName: string, t: TranslateFn) => Lines;
};

export const GENERIC_PLAN_LIMIT_WORDING: PlanLimitWording = {
  planName: (planKey) => planDisplayName(planKey),
  quotaGuidance: (window, t) => {
    if (window === "month") return t("This limit resets at the start of next month.");
    if (window === "item") return t("Upload a smaller file, or split it into parts.");
    return t("Delete records you no longer need to free room.");
  },
  readOnly: (t) => ({
    title: t("This workspace is read-only"),
    description: t(
      "Your organization's subscription does not allow changes right now: you can open and export everything, but nothing can be created or changed.",
    ),
    guidance: t("Ask an administrator of your organization to restore the subscription."),
  }),
  restricted: (capability, _planName, t) => ({
    title: t("{0} not available", t(planCapabilityLabel(capability))),
    description: t("This part of Trenova is not included in your current plan."),
    guidance: t("Ask an administrator of your organization about changing the plan."),
  }),
};

/**
 * The words the plan-limit dialog shows for one refusal. A quota names the meter, how
 * much of it is used and what frees room; a restriction names what the plan leaves out;
 * the read-only refusal explains why nothing can be saved rather than blaming the action.
 */
export function planLimitCopy(
  notice: PlanLimitNotice,
  t: TranslateFn,
  wording: PlanLimitWording = GENERIC_PLAN_LIMIT_WORDING,
): PlanLimitCopy {
  const planName = wording.planName(notice.plan);

  if (notice.kind === "quota") {
    const meter = planMeterDefinition(notice.meter);
    const label = t(meter.label);
    const hasFigures = notice.limit !== null;
    const usage =
      meter.window !== "item" && notice.limit !== null && notice.limit > 0
        ? {
            used: Math.min(notice.used ?? notice.limit, notice.limit),
            limit: notice.limit,
            usedLabel: formatPlanMeterValue(notice.meter, notice.used ?? notice.limit),
            limitLabel: formatPlanMeterValue(notice.meter, notice.limit),
          }
        : null;

    const description =
      meter.window === "item" && hasFigures
        ? t(
            "{0} allows files up to {1}. This one is larger.",
            t(planName),
            formatPlanMeterValue(notice.meter, notice.limit ?? 0),
          )
        : usage
          ? t("{0}: {1} of {2} used ({3}).", label, usage.usedLabel, usage.limitLabel, t(planName))
          : t("Your organization has reached its {0} limit ({1}).", label, t(planName));

    return {
      title: t("{0} limit reached", label),
      description,
      guidance: wording.quotaGuidance(meter.window, t),
      usage,
    };
  }

  if (notice.reason === SUBSCRIPTION_READ_ONLY_REASON) {
    return { ...wording.readOnly(t), usage: null };
  }

  return { ...wording.restricted(notice.capability, planName, t), usage: null };
}
