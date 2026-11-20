import { describe, expect, it } from "vitest";
import { accountingControlSchema } from "@/types/accounting-control";
import { accountingControlTabs } from "../accounting-control-tabs";

const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "organizationId",
  "businessUnitId",
]);

describe("accounting control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const tabs = accountingControlTabs({ runsAssets: true, driverPay: true });
    const placed = tabs.flatMap((tab) => tab.fields);
    const editable = Object.keys(accountingControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers every tab, in order, to a fleet whose plan includes driver pay", () => {
    expect(
      accountingControlTabs({ runsAssets: true, driverPay: true }).map((tab) => tab.value),
    ).toEqual(["posting", "accounts", "driverPay", "close", "currency"]);
  });

  it("leaves out driver pay for an organization that runs no fleet", () => {
    expect(
      accountingControlTabs({ runsAssets: false, driverPay: true }).map((tab) => tab.value),
    ).toEqual(["posting", "accounts", "close", "currency"]);
  });

  it("leaves out driver pay when the plan does not include it", () => {
    expect(
      accountingControlTabs({ runsAssets: true, driverPay: false }).some(
        (tab) => tab.value === "driverPay",
      ),
    ).toBe(false);
  });
});
