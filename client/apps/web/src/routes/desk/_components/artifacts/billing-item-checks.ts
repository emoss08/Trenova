import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { ShipmentBillingReadiness } from "@trenova/shared/types/shipment";

/** What one check asks of the person: nothing, a look, or a fix before approval. */
export type CheckState = "ok" | "warn" | "fail";

export type BillingCheck = {
  key: string;
  state: CheckState;
  title: string;
  /** The finding, in a line: who is assigned, what is missing, what was flagged. */
  detail: string;
  /** Why it was flagged, when there is more to say than the finding. */
  note?: string;
  /** "assign" offers to take the item; nothing else is fixed from the card. */
  action?: "assign";
};

/**
 * Everything that stands between a billing item and its approval, in the
 * order a biller works through it: who is on it, the paperwork, what the
 * readiness check found, detention waiting on approval, the charges, and the
 * customer. Each check is drawn from what the server already decided; the card
 * never re-derives a rule. A check that passes says what it found, so a clean
 * item reads as checked rather than empty.
 */
export function billingChecks(
  item: BillingQueueItem,
  readiness: ShipmentBillingReadiness | undefined,
  t: TranslateFn,
): BillingCheck[] {
  const checks: BillingCheck[] = [];

  const biller = item.assignedBiller?.name ?? "";
  checks.push(
    biller !== "" || item.assignedBillerId
      ? { key: "biller", state: "ok", title: t("Biller"), detail: biller || t("Assigned") }
      : {
          key: "biller",
          state: "fail",
          title: t("Biller"),
          detail: t("Nobody is assigned"),
          action: "assign",
        },
  );

  if (readiness) {
    const missing = readiness.missingRequirements.map((need) => need.documentTypeName);
    const have = readiness.requirements
      .filter((need) => need.satisfied)
      .map((need) => need.documentTypeName);
    if (missing.length > 0) {
      checks.push({
        key: "documents",
        state: "fail",
        title: t("Required documents"),
        detail: t("Missing {0}", missing.join(", ")),
      });
    } else if (have.length > 0) {
      checks.push({
        key: "documents",
        state: "ok",
        title: t("Required documents"),
        detail: have.join(" · "),
      });
    }

    readiness.validationFailures.forEach((failure, index) => {
      checks.push({
        key: `validation-${index}`,
        state: "fail",
        title: t("Shipment details"),
        detail: failure.message,
      });
    });
    readiness.warnings.forEach((warning, index) => {
      checks.push({
        key: `warning-${index}`,
        state: "warn",
        title: t("Worth a look"),
        detail: warning.message,
      });
    });
    if (readiness.serviceFailureContext.hasUnresolved) {
      checks.push({
        key: "service-failures",
        state: "warn",
        title: t("Service failures"),
        detail: t(
          "{0, plural, one {# service failure is unresolved} other {# service failures are unresolved}}",
          readiness.serviceFailureContext.unresolvedCount,
        ),
      });
    }
  }

  for (const hold of item.detentionHolds) {
    checks.push({
      key: `hold-${hold.occurrenceId}`,
      state: "fail",
      title: t("Detention waits on approval"),
      detail: hold.locationName
        ? t("{0} at {1}", money(hold.billableAmount, hold.currency), hold.locationName)
        : money(hold.billableAmount, hold.currency),
      note: t("The item can't be approved until this detention charge is approved or dropped."),
    });
  }

  const problem = item.payerShare?.resolutionError ?? "";
  if (problem !== "") {
    checks.push({
      key: "charges",
      state: "fail",
      title: t("Charges"),
      detail: t("The charges can't be divided between payers"),
      note: problem,
    });
  } else if (item.payerShare) {
    checks.push({
      key: "charges",
      state: "ok",
      title: t("Charges"),
      detail: t(
        "{0, plural, one {# charge} other {# charges}} · {1}",
        item.payerShare.lines.length,
        money(item.payerShare.totalAmount),
      ),
    });
  }

  const payer = readiness?.payers.find((candidate) => candidate.payerId === item.billToCustomerId);
  const customer = item.billToCustomer?.name ?? payer?.payerName ?? "";
  if (payer?.creditHold) {
    checks.push({
      key: "bill-to",
      state: "fail",
      title: t("Bill-to"),
      detail: t("{0} is on credit hold", customer),
    });
  } else if (customer !== "") {
    checks.push({
      key: "bill-to",
      state: "ok",
      title: t("Bill-to"),
      detail: payer?.creditStatus ? `${customer} · ${payer.creditStatus}` : customer,
    });
  }

  return checks;
}

/** Why Approve is not available yet, in a few words; empty when it is. */
export function approveBlocker(
  item: BillingQueueItem,
  checks: readonly BillingCheck[],
  readiness: ShipmentBillingReadiness | undefined,
  t: TranslateFn,
): string {
  if (item.status === "Approved" || item.status === "Posted") return t("Already approved");
  if (item.status === "Canceled") return t("This item was canceled");
  const failing = checks.find((check) => check.state === "fail");
  if (failing?.key === "biller") return t("Assign a biller first");
  if (failing?.key === "documents") return t("Add the missing documents first");
  if (failing?.key.startsWith("hold-")) return t("Approve the detention first");
  if (failing) return t("Resolve what's flagged first");
  if (readiness && !readiness.canMarkReadyToInvoice) return t("Billing requirements aren't met");
  if (item.status !== "InReview") return t("Start the review first");
  return "";
}

export function money(amount: string | number | null | undefined, currency = "USD"): string {
  const figure = Number(amount);
  if (amount === null || amount === undefined || amount === "" || !Number.isFinite(figure)) {
    return "—";
  }
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 2,
  }).format(figure);
}
