import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchFormulaTemplateTables, staleTablePins } from "../formula-template-tables";
import { fetchRateMatrixRecord, RateMatrixNotFoundError } from "../rate-matrix";

const requestGraphQL = vi.fn();

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: (...args: unknown[]) => requestGraphQL(...args),
}));

function version(versionNumber: number) {
  return { id: `rmv_${versionNumber}`, versionNumber };
}

describe("fetchRateMatrixRecord", () => {
  beforeEach(() => requestGraphQL.mockReset());

  it("orders the versions newest first, whatever order the server sends", async () => {
    requestGraphQL.mockResolvedValue({
      rateMatrix: { id: "rmx_1", code: "LANES" },
      rateMatrixVersions: [version(1), version(3), version(2)],
      rateMatrixReviews: [{ id: "rmr_1" }],
      rateMatrixUsages: [],
    });

    const record = await fetchRateMatrixRecord("rmx_1");

    expect(record.versions.map((entry) => entry.versionNumber)).toEqual([3, 2, 1]);
    expect(record.reviews).toEqual([{ id: "rmr_1" }]);
    expect(record.usages).toEqual([]);
    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "RateMatrixRecord", variables: { id: "rmx_1" } }),
    );
  });

  it("raises not found when the matrix is not the organisation's", async () => {
    requestGraphQL.mockResolvedValue({
      rateMatrix: null,
      rateMatrixVersions: [],
      rateMatrixReviews: [],
      rateMatrixUsages: [],
    });

    await expect(fetchRateMatrixRecord("rmx_x")).rejects.toBeInstanceOf(RateMatrixNotFoundError);
  });
});

describe("fetchFormulaTemplateTables", () => {
  beforeEach(() => requestGraphQL.mockReset());

  it("reads no tables for a template that is not there", async () => {
    requestGraphQL.mockResolvedValue({ formulaTemplate: null });

    expect(await fetchFormulaTemplateTables("ft_x")).toEqual([]);
  });
});

describe("staleTablePins", () => {
  const base = { rateMatrixId: "rmx_1", archived: false };

  it("keeps the pins a newer published version is out for, skipping archived tables", () => {
    const tables = [
      { ...base, code: "Fresh", versionNumber: 3, latestVersionNumber: 3 },
      { ...base, code: "Stale", versionNumber: 2, latestVersionNumber: 4 },
      { ...base, code: "Gone", versionNumber: 1, latestVersionNumber: 2, archived: true },
    ];

    expect(staleTablePins(tables).map((table) => table.code)).toEqual(["Stale"]);
  });
});
