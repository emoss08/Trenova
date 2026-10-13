import { describe, expect, it } from "vitest";
import {
  emptyCheckCallForm,
  issueTrackingLinkFormSchema,
  logCheckCallFormSchema,
  toLogCheckCallInput,
  type LogCheckCallFormValues,
} from "../check-call";

function form(overrides: Partial<LogCheckCallFormValues>): LogCheckCallFormValues {
  return { ...emptyCheckCallForm("sm_1"), ...overrides };
}

function issuePaths(values: LogCheckCallFormValues): string[] {
  const result = logCheckCallFormSchema.safeParse(values);
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
}

describe("logCheckCallFormSchema", () => {
  it("accepts a bare position update", () => {
    expect(issuePaths(form({}))).toEqual([]);
  });

  it("asks for the stop on an arrival or departure", () => {
    expect(issuePaths(form({ kind: "Arrival" }))).toEqual(["stopId"]);
    expect(issuePaths(form({ kind: "Departure", stopId: "stp_1" }))).toEqual([]);
  });

  it("asks why when the load is running late", () => {
    expect(issuePaths(form({ delayed: true, delayReason: "  " }))).toEqual(["delayReason"]);
    expect(issuePaths(form({ delayed: true, delayReason: "Weather" }))).toEqual([]);
  });

  it("refuses a reported time in the future", () => {
    const tomorrow = Math.floor(Date.now() / 1000) + 86_400;
    expect(issuePaths(form({ reportedAt: tomorrow }))).toEqual(["reportedAt"]);
  });
});

describe("toLogCheckCallInput", () => {
  it("drops empty text and a stop an update does not name", () => {
    expect(
      toLogCheckCallInput(
        form({ stopId: "stp_1", city: " Tulsa ", stateCode: "ok", note: "   ", reportedAt: 100 }),
      ),
    ).toEqual({
      shipmentMoveId: "sm_1",
      kind: "Update",
      stopId: undefined,
      reportedAt: 100,
      city: "Tulsa",
      stateCode: "OK",
      delayed: false,
      delayReason: undefined,
      estimatedArrival: undefined,
      contactName: undefined,
      note: undefined,
    });
  });

  it("keeps the stop on a visit and the reason only while delayed", () => {
    expect(
      toLogCheckCallInput(
        form({ kind: "Arrival", stopId: "stp_2", delayed: true, delayReason: " Dock full " }),
      ),
    ).toMatchObject({ kind: "Arrival", stopId: "stp_2", delayed: true, delayReason: "Dock full" });
    expect(
      toLogCheckCallInput(form({ delayed: false, delayReason: "left over" })).delayReason,
    ).toBeUndefined();
  });
});

describe("issueTrackingLinkFormSchema", () => {
  it("takes an empty address to mean copy the link", () => {
    expect(issueTrackingLinkFormSchema.safeParse({ recipientEmail: "" }).success).toBe(true);
    expect(
      issueTrackingLinkFormSchema.safeParse({ recipientEmail: "ops@carrier.com" }).success,
    ).toBe(true);
    expect(issueTrackingLinkFormSchema.safeParse({ recipientEmail: "not-an-email" }).success).toBe(
      false,
    );
  });
});
