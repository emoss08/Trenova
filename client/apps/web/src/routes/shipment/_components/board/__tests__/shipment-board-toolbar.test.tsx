import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DataTableToolbarSlots } from "@trenova/shared/types/data-table";
import { ShipmentBoard } from "../shipment-board";

const { dataTableProps, urlState, setUrl } = vi.hoisted(() => ({
  dataTableProps: vi.fn(),
  setUrl: vi.fn(),
  urlState: {
    view: "table",
    group: true,
    collapsed: [] as number[],
    expanded: null,
    qf: [] as { filter: string }[],
  },
}));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: unknown) => {
    dataTableProps(props);
    return null;
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: [
      { stage: "Late", rank: 1, count: 3, revenue: "900" },
      { stage: "Moving", rank: 3, count: 5, revenue: "1500" },
    ],
    isLoading: false,
  }),
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    shipmentBoard: {
      stageSummary: () => ({ queryKey: ["summary"] }),
      quickFilterCounts: () => ({ queryKey: ["quick-filter-counts"] }),
    },
  },
}));

vi.mock("../url-state", () => ({ useShipmentBoardUrl: () => [urlState, setUrl] }));
vi.mock("../use-board-scope", () => ({ useBoardScope: () => ({}) }));
vi.mock("../use-board-actions", () => ({ useBoardActions: () => ({}) }));
vi.mock("../record-actions", () => ({
  useShipmentRecordActions: () => ({
    rowActions: [],
    edit: vi.fn(),
    copyLink: vi.fn(),
    copyProNumber: vi.fn(),
  }),
}));
vi.mock("@/hooks/use-user-timezone", () => ({ useUserTimezone: () => "UTC" }));
vi.mock("@/lib/shipment-board/capabilities", () => ({
  useShipmentCapabilities: () => ({
    ai: true,
    operationType: "both",
    runsAssets: true,
    runsBrokerage: true,
  }),
}));

type BoardProps = {
  toolbar: DataTableToolbarSlots & Record<string, unknown>;
  initialColumnPinning?: { left: string[]; right: string[] };
};

function lastProps() {
  return dataTableProps.mock.calls.at(-1)?.[0] as BoardProps;
}

function renderBoard() {
  render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);
  return lastProps();
}

describe("ShipmentBoard toolbar", () => {
  beforeEach(() => {
    dataTableProps.mockClear();
    setUrl.mockClear();
    urlState.qf = [];
  });

  it("keeps the table's standard search and filter builder", () => {
    const { toolbar } = renderBoard();

    expect(toolbar.search).toBeUndefined();
    expect(toolbar.filter).toBeUndefined();
    expect(toolbar.searchShortcut).toBe("/");
  });

  it("offers the quick filters from the search field and adds the one picked", () => {
    urlState.qf = [{ filter: "Late" }];
    const { toolbar } = renderBoard();

    const items = toolbar.searchSuggestions?.items ?? [];
    expect(items.map((item) => item.label)).toEqual([
      "Late",
      "Uncovered",
      "Moving",
      "Delivering today",
      "Reefer",
      "Low margin",
    ]);
    expect(items[0].selected).toBe(true);

    items[1].onSelect();
    expect(setUrl).toHaveBeenCalledWith({
      qf: [{ filter: "Late" }, { filter: "Uncovered" }],
      expanded: null,
    });
  });

  it("shows applied quick filters as chips that remove and clear them", () => {
    urlState.qf = [{ filter: "Late" }, { filter: "Reefer" }];
    const { toolbar } = renderBoard();

    expect(toolbar.chips?.items.map((chip) => chip.label)).toEqual(["Late", "Reefer"]);
    toolbar.chips?.items[0].onRemove();
    expect(setUrl).toHaveBeenLastCalledWith({ qf: [{ filter: "Reefer" }], expanded: null });
    toolbar.chips?.onClear();
    expect(setUrl).toHaveBeenLastCalledWith({ qf: [], expanded: null });
  });

  it("pins the selection and lane columns", () => {
    expect(renderBoard().initialColumnPinning).toEqual({ left: ["select", "lane"], right: [] });
  });
});
