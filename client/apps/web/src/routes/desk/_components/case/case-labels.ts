import { recordPath } from "@/config/record-links";
import { turnTime } from "@/components/desk-chat/turn-time";
import { builtinStepLabel } from "@/lib/case-checklist-labels";
import { caseStateOf } from "@/lib/case-state";
import { invoicePanelPath } from "@/lib/invoice-links";
import type { CaseChecklist, CaseRecord, CaseSummary, ChecklistItem } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** The record a case is about as a sentence names it: "shipment 10293", "invoice INV-88". */
export function caseRecordPhrase(record: CaseRecord, t: TranslateFn): string {
  switch (record.type) {
    case "Shipment":
      return t("shipment {0}", record.label);
    case "Invoice":
      return t("invoice {0}", record.label);
    case "InvoiceDispute":
      return t("the dispute on invoice {0}", record.label);
  }
}

/** The record a case is about as a heading names it. */
export function caseRecordTitle(record: CaseRecord, t: TranslateFn): string {
  if (record.status === "Missing") {
    return t("The record this case was about is gone");
  }
  switch (record.type) {
    case "Shipment":
      return t("Shipment {0}", record.label);
    case "Invoice":
      return t("Invoice {0}", record.label);
    case "InvoiceDispute":
      return t("Dispute on invoice {0}", record.label);
  }
}

/** Where the case's record opens, or null when it is gone. */
export function caseRecordPath(record: CaseRecord): string | null {
  if (record.status === "Missing") {
    return null;
  }
  switch (record.type) {
    case "Shipment":
      return recordPath("shipment", record.id);
    case "Invoice":
      return invoicePanelPath(record.id);
    case "InvoiceDispute":
      return record.invoiceId ? invoicePanelPath(record.invoiceId, "disputes") : null;
  }
}

function closedLabel(closedAs: string, t: TranslateFn): string {
  switch (closedAs) {
    case "Invoiced":
      return t("Invoiced");
    case "Canceled":
      return t("Canceled");
    case "Paid":
      return t("Paid");
    case "Voided":
      return t("Voided");
    case "Withdrawn":
    case "CustomerWithdrew":
      return t("Withdrawn");
    case "CreditIssued":
      return t("Credit issued");
    case "InvoiceUpheld":
      return t("Invoice upheld");
    case "Rebilled":
      return t("Rebilled");
    case "WrittenOff":
      return t("Written off");
    default:
      return t("Resolved");
  }
}

/** The case's state as its chip says it. */
export function caseStateLabel(
  summary: CaseSummary,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  switch (caseStateOf(summary, now)) {
    case "Settled":
      return t("Settled · {0}", closedLabel(summary.record.closedAs, t));
    case "Snoozed": {
      const when = turnTime(summary.snoozedUntil ?? now, timezone, t);
      if (summary.snoozeAnchor === "Appointment") {
        return t("Snoozed to the appointment · {0}", when);
      }
      if (summary.snoozeAnchor === "ETA") {
        return t("Snoozed to the ETA · {0}", when);
      }
      return t("Snoozed until {0}", when);
    }
    case "Waiting":
      switch (summary.waitingOn) {
        case "Carrier":
          return t("Waiting on the carrier");
        case "Customer":
          return t("Waiting on the customer");
        case "Reply":
          return t("Waiting on a reply");
        default:
          return t("Waiting");
      }
    case "Working":
      return t("Working");
  }
}

/** The short form the rail shows beside a conversation. */
export function caseRailLabel(
  summary: CaseSummary,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  switch (caseStateOf(summary, now)) {
    case "Settled":
      return t("Settled");
    case "Snoozed":
      return turnTime(summary.snoozedUntil ?? now, timezone, t);
    case "Waiting":
      return summary.waitingOn === "Carrier"
        ? t("Carrier")
        : summary.waitingOn === "Customer"
          ? t("Customer")
          : t("Waiting");
    case "Working":
      return "";
  }
}

