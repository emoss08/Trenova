import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DataTableGrouping } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { ShipmentBoard } from "../shipment-board";

const { dataTableProps, urlState } = vi.hoisted(() => ({
  dataTableProps: vi.fn(),
  urlState: {
    view: "table",
    group: "stage",
    collapsed: [] as string[],
    expanded: null,
    qf: [],
  },
}));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: unknown) => {
    dataTableProps(props);
    return null;
  },
}));

const QUERY_DATA: Record<string, unknown> = {
  summary: [
    { stage: "Late", rank: 1, count: 3, revenue: "900" },
    { stage: "Moving", rank: 3, count: 5, revenue: "1500" },
  ],
  groups: [
    { key: "2026-10-07", label: "", count: 2, revenue: "800" },
    { key: "", label: "", count: 1, revenue: "0" },
  ],
};

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { queryKey: unknown[]; enabled?: boolean }) => ({
    data: options.enabled === false ? undefined : QUERY_DATA[String(options.queryKey[0])],
    isLoading: false,
  }),
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    shipmentBoard: {
      stageSummary: () => ({ queryKey: ["summary"] }),
      groups: (_input: unknown, groupBy: string) => ({ queryKey: ["groups", groupBy] }),
      quickFilterCounts: () => ({ queryKey: ["quick-filter-counts"] }),
    },
  },
}));

vi.mock("../url-state", () => ({ useShipmentBoardUrl: () => [urlState, vi.fn()] }));
vi.mock("../use-board-scope", () => ({ useBoardScope: () => ({}) }));
vi.mock("../use-board-actions", () => ({
  useBoardActions: () => ({
    tender: { mutateAsync: vi.fn() },
    autoAssign: { mutateAsync: vi.fn() },
  }),
}));
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
    Object.assign(urlState, { view: "table", group: "stage", collapsed: [] });
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
    Object.assign(urlState, { view: "timeline", collapsed: ["3"] });
    render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);

    expect(lastGrouping()?.collapsedKeys).toEqual([3]);
  });

  it("does not group once grouping is turned off", () => {
    urlState.group = "none";
    render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);

    expect(lastGrouping()).toBeUndefined();
  });

  it("groups by delivery day with the server's headers, in the board's order", () => {
    Object.assign(urlState, { group: "deliveryDate", collapsed: ["", "2026-10-07"] });
    render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);

    const grouping = lastGrouping();
    expect(grouping).toMatchObject({
      field: "consigneeStop.scheduledWindowStart",
      collapsedKeys: ["", "2026-10-07"],
    });
    expect(grouping?.groups.map((group) => [group.key, group.count])).toEqual([
      ["2026-10-07", 2],
      ["", 1],
    ]);
    expect(grouping?.groups[1].label).toBe("No delivery date");
    expect(grouping?.collapsedScope?.(grouping.collapsedKeys).fieldFilters).toEqual([
      { field: "consigneeStop.scheduledWindowStart", operator: "isnotnull", value: null },
    ]);
    expect(
      grouping?.getGroupKey({
        moves: [
          {
            sequence: 0,
            stops: [
              {
                id: "stp_1",
                type: "Delivery",
                sequence: 1,
                scheduledWindowStart: Date.UTC(2026, 9, 7, 15) / 1000,
              },
            ],
          },
        ],
      } as unknown as Shipment),
    ).toBe("2026-10-07");
  });

  it("breaks ties between same-named customers on the customer id", () => {
    urlState.group = "customer";
    render(<ShipmentBoard panelOpen={false} onPanelOpenChange={vi.fn()} />);

    expect(lastGrouping()).toMatchObject({
      field: "customer.name",
      tieBreakers: [{ field: "customerId", direction: "asc" }],
    });
  });
});
