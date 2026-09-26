import { describe, expect, it } from "vitest";
import {
  accountingStartDateSchema,
  accountingSyncSettingsSchema,
  startDateInClosedBooks,
} from "../accounting-start-date-schema";

const latest = 1_780_086_399;

describe("accountingStartDateSchema", () => {
  const schema = accountingStartDateSchema(latest);

  it("accepts a day up to the latest allowed", () => {
    expect(
      schema.safeParse({
        startDate: latest,
        autoSync: true,
        driverSettlements: false,
        backfill: false,
      }).success,
    ).toBe(true);
  });

  it("refuses a day in the future", () => {
    const result = schema.safeParse({
      startDate: latest + 1,
      autoSync: true,
      driverSettlements: false,
      backfill: false,
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe("The start date cannot be in the future");
  });

  it("refuses a missing start date", () => {
    for (const startDate of [null, undefined, 0]) {
      const result = schema.safeParse({
        startDate,
        autoSync: true,
        driverSettlements: false,
        backfill: false,
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0]?.path).toEqual(["startDate"]);
    }
  });
});

describe("accountingStartDateSchema driver settlements", () => {
  it("needs the owner-operator choice to be made", () => {
    const result = accountingStartDateSchema(latest).safeParse({
      startDate: latest,
      autoSync: true,
      backfill: false,
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["driverSettlements"]);
  });
});

describe("accountingSyncSettingsSchema", () => {
  it("takes both switches and a payment policy", () => {
    for (const inboundPayments of ["Propose", "Apply", "Off"]) {
      expect(
        accountingSyncSettingsSchema.safeParse({
          autoSync: false,
          driverSettlements: true,
          inboundPayments,
        }).success,
      ).toBe(true);
    }
  });

  it("refuses a missing switch", () => {
    const result = accountingSyncSettingsSchema.safeParse({
      autoSync: true,
      inboundPayments: "Propose",
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["driverSettlements"]);
  });

  it("refuses a payment policy the server does not know", () => {
    for (const inboundPayments of [undefined, "", "Automatic"]) {
      const result = accountingSyncSettingsSchema.safeParse({
        autoSync: true,
        driverSettlements: false,
        inboundPayments,
      });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0]?.path).toEqual(["inboundPayments"]);
    }
  });
});

describe("startDateInClosedBooks", () => {
  it("is true on or before the closed-through day", () => {
    expect(startDateInClosedBooks(100, 100)).toBe(true);
    expect(startDateInClosedBooks(99, 100)).toBe(true);
  });

  it("is false after it, or when the books are open", () => {
    expect(startDateInClosedBooks(101, 100)).toBe(false);
    expect(startDateInClosedBooks(100, null)).toBe(false);
    expect(startDateInClosedBooks(null, 100)).toBe(false);
  });
});
