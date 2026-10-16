import { afterEach, describe, expect, it } from "vitest";
import { reversalRequestBody, todayField } from "../journal-reversal-request";

describe("reversalRequestBody", () => {
  const zone = process.env.TZ;

  afterEach(() => {
    process.env.TZ = zone;
  });

  it("books the day asked for as that day's first second in UTC, wherever the reader is", () => {
    process.env.TZ = "America/Chicago";
    expect(
      reversalRequestBody({
        originalJournalEntryId: "je_2610_00418",
        requestedAccountingDate: "2026-10-16",
        reasonCode: " DUPLICATE ",
        reasonText: "\tBilled twice.\n",
      }),
    ).toEqual({
      originalJournalEntryId: "je_2610_00418",
      requestedAccountingDate: Date.UTC(2026, 9, 16) / 1000,
      reasonCode: "DUPLICATE",
      reasonText: "Billed twice.",
    });
  });
});

describe("todayField", () => {
  it("writes the reader's own day as a date field does, padded", () => {
    expect(todayField(new Date(2026, 0, 5, 23, 30))).toBe("2026-01-05");
    expect(todayField(new Date(2026, 10, 16, 0, 5))).toBe("2026-11-16");
  });
});
