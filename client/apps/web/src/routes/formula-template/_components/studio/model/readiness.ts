import type { BreakdownReconciliation } from "@/components/formula-editor/breakdown-reconcile";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ReadinessCheck } from "@trenova/shared/types/formula-template";

/** The keys the server's readiness uses, so a row in the studio is the row Submit enforces. */
export const READINESS_KEYS = {
  details: "details",
  expression: "expression",
  nullables: "nullableFields",
  scenarios: "scenarios",
  breakdown: "breakdown",
  saved: "saved",
} as const;

export type StudioReadinessInput = {
  mode: "create" | "edit";
  name: string;
  description: string;
  expressionEmpty: boolean;
  /** Problems the editor's linter found that stop the expression compiling. */
  expressionErrors: number;
  /** The preview's verdict on the expression: null before it has run. */
  previewValid: boolean | null;
  /** Optional shipment fields the expression reads unguarded, by name. */
  unguardedFields: readonly string[];
  scenarioTotal: number;
  scenarioPassed: number;
  breakdownLines: number;
  reconciliation: BreakdownReconciliation | null;
  dirty: boolean;
};

/**
 * The studio's live copy of `formulatemplateservice/readiness.go`, checked
 * against what is in the editor rather than what is saved. Name and
 * description, a valid expression and passing scenarios block; unguarded
 * fields and a breakdown that does not reconcile only warn. Create mode has
 * no scenarios row (they are saved with the template), and edit mode adds
 * "Changes saved" because review always covers what is saved.
 */
export function studioReadiness(input: StudioReadinessInput, t: TranslateFn): ReadinessCheck[] {
  const checks: ReadinessCheck[] = [];
  const name = input.name.trim();
  const description = input.description.trim();

  checks.push({
    key: READINESS_KEYS.details,
    label: t("Name and description"),
    status: name && description ? "pass" : "fail",
    detail: !name ? t("Name is missing") : !description ? t("Description is missing") : undefined,
  });

  const expressionFails =
    input.expressionEmpty || input.expressionErrors > 0 || input.previewValid === false;
  checks.push({
    key: READINESS_KEYS.expression,
    label: t("Expression is valid"),
    status: expressionFails ? "fail" : "pass",
    detail: input.expressionEmpty
      ? t("Empty")
      : input.expressionErrors > 0
        ? t("{0, plural, one {# error} other {# errors}}", input.expressionErrors)
        : input.previewValid === false
          ? t("Fails on the preview load")
          : undefined,
  });

  const unguarded = input.unguardedFields.length;
  checks.push({
    key: READINESS_KEYS.nullables,
    label: t("Safe on every shipment"),
    status: unguarded > 0 ? "warn" : "pass",
    detail:
      unguarded > 0
        ? t("{0, plural, one {# value can be empty} other {# values can be empty}}", unguarded)
        : undefined,
  });

  if (input.mode === "edit") {
    checks.push({
      key: READINESS_KEYS.scenarios,
      label: t("Scenarios pass"),
      status:
        input.scenarioTotal > 0 && input.scenarioPassed === input.scenarioTotal ? "pass" : "fail",
      detail:
        input.scenarioTotal === 0
          ? t("Add at least one")
          : t("{0} of {1}", input.scenarioPassed, input.scenarioTotal),
    });
  }

  if (input.breakdownLines > 0) {
    const reconciliation = input.reconciliation;
    const status =
      reconciliation === null || (reconciliation.balanced && !reconciliation.clampMismatch)
        ? "pass"
        : "warn";
    checks.push({
      key: READINESS_KEYS.breakdown,
      label: t("Breakdown reconciles"),
      status,
      detail: reconciliation?.clampMismatch
        ? t("Lines don't match the billed minimum")
        : reconciliation && !reconciliation.balanced
          ? t("Lines don't add up")
          : undefined,
    });
  }

  if (input.mode === "edit" && input.dirty) {
    checks.push({
      key: READINESS_KEYS.saved,
      label: t("Changes saved"),
      status: "fail",
      detail: t("Review covers what is saved"),
    });
  }

  return checks;
}

/** The goal the readiness meter counts toward, by where the template is. */
export function readinessGoal(
  mode: "create" | "edit",
  status: string | undefined,
  t: TranslateFn,
): string {
  if (mode === "create") return t("Ready to create");
  if (status === "InReview") return t("Ready to approve");
  return t("Ready for review");
}
