import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";

type BillingInput = Pick<FormulaTemplate, "status" | "currentVersionNumber" | "billing">;

/**
 * Which content prices shipments when it is not the content on screen: the
 * approved snapshot keeps billing while a revision is drafted or reviewed, and
 * an approval scheduled for later bills only from its start. Null when what
 * is shown is what bills, or nothing bills at all.
 */
export function billingLabel(template: BillingInput, t: TranslateFn): string | null {
  const billing = template.billing;
  const current = template.currentVersionNumber ?? 0;
  if (!billing || template.status === "Inactive" || billing.versionNumber >= current) {
    return null;
  }

  switch (template.status) {
    case "InReview":
      return t("Billing v{0} while v{1} is in review", billing.versionNumber, current);
    case "Draft":
      return t("Billing v{0} while v{1} is a draft", billing.versionNumber, current);
    default:
      return t("Billing v{0} until v{1} takes effect", billing.versionNumber, current);
  }
}
