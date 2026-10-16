import { aiAuditExportNotice } from "@/lib/ai-audit-exports";
import { deliveryDecision } from "@/lib/desktop-notifications";

type ToastCandidate = {
  eventType?: string | null;
  title?: string;
  data?: Record<string, unknown> | null;
};

const REPORT_RUN_COMPLETED_EVENT = "report_run_completed";
const BILLING_TRANSFER_RUN_EVENT_PREFIX = "billing_transfer_run_";

function runIdOf(notification: ToastCandidate): string | null {
  const runId = notification.data?.runId;
  return typeof runId === "string" && runId !== "" ? runId : null;
}

/** The finished report a notice hands back to whoever ran it. */
export function reportRunIdOf(notification: ToastCandidate): string | null {
  return notification.eventType === REPORT_RUN_COMPLETED_EVENT ? runIdOf(notification) : null;
}

/**
 * The bulk billing transfer a notice reports on. It runs unattended, so its result has to
 * come back to whoever started it with a way into the report.
 */
export function billingTransferRunIdOf(notification: ToastCandidate): string | null {
  return notification.eventType?.startsWith(BILLING_TRANSFER_RUN_EVENT_PREFIX)
    ? runIdOf(notification)
    : null;
}

/**
 * Every notification lands in the bell; only two kinds also interrupt with a toast: what
 * the server stamped urgent, and the result of something the person asked for (a report
 * run, a billing transfer run, an AI audit export). A watchtower item, a tender update or
 * a reminder stays in the bell, so a weather warning no longer toasts every dispatcher.
 */
export function shouldToastNotification(notification: ToastCandidate): boolean {
  if (!notification.title) {
    return false;
  }
  return (
    deliveryDecision(notification).urgent ||
    reportRunIdOf(notification) !== null ||
    billingTransferRunIdOf(notification) !== null ||
    aiAuditExportNotice(notification) !== null
  );
}
