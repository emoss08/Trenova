import type { ReportDashboardTile } from "@/types/report";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDashboardFilterCandidates } from "../use-tile-report";

/**
 * The candidates were read out of a page of the report library, so a dashboard
 * whose tile pointed at a report past that page offered no filters at all —
 * silently, because an empty candidate list looks exactly like a report with
 * nothing worth filtering on. They are resolved by id now, which is bounded by
 * how many tiles the dashboard has rather than by how large the library is.
 */

const mocks = vi.hoisted(() => ({
  useReportDefinitionsByIds: vi.fn(),
  useCannedReports: vi.fn(),
}));

vi.mock("@/hooks/use-reports", () => ({
  useReportDefinitionsByIds: mocks.useReportDefinitionsByIds,
  useCannedReports: mocks.useCannedReports,
  useReportDefinition: vi.fn(() => ({ data: undefined, isLoading: false })),
}));

function irWithFilter(entity: string, field: string) {
  return {
    irVersion: 1,
    entity,
    columns: [{ id: "c1", kind: "dimension", ref: { field: "serviceType" } }],
    filters: { op: "and", filters: [{ ref: { field } }] },
  };
}

function tile(overrides: Partial<ReportDashboardTile>): ReportDashboardTile {
  return {
    id: "tile_1",
    kind: "table",
    x: 0,
    y: 0,
    w: 6,
    h: 5,
    ...overrides,
  } as ReportDashboardTile;
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.useCannedReports.mockReturnValue({ data: [], isLoading: false });
  mocks.useReportDefinitionsByIds.mockReturnValue({ data: [], isLoading: false });
});

describe("useDashboardFilterCandidates", () => {
  it("asks only for the reports its own tiles point at", () => {
    renderHook(() =>
      useDashboardFilterCandidates([
        tile({ id: "t1", definitionId: "rdef_1" }),
        tile({ id: "t2", definitionId: "rdef_2" }),
      ]),
    );

    expect(mocks.useReportDefinitionsByIds).toHaveBeenCalledWith(["rdef_1", "rdef_2"]);
  });

  it("asks for each report once even when several tiles share it", () => {
    renderHook(() =>
      useDashboardFilterCandidates([
        tile({ id: "t1", definitionId: "rdef_1" }),
        tile({ id: "t2", definitionId: "rdef_1" }),
      ]),
    );

    expect(mocks.useReportDefinitionsByIds).toHaveBeenCalledWith(["rdef_1"]);
  });

  it("asks for nothing when every tile reads from the gallery", () => {
    renderHook(() => useDashboardFilterCandidates([tile({ id: "t1", cannedKey: "on_time" })]));

    expect(mocks.useReportDefinitionsByIds).toHaveBeenCalledWith([]);
  });

  // The id is what makes this reachable: a report sitting past the first page of
  // the library contributed nothing before.
  it("offers the filters of a report wherever it sits in the library", () => {
    mocks.useReportDefinitionsByIds.mockReturnValue({
      data: [
        {
          id: "rdef_9000",
          name: "Detention Recovery",
          definition: irWithFilter("shipment", "customerId"),
        },
      ],
      isLoading: false,
    });

    const { result } = renderHook(() =>
      useDashboardFilterCandidates([tile({ id: "t1", definitionId: "rdef_9000" })]),
    );

    expect(result.current.entities).toEqual(["shipment"]);
    expect(result.current.candidates.map((entry) => entry.ref.field).sort()).toEqual([
      "customerId",
      "serviceType",
    ]);
    expect(result.current.candidates[0].reports).toEqual(["Detention Recovery"]);
  });

  it("still reads gallery reports from the canned catalog", () => {
    mocks.useCannedReports.mockReturnValue({
      data: [
        {
          key: "on_time",
          name: "On-Time Service",
          definition: irWithFilter("shipment", "originId"),
        },
      ],
      isLoading: false,
    });

    const { result } = renderHook(() =>
      useDashboardFilterCandidates([tile({ id: "t1", cannedKey: "on_time" })]),
    );

    expect(result.current.candidates.map((entry) => entry.ref.field).sort()).toEqual([
      "originId",
      "serviceType",
    ]);
  });
});
