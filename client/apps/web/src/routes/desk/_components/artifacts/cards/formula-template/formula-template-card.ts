import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  FormulaTemplateStatus,
  TemplateUsageCount,
} from "@trenova/shared/types/formula-template";
import type { TariffOutlineItem } from "@trenova/shared/types/tariff";

/** Where a template stands, as the card colours it. */
export type FormulaStage = "active" | "review" | "draft" | "inactive";

export function formulaStage(status: FormulaTemplateStatus): FormulaStage {
  switch (status) {
    case "Active":
      return "active";
    case "InReview":
      return "review";
    case "Inactive":
      return "inactive";
    default:
      return "draft";
  }
}

export function formulaStatusLabel(status: FormulaTemplateStatus, t: TranslateFn): string {
  switch (status) {
    case "Active":
      return t("Active");
    case "InReview":
      return t("In review");
    case "Inactive":
      return t("Inactive");
    default:
      return t("Draft");
  }
}

/** Why the status matters to a biller, in the words the card puts beside it. */
export function formulaStatusNote(status: FormulaTemplateStatus, t: TranslateFn): string {
  switch (status) {
    case "Active":
      return t("Prices shipments");
    case "InReview":
      return t("Waiting on approval before it prices shipments");
    case "Inactive":
      return t("Retired; prices nothing new");
    default:
      return t("Not yet submitted for approval");
  }
}

/** What uses a template, counted the way a person says it. */
export function usageLabel(usage: TemplateUsageCount, t: TranslateFn): string {
  switch (usage.type) {
    case "shipment":
      return t("{0, plural, one {# shipment} other {# shipments}}", usage.count);
    case "rate_agreement_rule":
      return t("{0, plural, one {# agreement rule} other {# agreement rules}}", usage.count);
    case "rate_agreement":
      return t("{0, plural, one {# rate agreement} other {# rate agreements}}", usage.count);
    case "billing_control_fallback":
      return t("the billing fallback");
    default:
      return t("{0, plural, one {# other use} other {# other uses}}", usage.count);
  }
}

/** A total clause as the card lists it under the charges. */
export function totalClauseLabel(item: Pick<TariffOutlineItem, "name">, t: TranslateFn): string {
  switch (item.name) {
    case "minimum":
      return t("Minimum charge");
    case "maximum":
      return t("Maximum charge");
    case "round":
      return t("Rounding");
    default:
      return item.name;
  }
}
