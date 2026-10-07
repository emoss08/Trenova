import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DataTableGrouping } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { ShipmentBoard } from "../shipment-board";

const { dataTableProps, urlState } = vi.hoisted(() => ({
  dataTableProps: vi.fn(),
  urlState: { view: "table", group: true, collapsed: [] as number[], expanded: null, qf: [] },
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

vi.mock("../url-state", () => ({ useShipmentBoardUrl: () => [urlState, vi.fn()] }));
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

function lastGrouping() {
  const props = dataTableProps.mock.calls.at(-1)?.[0] as
    | { grouping?: DataTableGrouping<Shipment> }
    | undefined;
  return props?.grouping;
}

describe("ShipmentBoard grouping", () => {
  beforeEach(() => {
    dataTableProps.mockClear();
    Object.assign(urlState, { view: "table", group: true, collapsed: [] });
  });

  it.each(["table", "timeline", "map"])(
    "orders the %s view by stage, so every view pages through the same rows",
    (view) => {
      urlState.view = view;
      render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);

      expect(lastGrouping()).toMatchObject({ field: "stageRank", collapsedKeys: [] });
    },
  );

  it("leaves collapsed stages out of the timeline as it does the table", () => {
    Object.assign(urlState, { view: "timeline", collapsed: [3] });
    render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);

    expect(lastGrouping()?.collapsedKeys).toEqual([3]);
  });

  it("does not group once grouping is turned off", () => {
    urlState.group = false;
    render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);

    expect(lastGrouping()).toBeUndefined();
  });
});
