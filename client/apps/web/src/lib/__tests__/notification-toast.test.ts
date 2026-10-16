import { describe, expect, it } from "vitest";
import {
  billingTransferRunIdOf,
  reportRunIdOf,
  shouldToastNotification,
} from "../notification-toast";

function stamped(urgent: boolean) {
  return { urgent, desktop: urgent, sound: urgent, email: urgent ? "now" : "none" };
}

describe("shouldToastNotification", () => {
  it("leaves a watchtower item the server did not mark urgent in the bell", () => {
    expect(
      shouldToastNotification({
        eventType: "watchtower.critical",
        title: "Winter storm warning on I-80",
        data: {
          link: "/desk/watchtower?item=wti_01",
          delivery: stamped(false),
        },
      }),
    ).toBe(false);
  });

  it("leaves a notification without a delivery stamp in the bell", () => {
    expect(
      shouldToastNotification({ eventType: "tender_accepted", title: "Tender accepted", data: {} }),
    ).toBe(false);
    expect(
      shouldToastNotification({
        eventType: "tender_accepted",
        title: "Tender accepted",
        data: null,
      }),
    ).toBe(false);
  });

  it("toasts what the server stamped urgent", () => {
    expect(
      shouldToastNotification({
        eventType: "tender_waterfall_exhausted",
        title: "No carrier accepted",
        data: { delivery: stamped(true) },
      }),
    ).toBe(true);
  });

  it("toasts the results of work the person asked for", () => {
    expect(
      shouldToastNotification({
        eventType: "report_run_completed",
        title: "Report ready",
        data: { runId: "rrun_01", delivery: stamped(false) },
      }),
    ).toBe(true);
    expect(
      shouldToastNotification({
        eventType: "billing_transfer_run_completed",
        title: "Transfer finished",
        data: { runId: "btr_01", delivery: stamped(false) },
      }),
    ).toBe(true);
    expect(
      shouldToastNotification({
        eventType: "ai_audit_export_ready",
        title: "Export ready",
        data: { exportId: "aiex_01", delivery: stamped(false) },
      }),
    ).toBe(true);
  });

  it("does not toast a notice without a title", () => {
    expect(
      shouldToastNotification({
        eventType: "report_run_completed",
        title: "",
        data: { runId: "rrun_01", delivery: stamped(true) },
      }),
    ).toBe(false);
  });
});

describe("run ids", () => {
  it("reads the run only from the event that delivers it", () => {
    expect(reportRunIdOf({ eventType: "report_run_completed", data: { runId: "rrun_01" } })).toBe(
      "rrun_01",
    );
    expect(reportRunIdOf({ eventType: "report_run_failed", data: { runId: "rrun_01" } })).toBe(
      null,
    );
    expect(
      billingTransferRunIdOf({
        eventType: "billing_transfer_run_failed",
        data: { runId: "btr_01" },
      }),
    ).toBe("btr_01");
    expect(billingTransferRunIdOf({ eventType: "watchtower.critical", data: { runId: "x" } })).toBe(
      null,
    );
    expect(reportRunIdOf({ eventType: "report_run_completed", data: { runId: "" } })).toBe(null);
    expect(reportRunIdOf({ eventType: "report_run_completed", data: { runId: 7 } })).toBe(null);
  });
});
