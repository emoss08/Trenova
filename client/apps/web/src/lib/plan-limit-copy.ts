import {
  formatPlanMeterValue,
  planDisplayName,
  planMeterDefinition,
  type PlanMeterWindow,
} from "@/lib/plan-meters";
import type { PlanLimitCopy } from "@trenova/edition";
import { planFeatureOf } from "@trenova/shared/hooks/use-plan-feature";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  canManagePlan,
  planLockedGuidance,
  planLockedSentence,
} from "@trenova/shared/lib/plan-copy";
import {
  ORGANIZATION_SUSPENDED_REASON,
  SUBSCRIPTION_READ_ONLY_REASON,
  type PlanLimitNotice,
} from "@trenova/shared/lib/plan-limit";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";

type Lines = Pick<PlanLimitCopy, "title" | "description" | "guidance">;

/**
 * What an add-on refusal is about, in names: the feature's from the permission manifest, the
 * add-on's from the refusal itself (the manifest's when the refusal names none), and whether
 * the person manages the organization and so adds it themselves. Never a key or a code.
 */
export type PlanRestrictionNames = {
  featureName: string;
  addOnName: string;
  canManagePlan: boolean;
};

/**
 * What a plan-limit refusal says beyond its figures. The host's wording names no plan
 * and promises nothing; an edition with plans of its own passes its own.
 */
export type PlanLimitWording = {
  planName: (planKey: string) => string;
  quotaGuidance: (window: PlanMeterWindow, t: TranslateFn) => string;
  readOnly: (t: TranslateFn) => Lines;
  restricted: (names: PlanRestrictionNames, t: TranslateFn) => Lines;
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
  restricted: ({ featureName, addOnName, canManagePlan: managesPlan }, t) => ({
    title: featureName ? t("{0} is not on your plan", featureName) : t("Not on your plan"),
    description:
      featureName || addOnName
        ? planLockedSentence(t, featureName, addOnName)
        : t("This part of Trenova is not included in your current plan."),
    guidance: planLockedGuidance(t, managesPlan),
  }),
};

const MANAGED_PLAN = "managed";

/**
 * What frees room on a managed instance, keyed by the meter the refusal names. Overage is
 * turned on and allowances are added in the Trenova account, never priced here.
 */
const MANAGED_QUOTA_GUIDANCE: Readonly<
  Record<string, (limit: number | null, t: TranslateFn) => string>
> = {
  "documents.storage_bytes": (_limit, t) =>
    t(
      "Turn on overage from your Trenova account to keep uploading, or delete documents you no longer need.",
    ),
  "document_intelligence.pages": (_limit, t) =>
    t("Turn on overage from your Trenova account to keep extracting, or wait for the next period."),
  "organizations.total": (limit, t) =>
    t(
      "{0, plural, one {Your plan includes # organization.} other {Your plan includes # organizations.}} Add organizations to your plan from your Trenova account to create another.",
      limit ?? 0,
    ),
};

function managedQuotaGuidance(notice: PlanLimitNotice, t: TranslateFn): string | null {
  if (notice.kind !== "quota" || notice.plan !== MANAGED_PLAN) {
    return null;
  }
  return MANAGED_QUOTA_GUIDANCE[notice.meter]?.(notice.limit, t) ?? null;
}

/** The names an add-on refusal is explained with, read from the permission manifest. */
export function planRestrictionNames(feature: string, addOn: string): PlanRestrictionNames {
  const { manifest, hasPermission } = usePermissionStore.getState();
  const plan = planFeatureOf(manifest, feature);
  return {
    featureName: plan.featureName,
    addOnName: addOn || plan.addOnName,
    canManagePlan: canManagePlan(hasPermission),
  };
}

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
      guidance: managedQuotaGuidance(notice, t) ?? wording.quotaGuidance(meter.window, t),
      usage,
    };
  }

  if (notice.reason === SUBSCRIPTION_READ_ONLY_REASON) {
    return { ...wording.readOnly(t), usage: null };
  }

  if (notice.reason === ORGANIZATION_SUSPENDED_REASON) {
    return {
      title: t("Your organization is suspended"),
      description: t(
        "This organization's Trenova account is suspended. Its owner can resume it from the Trenova console.",
      ),
      guidance: t("Everything you entered is kept while the account is suspended."),
      usage: null,
    };
  }

  return {
    ...wording.restricted(planRestrictionNames(notice.feature, notice.addOn), t),
    usage: null,
  };
}
