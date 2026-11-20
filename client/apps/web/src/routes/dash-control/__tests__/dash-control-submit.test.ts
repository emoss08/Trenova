import { describe, expect, it } from "vitest";
import type { DashControl } from "@/types/dash-control";
import { prepareDashControlForSubmit } from "../dash-control-submit";
import { savedDashControl } from "./dash-control-fixture";

const record = (changes: Partial<DashControl>): DashControl => ({
  ...savedDashControl,
  ...changes,
});

describe("prepareDashControlForSubmit", () => {
  it("sends declines off when drivers do not accept assignments", () => {
    expect(
      prepareDashControlForSubmit(
        record({ requireLoadAcknowledgment: false, allowLoadRefusals: true }),
      ),
    ).toEqual(record({ requireLoadAcknowledgment: false, allowLoadRefusals: false }));
  });

  it("keeps declines while drivers accept assignments", () => {
    const values = record({ requireLoadAcknowledgment: true, allowLoadRefusals: true });
    expect(prepareDashControlForSubmit(values)).toEqual(values);
  });

  it("sends pay estimates off when pay per load is hidden", () => {
    expect(
      prepareDashControlForSubmit(record({ showLoadPay: false, showPayEstimates: true })),
    ).toEqual(record({ showLoadPay: false, showPayEstimates: false }));
  });

  it("keeps pay estimates while pay per load shows", () => {
    const values = record({ showLoadPay: true, showPayEstimates: true });
    expect(prepareDashControlForSubmit(values)).toEqual(values);
  });

  it("sends receipts not required when drivers cannot claim expenses", () => {
    expect(
      prepareDashControlForSubmit(
        record({ allowExpenseSubmission: false, requireExpenseReceipt: true }),
      ),
    ).toEqual(record({ allowExpenseSubmission: false, requireExpenseReceipt: false }));
  });

  it("keeps the receipt rule while drivers claim expenses", () => {
    const values = record({ allowExpenseSubmission: true, requireExpenseReceipt: true });
    expect(prepareDashControlForSubmit(values)).toEqual(values);
  });

  it("sends one notice per item when reminders are off, keeping the round-up day", () => {
    expect(
      prepareDashControlForSubmit(
        record({
          sendCredentialReminders: false,
          driverDigestCadence: "Weekly",
          driverDigestWeekday: 4,
        }),
      ),
    ).toEqual(
      record({
        sendCredentialReminders: false,
        driverDigestCadence: "Immediate",
        driverDigestWeekday: 4,
      }),
    );
  });

  it("keeps a round-up while reminders are on", () => {
    const values = record({
      sendCredentialReminders: true,
      driverDigestCadence: "Daily",
    });
    expect(prepareDashControlForSubmit(values)).toEqual(values);
  });

  it("sends the two-hour threshold when alerts are off and the threshold was emptied", () => {
    expect(
      prepareDashControlForSubmit(
        record({ enableDetentionAlerts: false, detentionAlertThresholdMinutes: null }),
      ),
    ).toEqual(record({ enableDetentionAlerts: false, detentionAlertThresholdMinutes: 120 }));
  });

  it("sends the two-hour threshold when alerts are off and the threshold is out of range", () => {
    expect(
      prepareDashControlForSubmit(
        record({ enableDetentionAlerts: false, detentionAlertThresholdMinutes: 5 }),
      ),
    ).toEqual(record({ enableDetentionAlerts: false, detentionAlertThresholdMinutes: 120 }));
  });

  it("keeps a valid threshold while alerts are off", () => {
    const values = record({ enableDetentionAlerts: false, detentionAlertThresholdMinutes: 90 });
    expect(prepareDashControlForSubmit(values)).toEqual(values);
  });

  it("leaves the values it was given untouched", () => {
    const values = record({ requireLoadAcknowledgment: false, allowLoadRefusals: true });

    prepareDashControlForSubmit(values);

    expect(values.allowLoadRefusals).toBe(true);
  });
});
