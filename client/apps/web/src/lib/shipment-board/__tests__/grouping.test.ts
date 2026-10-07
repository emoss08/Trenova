import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { Shipment, ShipmentMove, Stop } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import {
  boardGroupHeaders,
  boardGroupKey,
  collapsedScopeFor,
  consigneeStop,
  dayBounds,
  dayKey,
  groupSort,
  parseCollapsedKeys,
  shipperStop,
} from "../grouping";

const t = ((message: string | null | undefined, ...args: unknown[]) =>
  (message ?? "").replace(/\{(\d+)\}/g, (_, i) => String(args[Number(i)]))) as TranslateFn;

const CHICAGO = "America/Chicago";

function stop(id: string, type: string, sequence: number, scheduledWindowStart = 0): Stop {
  return { id, type, sequence, scheduledWindowStart } as unknown as Stop;
}

function move(sequence: number, stops: Stop[]): ShipmentMove {
  return { sequence, stops } as unknown as ShipmentMove;
}

function shipment(fields: Partial<Shipment>): Shipment {
  return fields as Shipment;
}

describe("shipperStop and consigneeStop", () => {
  it("take the first pickup of the earliest move and the last delivery of the latest", () => {
    const row = shipment({
      moves: [
        move(1, [stop("stp_c", "Delivery", 0), stop("stp_d", "Pickup", 1)]),
        move(0, [
          stop("stp_a", "Delivery", 0),
          stop("stp_b", "SplitPickup", 2),
          stop("stp_e", "Pickup", 3),
        ]),
        move(2, [stop("stp_f", "SplitDelivery", 4), stop("stp_g", "Delivery", 1)]),
      ],
    });

    expect(shipperStop(row)?.id).toBe("stp_b");
    expect(consigneeStop(row)?.id).toBe("stp_f");
  });

  it("settle a tie on sequence by the stop id, read as code points", () => {
    const row = shipment({
      moves: [
        move(0, [
          stop("stp_a01", "Pickup", 0),
          stop("stp_A01", "Pickup", 0),
          stop("stp_b01", "Delivery", 1),
          stop("stp_B01", "Delivery", 1),
        ]),
      ],
    });

    expect(shipperStop(row)?.id).toBe("stp_A01");
    expect(consigneeStop(row)?.id).toBe("stp_b01");
  });

  it("find nothing on a shipment without moves or without that end", () => {
    expect(shipperStop(shipment({}))).toBeNull();
    expect(shipperStop(shipment({ moves: [move(0, [stop("s", "Delivery", 0)])] }))).toBeNull();
    expect(consigneeStop(shipment({ moves: [] }))).toBeNull();
  });
});

describe("dayKey and dayBounds", () => {
  it("read the day in the board's zone, not UTC's", () => {
    expect(dayKey(Date.UTC(2026, 9, 8, 3, 0) / 1000, CHICAGO)).toBe("2026-10-07");
    expect(dayKey(Date.UTC(2026, 9, 8, 3, 0) / 1000, "UTC")).toBe("2026-10-08");
    expect(dayKey(null, CHICAGO)).toBe("");
  });

  it("cover a 25-hour day when the clocks fall back", () => {
    const bounds = dayBounds("2026-11-01", CHICAGO);
    expect(bounds).toEqual({
      start: Date.UTC(2026, 10, 1, 5) / 1000,
      end: Date.UTC(2026, 10, 2, 6) / 1000,
    });
    expect(dayBounds("not-a-day", CHICAGO)).toBeNull();
  });
});

describe("boardGroupKey", () => {
  const rankOf = () => 9;
  const row = shipment({
    customerId: "cus_1",
    ownerId: null,
    moves: [
      move(0, [
        stop("p", "Pickup", 0, Date.UTC(2026, 9, 7, 14) / 1000),
        stop("d", "Delivery", 1, Date.UTC(2026, 9, 9, 4) / 1000),
      ]),
    ],
  } as Partial<Shipment>);

  it("keys each grouping the way the server groups and counts it", () => {
    expect(boardGroupKey("shipDate", row, CHICAGO, rankOf)).toBe("2026-10-07");
    expect(boardGroupKey("deliveryDate", row, CHICAGO, rankOf)).toBe("2026-10-08");
    expect(boardGroupKey("customer", row, CHICAGO, rankOf)).toBe("cus_1");
    expect(boardGroupKey("owner", row, CHICAGO, rankOf)).toBe("");
    expect(boardGroupKey("stage", row, CHICAGO, rankOf)).toBe(9);
    expect(boardGroupKey("shipDate", shipment({ moves: [] }), CHICAGO, rankOf)).toBe("");
  });
});

