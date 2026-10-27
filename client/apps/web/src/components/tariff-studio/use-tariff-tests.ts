import { TABLE_CELL_PASSES, withTableCells } from "@/lib/tariff-table-cells";
import { useSettledValue } from "@trenova/shared/hooks/use-settled-value";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffTest } from "@trenova/shared/rules/tariff";
import type { TariffContext, TariffTestRun } from "@trenova/shared/types/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTariffTableCells } from "./use-tariff-table-cells";

const NO_NEEDS: TariffTestRun["needs"] = [];

/** How long the source must rest before the tests run again. */
export const TESTS_SETTLE_MS = 400;

/**
 * Where the rate table cells the tests read stand: all carried (`ready`), being loaded, not
 * loaded because the request failed, or still missing after every load (`unresolved`).
 */
export type TariffTestCells = "ready" | "loading" | "failed" | "unresolved";

export type TariffTestsState =
  /** The file has errors, so its tests cannot run until they are fixed. */
  | { status: "blocked"; errors: number }
  /** The file has no `test` or `property` blocks. */
  | { status: "empty" }
  | {
      status: "ran";
      run: TariffTestRun;
      passed: number;
      failed: number;
      /** The source changed, or cells are loading, since this run; a new one is on its way. */
      stale: boolean;
      cells: TariffTestCells;
      /** Loads the cells again after a failed load. */
      retryCells: () => void;
    };

/**
 * Runs a file's tests in the browser with `tariff.test` once the source has settled, so the
 * tests panel follows every change without running on each keystroke. A file the checker
 * refuses is not run: its tests would only repeat the errors.
 *
 * The context carries only the shapes of rate tables with an exact axis. When a run answers the
 * cells its tests read (`needs`), they are loaded from the server (`useTariffTableCells`), laid
 * into the context and the tests run again: three loads of the combinations needed, then the
 * tables still needed whole, as the server's own test runs do.
 */
export function useTariffTests({
  rules,
  source,
  context,
  tablesStamp,
  errors,
  testCount,
}: {
  rules: RulesRuntime;
  source: string;
  context: TariffContext | null;
  /** The rate table stamp the context was read at. */
  tablesStamp: string;
  /** The checker's error count for the current source. */
  errors: number;
  /** How many `test` and `property` blocks the outline lists. */
  testCount: number;
}): TariffTestsState {
  const settled = useSettledValue(source, TESTS_SETTLE_MS);
  const runnable = errors === 0 && testCount > 0;
  const settledRunnable = useSettledValue(runnable, TESTS_SETTLE_MS);
  const cells = useTariffTableCells(tablesStamp);
  const knowledge = useMemo(
    () => withTableCells(context ?? EMPTY_TARIFF_CONTEXT, cells.cache),
    [context, cells.cache],
  );

  const run = useMemo(
    () => (settledRunnable ? tariffTest(rules, { source: settled, context: knowledge }) : null),
    [settledRunnable, rules, settled, knowledge],
  );

  const [passes, setPasses] = useState({ source: "", count: 0 });
  const used = passes.source === settled ? passes.count : 0;
  const needs = run?.needs ?? NO_NEEDS;
  const { load, loading, reset } = cells;
  // A failed load stops this source's loads until Retry; a new source loads afresh.
  const failed = cells.failed && passes.source === settled;
  const inFlight = useRef(false);

  useEffect(() => {
    if (needs.length === 0 || loading || failed || inFlight.current) return;
    if (used >= TABLE_CELL_PASSES) return;
    inFlight.current = true;
    if (cells.failed) reset();
    const loaded = { source: settled, count: used + 1 };
    void load(needs, used === TABLE_CELL_PASSES - 1).finally(() => {
      inFlight.current = false;
      setPasses(loaded);
    });
  }, [needs, loading, failed, cells.failed, used, settled, load, reset]);

  if (errors > 0) return { status: "blocked", errors };
  if (testCount === 0) return { status: "empty" };
  const cellState: TariffTestCells =
    needs.length === 0
      ? "ready"
      : failed
        ? "failed"
        : used >= TABLE_CELL_PASSES && !loading
          ? "unresolved"
          : "loading";
  const retryCells = () => {
    setPasses({ source: settled, count: 0 });
    reset();
  };
  if (!run) {
    return {
      status: "ran",
      run: { tests: [], needs: [], error: null, coverage: null },
      passed: 0,
      failed: 0,
      stale: true,
      cells: "ready",
      retryCells,
    };
  }
  const passed = run.tests.filter((test) => test.status === "passed").length;
  return {
    status: "ran",
    run,
    passed,
    failed: run.tests.length - passed,
    stale: settled !== source || cellState === "loading",
    cells: cellState,
    retryCells,
  };
}
