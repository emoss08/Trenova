import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { watchtowerPageContext } from "../page-context";
import { watchtowerItem } from "./item-fixture";

const t = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

describe("watchtowerPageContext", () => {
  it("names the record, selects the item on the watchtower and carries what the card showed", () => {
    const item = watchtowerItem({
      dueAt: null,
      facts: [{ label: "Customer", value: "Northline Foods", at: null }],
      impact: ["Late fee $250"],
    });

    expect(watchtowerPageContext(t, item, 1_800_000_000, "UTC")).toEqual({
      path: "/desk/watchtower?item=wt_1",
      entityType: "shipment",
      entityId: "shp_1",
      title: item.title,
      view: {
        resource: "watchtower",
        selection: { count: 1, ids: ["wt_1"] },
        kpis: [
          { label: "Customer", value: "Northline Foods" },
          { label: "At stake", value: "Late fee $250" },
        ],
      },
      draft: null,
    });
  });

  it("names no record for an item about none", () => {
    const context = watchtowerPageContext(
      t,
      watchtowerItem({ subjectType: null, subjectId: null }),
      1_800_000_000,
      "UTC",
    );

    expect(context.entityType).toBe("");
    expect(context.entityId).toBe("");
  });

  it("keeps each figure inside the bounds the server holds a page to", () => {
    const context = watchtowerPageContext(
      t,
      watchtowerItem({
        dueAt: null,
        impact: [],
        facts: [{ label: "L".repeat(120), value: "V".repeat(200), at: null }],
      }),
      1_800_000_000,
      "UTC",
    );

    expect(context.view?.kpis).toEqual([{ label: "L".repeat(80), value: "V".repeat(60) }]);
  });

  it("counts the bounds in bytes, as the server does, for text that is not ASCII", () => {
    const context = watchtowerPageContext(
      t,
      watchtowerItem({
        dueAt: null,
        impact: [],
        facts: [{ label: "客户", value: "瞭望台".repeat(30), at: null }],
      }),
      1_800_000_000,
      "UTC",
    );

    const value = context.view?.kpis?.[0].value ?? "";
    expect(new TextEncoder().encode(value).length).toBeLessThanOrEqual(60);
    expect(value).toBe("瞭望台".repeat(6) + "瞭望");
  });
});
