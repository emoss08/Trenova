import type { DeskAttentionGroup, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { groupLenses, groupName, lensFromParam, lensParam, DESK_LENSES } from "../lens";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function item(overrides: Partial<DeskAttentionItem>): DeskAttentionItem {
  return {
    id: "wt_1",
    title: "Projected past its window",
    kindLabel: "ETA at risk",
    record: null,
    deskLens: "Watchtower",
    aboutId: null,
    ...overrides,
  } as DeskAttentionItem;
}

function group(overrides: Partial<DeskAttentionGroup>): DeskAttentionGroup {
  return {
    key: "shp_1",
    record: null,
    severity: "Warning",
    dueAt: null,
    dueLabel: null,
    unseen: false,
    items: [item({})],
    ...overrides,
  };
}

describe("Needs you lenses", () => {
  it("reads every lens back from the address it writes", () => {
    for (const lens of DESK_LENSES) {
      expect(lensFromParam(lensParam(lens) || null)).toBe(lens);
    }
  });

  it("falls back to everything for an address it does not know", () => {
    expect(lensFromParam("archive")).toBe("Everything");
    expect(lensFromParam(null)).toBe("Everything");
  });

  it("names each lens a row has signals in once, in the page's order", () => {
    const row = group({
      items: [
        item({ id: "wt_1", deskLens: "Decisions" }),
        item({ id: "wt_2", deskLens: "Watchtower" }),
        item({ id: "wt_3", deskLens: "Decisions" }),
        item({ id: "wt_4", deskLens: "Triage" }),
      ],
    });

    expect(groupLenses(row)).toEqual(["Watchtower", "Triage", "Decisions"]);
  });
});

describe("a row's name", () => {
  it("is the record's label and subtitle when the reader may read it", () => {
    const row = group({
      record: {
        kind: "shipment",
        noun: "Shipment",
        id: "shp_1",
        label: "S-240412",
        subtitle: "Dallas, TX → Memphis, TN",
        status: "InTransit",
        readable: true,
      },
    });

    expect(groupName(t, row)).toBe("S-240412 · Dallas, TX → Memphis, TN");
  });

  it("never names a record the reader may not read", () => {
    const row = group({
      record: {
        kind: "shipment",
        noun: "Shipment",
        id: "shp_1",
        label: "",
        subtitle: "",
        status: "",
        readable: false,
      },
    });

    expect(groupName(t, row)).toBe("A record you can't open");
  });

  it("is what its lead item is about when it is about nothing beyond itself", () => {
    const row = group({
      items: [item({ record: { kind: "inbound_message", noun: "Message", label: "Rate con for S-1" } })],
    });

    expect(groupName(t, row)).toBe("Rate con for S-1");
  });
});
