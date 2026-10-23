import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { rulesSync } from "@trenova/shared/rules/runtime";
import {
  tariffContextSchema,
  tariffTableCellsSchema,
  type TariffNeed,
  type TariffTableCells,
} from "@trenova/shared/types/tariff";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const getContextCells =
  vi.fn<(request: { needs: readonly TariffNeed[]; whole: boolean }) => Promise<TariffTableCells>>();

vi.mock("@/services/api", () => ({
  apiService: { formulaTemplateService: { getContextCells } },
}));

const { useTariffTests } = await import("../use-tariff-tests");

const STAMP = "1:3:1767225600";

const SOURCE = `tariff "Grid"

let rate = tables.Grid["A", 100 lb] ?? fail "No rate"
charge "Linehaul" (quantity: 100 lb, rate: rate)

test "bills the grid" {
  expect total == $20
}
`;

const context = tariffContextSchema.parse({
  tables: [
    {
      code: "Grid",
      valueType: "Money/cwt",
      axes: [
        { match: "Exact", kind: "Zone" },
        { match: "Range", kind: "WeightBreak", unit: "lb" },
      ],
      cells: null,
      loaded: [],
    },
  ],
  codes: [],
  settings: { distanceUnit: "mi" },
});

const cellsForA = tariffTableCellsSchema.parse({
  stamp: STAMP,
  tables: [
    { code: "Grid", cells: [{ at: [{ key: "A" }, { min: "0" }], value: "20" }], loaded: [["A"]] },
  ],
});

function render() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidate = vi.spyOn(client, "invalidateQueries");
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  const hook = renderHook(
    () =>
      useTariffTests({
        rules: rulesSync(),
        source: SOURCE,
        context,
        tablesStamp: STAMP,
        errors: 0,
        testCount: 1,
      }),
    { wrapper },
  );
  return { ...hook, invalidate };
}

describe("useTariffTests with a rate table the context carries by shape only", () => {
  beforeEach(() => {
    getContextCells.mockReset();
  });

  it("loads the cells its tests read and runs them again", async () => {
    getContextCells.mockResolvedValue(cellsForA);
    const { result, invalidate } = render();

    await waitFor(() => {
      expect(result.current).toMatchObject({ status: "ran", passed: 1, failed: 0, stale: false });
    });
    expect(getContextCells).toHaveBeenCalledTimes(1);
    expect(getContextCells).toHaveBeenCalledWith({
      needs: [{ table: "Grid", keys: ["A"] }],
      whole: false,
    });
    expect(result.current).toMatchObject({ cells: "ready" });
    expect(invalidate).not.toHaveBeenCalled();
  });

  it("reads the context again when the cells come from a newer stamp", async () => {
    getContextCells.mockResolvedValue({ ...cellsForA, stamp: "1:4:1767312000" });
    const { invalidate } = render();

    await waitFor(() => {
      expect(invalidate).toHaveBeenCalledWith({ queryKey: ["formulaTemplate", "context"] });
    });
  });

  it("loads three times by combination, then the tables whole, then stops", async () => {
    getContextCells.mockResolvedValue({ stamp: STAMP, tables: [] });
    const { result } = render();

    await waitFor(() => {
      expect(result.current).toMatchObject({ cells: "unresolved", stale: false });
    });
    expect(getContextCells.mock.calls.map(([request]) => request.whole)).toEqual([
      false,
      false,
      false,
      true,
    ]);
    if (result.current.status !== "ran") throw new Error("the tests ran");
    expect(result.current.run.tests[0]?.status).toBe("needs");
  });

  it("reports a failed load, and loads again when asked", async () => {
    getContextCells.mockRejectedValueOnce(new Error("offline")).mockResolvedValue(cellsForA);
    const { result } = render();

    await waitFor(() => {
      expect(result.current).toMatchObject({ cells: "failed", stale: false });
    });
    expect(getContextCells).toHaveBeenCalledTimes(1);

    if (result.current.status !== "ran") throw new Error("the tests ran");
    const retry = result.current.retryCells;
    act(() => retry());
    await waitFor(() => {
      expect(result.current).toMatchObject({ passed: 1, cells: "ready" });
    });
    expect(getContextCells).toHaveBeenCalledTimes(2);
  });
});
