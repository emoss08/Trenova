import { useSettledValue } from "@trenova/shared/hooks/use-settled-value";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffTest } from "@trenova/shared/rules/tariff";
import type { TariffContext, TariffTestRun } from "@trenova/shared/types/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { useMemo } from "react";

/** How long the source must rest before the tests run again. */
export const TESTS_SETTLE_MS = 400;

export type TariffTestsState =
  /** The file has errors, so its tests cannot run until they are fixed. */
  | { status: "blocked"; errors: number }
  /** The file has no `test` blocks. */
  | { status: "empty" }
  | {
      status: "ran";
      run: TariffTestRun;
      passed: number;
      failed: number;
      /** The source changed since this run; a new one is on its way. */
      stale: boolean;
    };

/**
 * Runs a file's tests in the browser with `tariff.test` once the source has settled, so the
 * tests panel follows every change without running on each keystroke. A file the checker
 * refuses is not run: its tests would only repeat the errors.
 */
export function useTariffTests({
  rules,
  source,
  context,
  errors,
  testCount,
}: {
  rules: RulesRuntime;
  source: string;
  context: TariffContext | null;
  /** The checker's error count for the current source. */
  errors: number;
  /** How many `test` blocks the outline lists. */
  testCount: number;
}): TariffTestsState {
  const settled = useSettledValue(source, TESTS_SETTLE_MS);
  const runnable = errors === 0 && testCount > 0;
  const settledRunnable = useSettledValue(runnable, TESTS_SETTLE_MS);
  const knowledge = context ?? EMPTY_TARIFF_CONTEXT;

  const run = useMemo(
    () => (settledRunnable ? tariffTest(rules, { source: settled, context: knowledge }) : null),
    [settledRunnable, rules, settled, knowledge],
  );

  if (errors > 0) return { status: "blocked", errors };
  if (testCount === 0) return { status: "empty" };
  if (!run)
    return { status: "ran", run: { tests: [], error: null }, passed: 0, failed: 0, stale: true };
  const passed = run.tests.filter((test) => test.status === "passed").length;
  return {
    status: "ran",
    run,
    passed,
    failed: run.tests.length - passed,
    stale: settled !== source,
  };
}
