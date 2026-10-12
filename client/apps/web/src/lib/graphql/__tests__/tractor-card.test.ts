import {
  DeskTractorCardDocument,
  DeskTractorInspectionsDocument,
  DeskTractorPositionDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchDeskTractor, fetchTractorInspections, fetchTractorPosition } from "../tractor-card";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: Record<string, unknown>;
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

function inspection(id: string, startedAt: number) {
  return {
    id,
    inspectionType: "preTrip",
    safetyStatus: "safe",
    workerName: null,
    startedAt,
    location: null,
    odometerMeters: null,
    defectCount: 0,
    unresolvedDefectCount: 0,
  };
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchDeskTractor", () => {
  it("reads the tractor by id and passes the signal on", async () => {
    requestGraphQLMock.mockResolvedValue({ tractor: { id: "tr_1", code: "T-101" } } as never);
    const controller = new AbortController();

    await expect(fetchDeskTractor("tr_1", { signal: controller.signal })).resolves.toEqual({
      id: "tr_1",
      code: "T-101",
    });
    const call = lastCall();
    expect(call.document).toBe(DeskTractorCardDocument);
    expect(call.operationName).toBe("DeskTractorCard");
    expect(call.variables).toEqual({ id: "tr_1" });
    expect(call.signal).toBe(controller.signal);
  });

  it("treats a tractor the server does not return as an error", async () => {
    requestGraphQLMock.mockResolvedValue({ tractor: null } as never);

    await expect(fetchDeskTractor("tr_gone")).rejects.toThrow("Tractor not found");
  });
});

describe("fetchTractorPosition", () => {
  it("is null for a tractor that has never reported a position", async () => {
    requestGraphQLMock.mockResolvedValue({ vehiclePosition: null } as never);

    await expect(fetchTractorPosition("tr_1")).resolves.toBeNull();
    expect(lastCall().document).toBe(DeskTractorPositionDocument);
    expect(lastCall().variables).toEqual({ tractorId: "tr_1" });
  });
});

describe("fetchTractorInspections", () => {
  it("asks for one more than a page, and starts the next page before the last one shown", async () => {
    requestGraphQLMock.mockResolvedValue({
      vehicleInspections: [
        inspection("vinsp_3", 300),
        inspection("vinsp_2", 200),
        inspection("vinsp_1", 100),
      ],
    } as never);

    const page = await fetchTractorInspections("tr_1", { size: 2, cursor: null });

    const call = lastCall();
    expect(call.document).toBe(DeskTractorInspectionsDocument);
    expect(call.variables).toEqual({ tractorId: "tr_1", limit: 3, before: null, beforeId: null });
    expect(page.rows.map((row) => row.id)).toEqual(["vinsp_3", "vinsp_2"]);
    expect(page.next).toEqual({ before: 200, beforeId: "vinsp_2" });
  });

  it("has no next page when the server returned no more than a page", async () => {
    requestGraphQLMock.mockResolvedValue({
      vehicleInspections: [inspection("vinsp_1", 100), inspection("vinsp_0", 100)],
    } as never);

    const page = await fetchTractorInspections("tr_1", {
      size: 2,
      cursor: { before: 200, beforeId: "vinsp_2" },
    });

    expect(lastCall().variables).toMatchObject({ limit: 3, before: 200, beforeId: "vinsp_2" });
    expect(page.rows).toHaveLength(2);
    expect(page.next).toBeNull();
  });

  it("has no rows and no next page for a tractor never inspected", async () => {
    requestGraphQLMock.mockResolvedValue({ vehicleInspections: [] } as never);

    await expect(fetchTractorInspections("tr_1", { size: 5, cursor: null })).resolves.toEqual({
      rows: [],
      next: null,
    });
  });
});