export function checklistTitle(checklist: CaseChecklist, t: TranslateFn): string {
  return checklist.kind === "ReadyToBill" ? t("Ready to bill") : t("Ready to close");
}

export function checklistItemLabel(item: ChecklistItem, t: TranslateFn): string {
  return item.label || builtinStepLabel(item.key, t);
}

function codeLabel(code: string, t: TranslateFn): string {
  switch (code) {
    case "missing_bol":
      return t("No BOL number");
    case "credit_hold":
      return t("Customer on credit hold");
    case "unresolved_service_failures":
      return t("Unresolved service failures");
    case "rate_missing_basis":
      return t("No rated basis");
    case "rate_variance_requires_action":
      return t("Rate differs from the contract");
    case "rate_departure_reason_missing":
      return t("No reason for the off-contract rate");
    case "AccessorialNotOnRateCon":
      return t("An accessorial isn't on the rate con");
    case "ChargeOverRateCon":
      return t("A charge is over the rate con");
    case "NotSent":
      return t("Not sent");
    case "Failed":
      return t("Sending failed");
    case "PartiallySent":
      return t("Sent to some recipients");
    case "Unpaid":
      return t("Unpaid");
    case "PartiallyPaid":
      return t("Partly paid");
    default:
      return code;
  }
}

/** What stands behind an item, in a line under it; empty when there is nothing to add. */
export function checklistItemDetail(item: ChecklistItem, timezone: string, t: TranslateFn): string {
  const when = item.at ? turnTime(item.at, timezone, t) : "";
  switch (item.state) {
    case "NotNeeded":
      return t("Not required for this customer");
    case "Done":
      return item.tickedBy && when
        ? t("Ticked by {0} · {1}", item.tickedBy, when)
        : item.tickedBy
          ? t("Ticked by {0}", item.tickedBy)
          : when;
    case "Pending":
      switch (item.key) {
        case "delivered":
          return t("Not delivered yet");
        case "pod":
        case "customerNotified":
          return t("After delivery");
        case "accessorials":
          return t(
            "{0, plural, one {# detention clock still running} other {# detention clocks still running}}",
            item.count,
          );
        case "paid":
          return when ? t("Due {0}", when) : codeLabels(item, t);
        default:
          return codeLabels(item, t);
      }
    case "Blocked":
      switch (item.key) {
        case "accessorials":
          return t(
            "{0, plural, one {# charge waits on approval} other {# charges wait on approval}}",
            item.count,
          );
        case "paid":
          return when ? t("Past due since {0}", when) : codeLabels(item, t);
        case "customerNotified":
          return t("Not told since delivery");
        case "carrierRateConfirmed":
          return t("Not confirmed by {0}", item.names.join(", "));
        default:
          return [...item.names, ...item.codes.map((code) => codeLabel(code, t))].join(" · ");
      }
  }
}

function codeLabels(item: ChecklistItem, t: TranslateFn): string {
  return item.codes.map((code) => codeLabel(code, t)).join(" · ");
}

/** The checklist item a step belongs to, when the step is one the organization added. */
function customStepItem(
  step: string,
  checklist: CaseChecklist | null | undefined,
): ChecklistItem | undefined {
  return step.startsWith(CUSTOM_PREFIX)
    ? checklist?.items.find((candidate) => candidate.key === step)
    : undefined;
}

const CUSTOM_PREFIX = "custom:";

/** The next step as its button says it. */
export function stepLabel(
  step: string,
  t: TranslateFn,
  checklist?: CaseChecklist | null,
): string {
  const custom = customStepItem(step, checklist);
  if (custom) {
    return custom.stepLabel || custom.label;
  }
  switch (step) {
    case "track_delivery":
      return t("Check on delivery");
    case "request_pod":
      return t("Request POD");
    case "request_paperwork":
      return t("Request paperwork");
    case "review_rate":
      return t("Review the rate");
    case "confirm_rate":
      return t("Chase the rate confirmation");
    case "approve_accessorials":
      return t("Review accessorials");
    case "notify_customer":
      return t("Notify the customer");
    case "clear_holds":
      return t("Clear the holds");
    case "mark_ready":
      return t("Mark ready to invoice");
    case "send_invoice":
      return t("Send invoice");
    case "post_invoice":
      return t("Post the invoice");
    case "work_dispute":
      return t("Work the dispute");
    case "follow_up_payment":
      return t("Follow up on payment");
    default:
      return step;
  }
}