describe("groupSort", () => {
  it("breaks ties on a name with the id it groups by", () => {
    expect(groupSort("customer")).toEqual({
      field: "customer.name",
      tieBreakers: [{ field: "customerId", direction: "asc" }],
    });
    expect(groupSort("owner").tieBreakers).toEqual([{ field: "ownerId", direction: "asc" }]);
    expect(groupSort("deliveryDate")).toEqual({ field: "consigneeStop.scheduledWindowStart" });
  });
});

describe("collapsedScopeFor", () => {
  it("drops a collapsed day as a range of instants, and the dateless with not-null", () => {
    const scope = collapsedScopeFor("shipDate", CHICAGO)!(["2026-11-01", ""]);
    expect(scope).toEqual({
      fieldFilters: [
        { field: "shipperStop.scheduledWindowStart", operator: "isnotnull", value: null },
      ],
      filterGroups: [
        {
          filters: [
            {
              field: "shipperStop.scheduledWindowStart",
              operator: "lt",
              value: Date.UTC(2026, 10, 1, 5) / 1000,
            },
            {
              field: "shipperStop.scheduledWindowStart",
              operator: "gte",
              value: Date.UTC(2026, 10, 2, 6) / 1000,
            },
            { field: "shipperStop.scheduledWindowStart", operator: "isnull", value: null },
          ],
        },
      ],
    });
  });

  it("keeps shipments without an owner when only named owners are collapsed", () => {
    expect(collapsedScopeFor("owner", CHICAGO)!(["usr_1", "usr_2"])).toEqual({
      fieldFilters: [],
      filterGroups: [
        {
          filters: [
            { field: "ownerId", operator: "notin", value: ["usr_1", "usr_2"] },
            { field: "ownerId", operator: "isnull", value: null },
          ],
        },
      ],
    });
    expect(collapsedScopeFor("owner", CHICAGO)!(["", "usr_1"])).toEqual({
      fieldFilters: [
        { field: "ownerId", operator: "isnotnull", value: null },
        { field: "ownerId", operator: "notin", value: ["usr_1"] },
      ],
      filterGroups: [],
    });
  });

  it("leaves stage to the table's own notin on the rank", () => {
    expect(collapsedScopeFor("stage", CHICAGO)).toBeUndefined();
    expect(collapsedScopeFor("customer", CHICAGO)!(["cus_1"])).toEqual({
      fieldFilters: [{ field: "customerId", operator: "notin", value: ["cus_1"] }],
      filterGroups: [],
    });
  });
});

describe("parseCollapsedKeys", () => {
  it("turns stage keys back into ranks and leaves every other key a string", () => {
    expect(parseCollapsedKeys("stage", ["2", "x", "5"])).toEqual([2, 5]);
    expect(parseCollapsedKeys("owner", ["", "usr_1"])).toEqual(["", "usr_1"]);
  });
});

describe("boardGroupHeaders", () => {
  const now = new Date(Date.UTC(2026, 9, 7, 18));

  it("names days relative to today in the board's zone and skips empty groups", () => {
    const headers = boardGroupHeaders({
      grouping: "deliveryDate",
      groups: [
        { key: "2026-10-07", label: "", count: 3, revenue: "1200.4" },
        { key: "2026-10-08", label: "", count: 1, revenue: "0" },
        { key: "2026-10-12", label: "", count: 2, revenue: "500" },
        { key: "2026-10-13", label: "", count: 0, revenue: "0" },
        { key: "", label: "", count: 4, revenue: "0" },
      ],
      t,
      timezone: CHICAGO,
      now,
    });

    expect(headers.map((header) => header.key)).toEqual([
      "2026-10-07",
      "2026-10-08",
      "2026-10-12",
      "",
    ]);
    expect(headers[0].label).toMatch(/^Today · /);
    expect(headers[1].label).toMatch(/^Tomorrow · /);
    expect(headers[2].label).not.toMatch(/Today|Tomorrow|Yesterday/);
    expect(headers[3].label).toBe("No delivery date");
    expect(headers[0].count).toBe(3);
    expect(headers[0].aggregate).toBe("$1,200");
  });

  it("names owners and customers from the server, with a word for the missing ones", () => {
    const owners = boardGroupHeaders({
      grouping: "owner",
      groups: [
        { key: "usr_1", label: "Dana Ruiz", count: 1, revenue: "0" },
        { key: "usr_9", label: "", count: 1, revenue: "0" },
        { key: "", label: "", count: 1, revenue: "0" },
      ],
      t,
      timezone: CHICAGO,
      now,
    });
    expect(owners.map((header) => header.label)).toEqual(["Dana Ruiz", "Former user", "No owner"]);

    const customers = boardGroupHeaders({
      grouping: "customer",
      groups: [{ key: "cus_1", label: "", count: 2, revenue: "0" }],
      t,
      timezone: CHICAGO,
      now,
    });
    expect(customers[0].label).toBe("Unnamed customer");
  });
});
