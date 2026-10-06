import { describe, expect, it } from "vitest";
import {
  addQuickFilter,
  parseQuickFilters,
  quickFilterKey,
  removeQuickFilter,
  serializeQuickFilters,
} from "../quick-filters";

describe("quick filters", () => {
  it("round-trips through the URL, dropping anything the server would refuse", () => {
    const filters = [
      { filter: "Late" as const },
      { filter: "DeliveryHour" as const, hour: 14 },
      { filter: "PickupWindow" as const, windowStartMinutes: 120, windowEndMinutes: 360 },
    ];
    expect(parseQuickFilters(serializeQuickFilters(filters))).toEqual(filters);
    expect(parseQuickFilters('[{"filter":"Nope"},{"filter":"Late"}]')).toEqual([
      { filter: "Late" },
    ]);
    expect(parseQuickFilters('[{"filter":"DeliveryHour","hour":24}]')).toEqual([]);
    expect(parseQuickFilters('[{"filter":"DeliveryHour"}]')).toEqual([]);
    expect(parseQuickFilters('[{"filter":"PickupWindow","windowStartMinutes":-5}]')).toEqual([]);
    expect(parseQuickFilters("not json")).toEqual([]);
    expect(serializeQuickFilters([])).toBe("");
  });

  it("keeps one filter per kind, the newest winning", () => {
    const first = addQuickFilter([], { filter: "DeliveryHour", hour: 9 });
    const second = addQuickFilter(first, { filter: "DeliveryHour", hour: 14 });
    expect(second).toEqual([{ filter: "DeliveryHour", hour: 14 }]);
    expect(addQuickFilter(second, { filter: "Late" })).toEqual([
      { filter: "DeliveryHour", hour: 14 },
      { filter: "Late" },
    ]);
  });

  it("removes by identity", () => {
    const list = [{ filter: "Late" as const }, { filter: "Reefer" as const }];
    expect(removeQuickFilter(list, { filter: "Late" })).toEqual([{ filter: "Reefer" }]);
    expect(quickFilterKey({ filter: "DeliveryHour", hour: 3 })).not.toBe(
      quickFilterKey({ filter: "DeliveryHour", hour: 4 }),
    );
  });
});

describe("quickFilterLabel", () => {
  const t = (message: string, ...args: Array<string | number>) =>
    args.reduce<string>((text, arg, i) => text.replace(`{${i}}`, String(arg)), message);

  it("names hours and windows the way a dispatcher reads them", async () => {
    const { quickFilterLabel } = await import("../quick-filters");
    expect(quickFilterLabel({ filter: "Late" }, t)).toBe("Late");
    expect(quickFilterLabel({ filter: "DeliveringToday" }, t)).toBe("Delivering today");
    expect(quickFilterLabel({ filter: "DeliveryHour", hour: 23 }, t)).toBe("Delivering 23:00–00:00");
    expect(
      quickFilterLabel({ filter: "PickupWindow", windowStartMinutes: 0, windowEndMinutes: 120 }, t),
    ).toBe("Pickup in < 2h");
    expect(
      quickFilterLabel({ filter: "PickupWindow", windowStartMinutes: 120, windowEndMinutes: 360 }, t),
    ).toBe("Pickup in 2–6h");
    expect(quickFilterLabel({ filter: "PickupWindow", windowStartMinutes: 760 }, t)).toBe(
      "Pickup after 13h",
    );
  });
});