/**
 * What the next step asks the case's agent, as the person: the record named,
 * and what the checklist knows about the item, in the person's own words.
 */
export function stepPrompt(
  step: string,
  record: CaseRecord,
  checklist: CaseChecklist | null | undefined,
  t: TranslateFn,
): string {
  const what = caseRecordPhrase(record, t);
  const custom = customStepItem(step, checklist);
  if (custom) {
    return custom.prompt
      ? t("For {0}: {1}", what, custom.prompt)
      : t("Take care of “{0}” for {1}.", custom.label, what);
  }
  const item = checklist?.items.find((candidate) => candidate.step === step);
  const names = item?.names.join(", ") ?? "";

  switch (step) {
    case "track_delivery":
      return t("Where is {0} now, and when will it deliver?", what);
    case "request_pod":
      return t("Request the proof of delivery for {0} and tell me when it is in.", what);
    case "request_paperwork":
      return names
        ? t("Request the missing paperwork for {0}: {1}.", what, names)
        : t("Request the missing paperwork for {0}.", what);
    case "review_rate":
      return t(
        "Check the charges on {0} against the rate confirmation and tell me what does not match.",
        what,
      );
    case "confirm_rate":
      return names
        ? t("Get the rate confirmation for {0} confirmed by {1}.", what, names)
        : t("Get the rate confirmation for {0} confirmed.", what);
    case "approve_accessorials":
      return t("Walk me through the accessorials on {0} that wait on approval.", what);
    case "notify_customer":
      return t("Let the customer know {0} has delivered.", what);
    case "clear_holds":
      return t("What is holding the bill for {0}, and how do I clear it?", what);
    case "mark_ready":
      return t("Mark {0} ready to invoice.", what);
    case "send_invoice":
      return record.type === "Shipment"
        ? t("Send the invoice for {0}.", what)
        : t("Send {0} to the customer.", what);
    case "post_invoice":
      return t("Post {0}.", what);
    case "work_dispute":
      return t(
        "Work the open dispute on {0}: what is the customer disputing, and what should we do?",
        what,
      );
    case "follow_up_payment":
      return t("Invoice {0} is past due. Follow up with the customer on payment.", record.label);
    default:
      return stepLabel(step, t, checklist);
  }
}

/**
 * Where a person takes a step themselves when no agent they may use can:
 * the record's own page, or the desk where that kind of work is done.
 */
export function stepPage(step: string, record: CaseRecord): string | null {
  const invoiceId = record.type === "InvoiceDispute" ? record.invoiceId : record.id;
  switch (step) {
    case "review_rate":
    case "mark_ready":
      return "/billing/queue";
    case "approve_accessorials":
      return "/detention/desk";
    case "work_dispute":
      return invoiceId ? invoicePanelPath(invoiceId, "disputes") : null;
    case "send_invoice":
    case "post_invoice":
    case "follow_up_payment":
      return record.type === "Shipment" ? "/billing/queue" : invoiceId ? invoicePanelPath(invoiceId) : null;
    default:
      return caseRecordPath(record);
  }
}

/** What the link to that page says. */
export function stepPageLabel(step: string, record: CaseRecord, t: TranslateFn): string {
  switch (stepPage(step, record)) {
    case "/billing/queue":
      return t("Open the billing queue");
    case "/detention/desk":
      return t("Open the detention desk");
    default:
      return record.type === "Shipment" ? t("Open the shipment") : t("Open the invoice");
  }
}
