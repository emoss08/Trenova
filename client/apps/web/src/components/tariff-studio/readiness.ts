import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ReadinessCheck } from "@trenova/shared/types/formula-template";
import type { TariffTestsState } from "./use-tariff-tests";

/** The keys the server's readiness uses, so a row in the Studio is the row Submit enforces. */
export const READINESS_KEYS = {
  details: "details",
  source: "source",
  warnings: "warnings",
  tests: "tests",
  saved: "saved",
} as const;

export type TariffReadinessInput = {
  mode: "create" | "edit";
  name: string;
  description: string;
  sourceEmpty: boolean;
  errors: number;
  warnings: number;
  tests: TariffTestsState;
  dirty: boolean;
  /** A file with no tests is refused, not warned about: true for a library. */
  testsRequired?: boolean;
};

/**
 * The Studio's live copy of the server's readiness, checked against what is in the editor
 * rather than what is saved: a name and description, a source the checker accepts and every
 * test passing block; warnings only warn, and so does a tariff with no tests (a library must
 * have one). Edit mode adds "Changes saved", because review always covers what is saved.
 */
export function tariffReadiness(input: TariffReadinessInput, t: TranslateFn): ReadinessCheck[] {
  const checks: ReadinessCheck[] = [];
  const name = input.name.trim();
  const description = input.description.trim();

  checks.push({
    key: READINESS_KEYS.details,
    label: t("Name and description"),
    status: name && description ? "pass" : "fail",
    detail: !name ? t("Name is missing") : !description ? t("Description is missing") : undefined,
  });

  checks.push({
    key: READINESS_KEYS.source,
    label: t("Source checks clean"),
    status: input.sourceEmpty || input.errors > 0 ? "fail" : "pass",
    detail: input.sourceEmpty
      ? t("Empty")
      : input.errors > 0
        ? t("{0, plural, one {# error} other {# errors}}", input.errors)
        : undefined,
  });

  if (input.warnings > 0) {
    checks.push({
      key: READINESS_KEYS.warnings,
      label: t("No warnings"),
      status: "warn",
      detail: t("{0, plural, one {# warning} other {# warnings}}", input.warnings),
    });
  }

  const tests = input.tests;
  checks.push(
    tests.status === "ran"
      ? {
          key: READINESS_KEYS.tests,
          label: t("Tests pass"),
          status: tests.stale ? "warn" : tests.failed > 0 || tests.run.error ? "fail" : "pass",
          detail: tests.stale
            ? t("Running…")
            : tests.run.error
              ? t("The tests could not run")
              : tests.cells === "failed" || tests.cells === "unresolved"
                ? t("The rate table cells the tests read could not be loaded")
                : t("{0} of {1}", tests.passed, tests.passed + tests.failed),
        }
      : tests.status === "empty"
        ? {
            key: READINESS_KEYS.tests,
            label: t("Tests pass"),
            status: input.testsRequired ? "fail" : "warn",
            detail: input.testsRequired
              ? t("Add at least one test")
              : t("Add a test so a reviewer can see what it bills"),
          }
        : {
            key: READINESS_KEYS.tests,
            label: t("Tests pass"),
            status: "fail",
            detail: t("Fix the errors to run the tests"),
          },
  );

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

/** The goal the readiness meter counts toward, by where the record is. */
export function readinessGoal(
  mode: "create" | "edit",
  status: string | undefined,
  t: TranslateFn,
): string {
  if (mode === "create") return t("Ready to create");
  if (status === "InReview") return t("Ready to approve");
  return t("Ready for review");
}
