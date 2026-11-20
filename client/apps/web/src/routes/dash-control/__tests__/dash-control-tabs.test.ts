import { describe, expect, it } from "vitest";
import { dashControlSchema } from "@/types/dash-control";
import { dashControlTabs } from "../dash-control-tabs";

const RECORD_KEYS = new Set(["id", "version"]);

describe("dash control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = dashControlTabs({ driverPay: true, reminders: true }).flatMap(
      (tab) => tab.fields,
    );
    const editable = Object.keys(dashControlSchema.shape).filter((key) => !RECORD_KEYS.has(key));

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers every tab, in order, on a plan with driver pay and reminders", () => {
    expect(dashControlTabs({ driverPay: true, reminders: true }).map((tab) => tab.value)).toEqual([
      "loads",
      "pay",
      "profile",
      "reminders",
    ]);
  });

  it("leaves out pay and expenses when the plan has no driver settlements", () => {
    expect(dashControlTabs({ driverPay: false, reminders: true }).map((tab) => tab.value)).toEqual([
      "loads",
      "profile",
      "reminders",
    ]);
  });

  it("leaves out reminders when there is nothing to remind drivers of", () => {
    expect(dashControlTabs({ driverPay: true, reminders: false }).map((tab) => tab.value)).toEqual([
      "loads",
      "pay",
      "profile",
    ]);
  });
});
