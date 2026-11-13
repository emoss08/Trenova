import { cleanup, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ExceptionsList } from "../exceptions-list";
import { UnassignedQueueList, type UnassignedQueueSummary } from "../unassigned-queue-list";

const useUnassignedShipments = vi.fn();
const useExceptionShipments = vi.fn();

vi.mock("../use-work-queues", () => ({
  useUnassignedShipments: (...args: unknown[]) => useUnassignedShipments(...args),
  useExceptionShipments: (...args: unknown[]) => useExceptionShipments(...args),
}));

function unassignedPage(hasNextPage: boolean) {
  return {
    data: {
      pages: [
        {
          count: 1213,
          totalCountExact: true,
          results: [
            { id: "shp_1", status: "New", totalChargeAmount: "1200.50", moves: [] },
            { id: "shp_2", status: "New", totalChargeAmount: "800", moves: [] },
          ],
        },
      ],
    },
    isLoading: false,
    hasNextPage,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  };
}

beforeEach(() => {
  useUnassignedShipments.mockReset();
  useExceptionShipments.mockReset();
  useExceptionShipments.mockReturnValue({ data: { results: [] }, isLoading: false });
});

afterEach(cleanup);

describe("UnassignedQueueList", () => {
  it("asks for only the rows a capped list shows", () => {
    useUnassignedShipments.mockReturnValue(unassignedPage(true));
    render(<UnassignedQueueList limit={8} onSelect={() => {}} />);

    expect(useUnassignedShipments).toHaveBeenLastCalledWith(8, true);
  });

  it("leaves the page size to the queue when it is not capped", () => {
    useUnassignedShipments.mockReturnValue(unassignedPage(true));
    render(<UnassignedQueueList onSelect={() => {}} />);

    expect(useUnassignedShipments).toHaveBeenLastCalledWith(undefined, true);
  });

  it("reports no waiting revenue while the rows read are only part of the queue", () => {
    useUnassignedShipments.mockReturnValue(unassignedPage(true));
    const summaries: UnassignedQueueSummary[] = [];
    render(
      <UnassignedQueueList limit={8} onSelect={() => {}} onSummary={(s) => summaries.push(s)} />,
    );

    expect(summaries.at(-1)).toEqual({
      totalCount: 1213,
      totalCountExact: true,
      pendingRevenue: null,
    });
  });

  it("sums the waiting revenue once the whole queue has been read", () => {
    useUnassignedShipments.mockReturnValue(unassignedPage(false));
    const summaries: UnassignedQueueSummary[] = [];
    render(
      <UnassignedQueueList limit={8} onSelect={() => {}} onSummary={(s) => summaries.push(s)} />,
    );

    expect(summaries.at(-1)?.pendingRevenue).toBe(2000.5);
  });
});

describe("ExceptionsList", () => {
  it("asks for only the rows a capped list shows", () => {
    render(<ExceptionsList limit={8} onSelect={() => {}} />);

    expect(useExceptionShipments).toHaveBeenLastCalledWith("all", 8, true);
  });

  it("leaves the page size to the queue when it is not capped", () => {
    render(<ExceptionsList onSelect={() => {}} />);

    expect(useExceptionShipments).toHaveBeenLastCalledWith("all", undefined, true);
  });
});
