import type {
  DeskInvoice,
  DeskInvoiceChargeGroup,
  DeskInvoiceDispute,
  DeskInvoiceNeighbor,
} from "@/lib/graphql/invoice-card";
import {
  choiceLabel,
  invoiceDisputeReasonCodeChoices,
  invoiceDisputeResolutionChoices,
  paymentMethodChoices,
  paymentTermChoices,
} from "@/lib/choices";
import {
  accountingSyncNeedsAction,
  accountingSyncStateDetail,
  accountingSyncStateLabel,
} from "@/lib/accounting-sync";
import type { AccountingSyncRecord } from "@/lib/graphql/accounting-sync-ledger";
import type { AccountingSyncRecordStatus, RateUnit } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { invoiceVoidBlocker } from "@trenova/shared/types/invoice";
import type { CardStatusMark, CardTone } from "../../record-card/record-card";
import { cardDay, cardMoment, money } from "../../record-card/record-card-values";

const DAY_SECONDS = 86_400;
const MINUS = "−";

/** An amount in whole cents, so sums and comparisons never drift. */
function cents(amount: string | number | null | undefined): number {
  const figure = Number(amount);
  return Number.isFinite(figure) ? Math.round(figure * 100) : 0;
}

function centsMoney(value: number, currency: string): string {
  return money(value / 100, currency);
}

/** An amount taken off, as a bill's foot writes it: a true minus before the figure. */
function less(value: number, currency: string): string {
  return `${MINUS}${centsMoney(Math.abs(value), currency)}`;
}

function wholeDays(from: number, to: number): number {
  return Math.floor((to - from) / DAY_SECONDS);
}

function termLabel(record: DeskInvoice): string {
  return choiceLabel(paymentTermChoices, record.paymentTerm);
}

export type InvoiceKind = { kind: "invoice" | "credit" | "debit"; mark: "INV" | "CR" | "DR" };

/** Whether the record bills, credits or adds to a bill, as its slip names it. */
export function invoiceKind(record: Pick<DeskInvoice, "billType">): InvoiceKind {
  switch (record.billType) {
    case "CreditMemo":
      return { kind: "credit", mark: "CR" };
    case "DebitMemo":
      return { kind: "debit", mark: "DR" };
    case "Invoice":
      return { kind: "invoice", mark: "INV" };
  }
}

function isOpen(record: DeskInvoice): boolean {
  return record.status === "Posted" && cents(record.openBalance) > 0;
}

function openDispute(record: DeskInvoice): DeskInvoiceDispute | undefined {
  return record.disputes.find((dispute) => dispute.status === "Open");
}

/** The kicker over the title: the number, and what the invoice bills or corrects. */
export function invoiceKicker(record: DeskInvoice, t: TranslateFn): string[] {
  if (invoiceKind(record).kind === "credit") {
    return [
      record.number,
      record.referenceInvoice ? t("Corrects {0}", record.referenceInvoice.number) : "",
    ].filter(Boolean);
  }
  const period =
    record.scope === "Consolidated" && record.periodStart && record.periodEnd
      ? t("{0} – {1}", cardDay(record.periodStart), cardDay(record.periodEnd))
      : "";
  return [record.number, record.shipmentBol ? t("BOL {0}", record.shipmentBol) : "", period].filter(
    Boolean,
  );
}

/** What the invoice covers, and the day it is dated. */
export function invoiceSubtitle(record: DeskInvoice, t: TranslateFn): string {
  const dated = cardDay(record.invoiceDate);
  switch (invoiceKind(record).kind) {
    case "credit":
      return t("Credit memo · {0}", dated);
    case "debit":
      return t("Debit memo · {0}", dated);
    case "invoice":
      break;
  }
  switch (record.scope) {
    case "Consolidated":
      return t(
        "{0, plural, one {Consolidated invoice for # shipment} other {Consolidated invoice for # shipments}} · {1}",
        record.shipmentCount,
        dated,
      );
    case "Order":
      return record.orderNumber
        ? t("Invoice for order {0} · {1}", record.orderNumber, dated)
        : t("Invoice for an order · {0}", dated);
    case "Adjustment":
      return t("Corrected invoice · {0}", dated);
    case "Shipment":
    case "Memo":
      return t(
        "{0, plural, one {Invoice for # shipment} other {Invoice for # shipments}} · {1}",
        record.shipmentCount,
        dated,
      );
  }
}

export type InvoiceStanding = { label: string; tone: CardTone; mark: CardStatusMark };

/** Where the invoice stands, as its pill says it: what is late before what is disputed. */
export function invoiceStanding(record: DeskInvoice, t: TranslateFn): InvoiceStanding {
  if (record.status === "Voided") {
    return { label: t("Voided"), tone: "muted", mark: "dot" };
  }
  if (record.status === "Draft") {
    return { label: t("Draft"), tone: "draft", mark: "dot" };
  }
  if (invoiceKind(record).kind === "credit") {
    const left = cents(record.creditRemaining);
    if (left <= 0) {
      return { label: t("Fully applied"), tone: "success", mark: "check" };
    }
    return left >= Math.abs(cents(record.totalAmount))
      ? { label: t("Not applied yet"), tone: "progress", mark: "dot" }
      : { label: t("Partly applied"), tone: "progress", mark: "dot" };
  }
  if (record.settlementStatus === "Paid") {
    return { label: t("Paid"), tone: "success", mark: "check" };
  }
  if (record.daysPastDue && record.daysPastDue > 0) {
    return {
      label: t("{0, plural, one {# day past due} other {# days past due}}", record.daysPastDue),
      tone: "warn",
      mark: "dot",
    };
  }
  if (record.disputeStatus === "Disputed") {
    return { label: t("Disputed"), tone: "warn", mark: "dot" };
  }
  return record.settlementStatus === "PartiallyPaid"
    ? { label: t("Partly paid"), tone: "progress", mark: "dot" }
    : { label: t("Open"), tone: "progress", mark: "dot" };
}

export type InvoiceFigure = { amount: string; caption: string; tone: "owed" | "struck" | "plain" };

/** The figure beside the title: what is still owed, what a credit has left, or the total. */
export function invoiceFigure(record: DeskInvoice, t: TranslateFn): InvoiceFigure {
  const currency = record.currencyCode;
  const total = Math.abs(cents(record.totalAmount));
  if (record.status === "Voided") {
    return {
      amount: centsMoney(total, currency),
      caption: t("voided, nothing owed"),
      tone: "struck",
    };
  }
  if (record.status === "Draft") {
    return { amount: centsMoney(total, currency), caption: t("draft total"), tone: "plain" };
  }
  if (invoiceKind(record).kind === "credit") {
    return {
      amount: money(record.creditRemaining, currency),
      caption: t("credit left of {0}", centsMoney(total, currency)),
      tone: "plain",
    };
  }
  const open = cents(record.openBalance);
  if (record.settlementStatus === "Paid" || open <= 0) {
    return { amount: centsMoney(total, currency), caption: t("paid in full"), tone: "plain" };
  }
  const late = (record.daysPastDue ?? 0) > 0;
  return {
    amount: centsMoney(open, currency),
    caption:
      open >= total
        ? record.dueDate
          ? t("owed, due {0}", cardDay(record.dueDate))
          : t("owed")
        : t("still owed of {0}", centsMoney(total, currency)),
    tone: late ? "owed" : "plain",
  };
}

/** The facts set beside the pill under the title. */
export function headFacts(record: DeskInvoice, now: number, t: TranslateFn): string[] {
  if (record.status === "Voided") {
    const when = cardDay(record.voidedAt);
    return [
      record.voidedBy ? t("Voided {0} by {1}", when, record.voidedBy.name) : t("Voided {0}", when),
    ];
  }
  const facts: string[] = [];
  if (record.status === "Draft") {
    facts.push(
      t(
        "{0, plural, one {Unposted # day} other {Unposted # days}}",
        Math.max(0, wholeDays(record.invoiceDate, now)),
      ),
    );
  }
  if (record.dueDate && (record.status === "Draft" || isOpen(record))) {
    facts.push(t("Due {0}", cardDay(record.dueDate)));
  }
  const dispute = openDispute(record);
  if (dispute && isOpen(record)) {
    facts.push(
      dispute.disputedAmount === null
        ? t("In dispute")
        : t("{0} in dispute", money(dispute.disputedAmount, record.currencyCode)),
    );
  }
  if (record.isSplitBill) {
    facts.push(t("Split bill"));
  }
  if (record.factoringCompanyId) {
    facts.push(t("Factored"));
  }
  return facts;
}

/** What settled an invoice, in cents: payments still standing and credits still applied. */
function settled(record: DeskInvoice): { paid: number; credited: number } {
  const paid = record.paymentApplications
    .filter((application) => application.payment?.status !== "Reversed")
    .reduce((sum, application) => sum + application.appliedAmountMinor, 0);
  const credited = record.creditApplications
    .filter((application) => application.status === "Applied")
    .reduce((sum, application) => sum + application.appliedAmountMinor, 0);
  return { paid, credited };
}

export type OwedSegment = {
  key: "paid" | "credited" | "disputed" | "open";
  label: string;
  amount: string;
  /** The segment's part of the whole, 0 to 1. */
  share: number;
  zero: boolean;
};

/**
 * The total of an invoice as one strip, split into what was paid, credited,
 * is in dispute and is still open; a credit memo's into what was applied and
 * what is left.
 */
export function owedSegments(record: DeskInvoice, t: TranslateFn): OwedSegment[] {
  const currency = record.currencyCode;
  const total = Math.abs(cents(record.totalAmount));
  const segment = (key: OwedSegment["key"], label: string, value: number): OwedSegment => ({
    key,
    label,
    amount: centsMoney(value, currency),
    share: total > 0 ? value / total : 0,
    zero: value <= 0,
  });

  if (invoiceKind(record).kind === "credit") {
    const left = Math.max(0, cents(record.creditRemaining));
    return [
      segment("paid", t("Applied"), Math.max(0, total - left)),
      segment("open", t("Left to apply"), left),
    ];
  }

  const { paid, credited } = settled(record);
  const open = Math.max(0, cents(record.openBalance));
  const dispute = openDispute(record);
  if (dispute && dispute.disputedAmount === null) {
    return [
      segment("paid", t("Paid"), paid),
      segment("credited", t("Credited"), credited),
      segment("open", t("Open, part in dispute"), open),
    ];
  }
  const disputed = dispute ? Math.min(open, Math.max(0, cents(dispute.disputedAmount))) : 0;
  return [
    segment("paid", t("Paid"), paid),
    segment("credited", t("Credited"), credited),
    segment("disputed", t("In dispute"), disputed),
    segment("open", t("Open, not disputed"), open - disputed),
  ];
}

export type TermsRuler = {
  kind: "running" | "late" | "unposted";
  startLabel: string;
  startDay: string;
  /** Where today sits along the ruler, 0 to 1. */
  nowPosition: number;
  nowLabel: string;
  nowDay: string;
  /** Where the due date sits, when it falls inside the ruler. */
  duePosition: number | null;
  dueLabel: string;
  dueDay: string;
  endLabel: string;
  endDay: string;
  /** Days a draft has waited, from its invoice date. */
  waitedDays: number;
  /** Days the terms give, from the invoice date to the due date. */
  termDays: number;
};

/**
 * The terms as a ruler from the invoice date: to the due date with today on
 * the way, past it to today once late, or, for a draft, the days its terms
 * have already run while it waits to be posted. Nothing for an invoice that
 * owes nothing.
 */
export function termsRuler(record: DeskInvoice, now: number, t: TranslateFn): TermsRuler | null {
  if (
    !record.dueDate ||
    record.status === "Voided" ||
    invoiceKind(record).kind === "credit" ||
    (record.status === "Posted" && !isOpen(record))
  ) {
    return null;
  }
  const start = record.invoiceDate;
  const due = record.dueDate;
  const termDays = Math.max(1, wholeDays(start, due));
  const elapsed = Math.max(0, wholeDays(start, now));
  const term = termLabel(record);
  const today = t("Today, {0}", cardDay(now));
  const dueDay = t("{0} · {1}", cardDay(due), term);

  if (record.status === "Draft") {
    const span = Math.max(termDays, elapsed, 1);
    return {
      kind: "unposted",
      startLabel: t("Invoice date"),
      startDay: t("{0} · drafted", cardDay(start)),
      nowPosition: elapsed / span,
      nowLabel: t("{0, plural, one {# day unposted} other {# days unposted}}", elapsed),
      nowDay: today,
      duePosition: termDays / span,
      dueLabel: t("Due"),
      dueDay,
      endLabel: t("Due"),
      endDay: dueDay,
      waitedDays: elapsed,
      termDays,
    };
  }

  const pastDue = record.daysPastDue ?? 0;
  if (pastDue > 0) {
    const span = Math.max(elapsed, termDays, 1);
    return {
      kind: "late",
      startLabel: t("Invoiced"),
      startDay: cardDay(start),
      nowPosition: 1,
      nowLabel: t("{0, plural, one {# day past due} other {# days past due}}", pastDue),
      nowDay: today,
      duePosition: termDays / span,
      dueLabel: t("Due"),
      dueDay,
      endLabel: t("{0, plural, one {# day past due} other {# days past due}}", pastDue),
      endDay: today,
      waitedDays: 0,
      termDays,
    };
  }

  const left = Math.max(0, Math.ceil((due - now) / DAY_SECONDS));
  return {
    kind: "running",
    startLabel: t("Invoiced"),
    startDay: cardDay(start),
    nowPosition: Math.min(1, elapsed / termDays),
    nowLabel: t("Today"),
    nowDay: today,
    duePosition: 1,
    dueLabel: t("Due"),
    dueDay,
    endLabel:
      left === 0
        ? t("Due today")
        : t("{0, plural, one {Due in # day} other {Due in # days}}", left),
    endDay: dueDay,
    waitedDays: 0,
    termDays,
  };
}

/** What a draft's waiting has cost: how many of the customer's days have run before it was posted. */
export function waitingNote(
  ruler: Pick<TermsRuler, "waitedDays" | "termDays">,
  t: TranslateFn,
): string {
  return ruler.waitedDays === 1
    ? t(
        "Its terms count from the invoice date, so 1 of its {0} days has run while it waits to be posted.",
        ruler.termDays,
      )
    : t(
        "Its terms count from the invoice date, so {0} of its {1} days have run while it waits to be posted.",
        ruler.waitedDays,
        ruler.termDays,
      );
}

export type ChargeBasis =
  | { kind: "formula"; templateId: string | null; name: string }
  | { kind: "text"; text: string };

export type ChargeRow = {
  key: string;
  name: string;
  code: string;
  detail: string;
  basis: ChargeBasis;
  amount: string;
};

function quantityOf(group: DeskInvoiceChargeGroup): number {
  const figure = Number(group.quantity);
  return Number.isFinite(figure) ? figure : 0;
}

function unitsText(unit: RateUnit | null, quantity: number, t: TranslateFn): string {
  switch (unit) {
    case "Hour":
      return t("{0, plural, one {# hour} other {# hours}}", quantity);
    case "Mile":
      return t("{0, plural, one {# mile} other {# miles}}", quantity);
    case "Day":
      return t("{0, plural, one {# day} other {# days}}", quantity);
    case "Stop":
      return t("{0, plural, one {# stop} other {# stops}}", quantity);
    case null:
      return t("{0, plural, one {# unit} other {# units}}", quantity);
  }
}

function chargeBasis(group: DeskInvoiceChargeGroup, currency: string, t: TranslateFn): ChargeBasis {
  if (group.type === "Freight") {
    return group.formulaTemplateName
      ? { kind: "formula", templateId: group.formulaTemplateId, name: group.formulaTemplateName }
      : { kind: "text", text: t("No formula template") };
  }
  const quantity = quantityOf(group);
  switch (group.chargeMethod) {
    case "Percentage":
      if (!group.rate) return { kind: "text", text: t("Percentage of linehaul") };
      return {
        kind: "text",
        text: group.rateBasisAmount
          ? t("{0}% of {1}", Number(group.rate), money(group.rateBasisAmount, currency))
          : t("{0}% of linehaul", Number(group.rate)),
      };
    case "PerUnit": {
      const units = unitsText(group.rateUnit, quantity, t);
      return {
        kind: "text",
        text: group.rate ? t("{0} × {1}", units, money(group.rate, currency)) : units,
      };
    }
    case "Flat":
      return {
        kind: "text",
        text:
          quantity > 1 && group.rate
            ? t("{0} × {1}", quantity, money(group.rate, currency))
            : t("Flat"),
      };
    case null:
      return { kind: "text", text: "" };
  }
}

/**
 * The charges as the bill reads them: one row for each way a charge was
 * billed, the template that rated the freight or how an accessorial was
 * charged, and, on an invoice for several shipments, how many share it.
 */
export function chargeRows(record: DeskInvoice, t: TranslateFn): ChargeRow[] {
  const currency = record.currencyCode;
  const several = record.shipmentCount > 1;
  return record.chargeSummary.map((group, index) => {
    const parts = [
      several && group.shipmentCount > 0
        ? t("{0, plural, one {# shipment} other {# shipments}}", group.shipmentCount)
        : "",
      group.type === "Freight" && group.rate ? t("{0} base rate", money(group.rate, currency)) : "",
    ].filter(Boolean);
    return {
      key: `${group.type}-${group.chargeCode ?? ""}-${group.formulaTemplateId ?? group.formulaTemplateName ?? ""}-${index}`,
      name: group.type === "Freight" ? t("Linehaul") : group.description,
      code: group.chargeCode ?? "",
      detail: parts.length === 2 ? t("{0} · {1}", parts[0], parts[1]) : (parts[0] ?? ""),
      basis: chargeBasis(group, currency, t),
      amount: money(group.amount, currency),
    };
  });
}

export type FootRow = { label: string; value: string; kind: "line" | "total" | "less" | "due" };

/** The foot of the bill: its parts, its total, then what took it down to the balance due. */
export function chargeFoot(record: DeskInvoice, t: TranslateFn): FootRow[] {
  const currency = record.currencyCode;
  const total = cents(record.totalAmount);
  if (invoiceKind(record).kind === "credit") {
    return [{ label: t("Credit"), value: less(total, currency), kind: "total" }];
  }

  const rows: FootRow[] = [];
  const freight = cents(record.subtotalAmount);
  const other = cents(record.otherAmount);
  if (freight !== 0) {
    rows.push({ label: t("Freight"), value: centsMoney(freight, currency), kind: "line" });
  }
  if (other !== 0) {
    const memo = record.chargeSummary.some((group) => group.type === "Memo");
    rows.push({
      label: memo ? t("Other charges") : t("Accessorials"),
      value: centsMoney(other, currency),
      kind: "line",
    });
  }
  rows.push({ label: t("Total"), value: centsMoney(total, currency), kind: "total" });

  if (record.status === "Draft") {
    return rows;
  }
  if (record.status === "Voided") {
    rows.push(
      { label: t("Reversed"), value: less(total, currency), kind: "less" },
      { label: t("Balance due"), value: centsMoney(0, currency), kind: "due" },
    );
    return rows;
  }
  const { paid, credited } = settled(record);
  if (paid > 0) {
    rows.push({ label: t("Paid"), value: less(paid, currency), kind: "less" });
  }
  if (credited > 0) {
    rows.push({ label: t("Credited"), value: less(credited, currency), kind: "less" });
  }
  rows.push({
    label: t("Balance due"),
    value: money(record.openBalance, currency),
    kind: "due",
  });
  return rows;
}

export type RemitRow = {
  key: string;
  kind: "payment" | "credit";
  title: string;
  tag: string;
  line: string;
  amount: string;
  /** A payment reversed or a credit taken back, kept and struck through. */
  undone: boolean;
  /** The record the row opens, when it is one the Desk draws. */
  open: { entity: "invoice"; id: string } | null;
  at: number;
};

/** What settled an invoice, newest first: payments as remittance stubs, credit memos applied. */
export function remittance(record: DeskInvoice, t: TranslateFn): RemitRow[] {
  const currency = record.currencyCode;
  const payments = record.paymentApplications.map((application): RemitRow => {
    const paid = application.payment;
    const reversed = paid?.status === "Reversed";
    const method = paid ? choiceLabel(paymentMethodChoices, paid.paymentMethod) : t("Payment");
    const reference = paid?.referenceNumber.trim();
    const at = paid?.paymentDate ?? application.createdAt;
    const received = t("Received {0}", cardDay(at));
    return {
      key: application.id,
      kind: "payment",
      title: reference ? t("{0} · ref {1}", method, reference) : method,
      tag:
        application.shortPayAmountMinor > 0
          ? t("Short {0}", centsMoney(application.shortPayAmountMinor, currency))
          : "",
      line: reversed ? t("{0} · reversed", received) : received,
      amount: less(application.appliedAmountMinor, currency),
      undone: reversed,
      open: null,
      at,
    };
  });
  const credits = record.creditApplications.map((application): RemitRow => ({
    key: application.id,
    kind: "credit",
    title: application.creditMemo
      ? t("Credit memo {0}", application.creditMemo.number)
      : t("Credit memo"),
    tag: "",
    line: appliedLine(application, t),
    amount: less(application.appliedAmountMinor, currency),
    undone: application.status === "Unapplied",
    open: application.creditMemo ? { entity: "invoice", id: application.creditMemo.id } : null,
    at: application.accountingDate,
  }));
  return [...payments, ...credits].sort((a, b) => b.at - a.at);
}

function appliedLine(application: DeskInvoice["creditApplications"][number], t: TranslateFn) {
  const applied = cardDay(application.accountingDate);
  if (application.status !== "Unapplied") {
    return t("Applied {0}", applied);
  }
  const back = t("Applied {0}, taken back {1}", applied, cardDay(application.unappliedAt));
  const reason = application.unappliedReason.trim();
  return reason ? t("{0} · {1}", back, reason) : back;
}

/** The invoices a credit memo settled, a use taken back kept and struck through. */
export function creditUses(record: DeskInvoice, t: TranslateFn): RemitRow[] {
  const currency = record.currencyCode;
  return record.creditApplications.map((application) => ({
    key: application.id,
    kind: "credit",
    title: application.invoice ? t("Invoice {0}", application.invoice.number) : t("An invoice"),
    tag: "",
    line: appliedLine(application, t),
    amount: centsMoney(application.appliedAmountMinor, currency),
    undone: application.status === "Unapplied",
    open: application.invoice ? { entity: "invoice", id: application.invoice.id } : null,
    at: application.accountingDate,
  }));
}

/**
 * A dispute's amount and notes are outside its reference view: for someone who may read the
 * invoice but not disputes, `amount` is null (drawn as withheld) and the notes are left out.
 */
export type OpenCase = {
  reason: string;
  amount: string | null;
  notes: string;
  opened: string;
  age: string;
};

export type ClosedCase = { key: string; title: string; amount: string | null; line: string };

/** The open dispute apart from the ones already closed, each with its outcome. */
export function disputeCases(
  record: DeskInvoice,
  now: number,
  t: TranslateFn,
): { open: OpenCase | null; closed: ClosedCase[] } {
  const currency = record.currencyCode;
  const reason = (dispute: DeskInvoiceDispute) =>
    choiceLabel(invoiceDisputeReasonCodeChoices, dispute.reasonCode);
  const current = openDispute(record);
  const open: OpenCase | null = current
    ? {
        reason: reason(current),
        amount: current.disputedAmount === null ? null : money(current.disputedAmount, currency),
        notes: current.notes?.trim() ?? "",
        opened: current.openedBy
          ? t("Opened by {0} · {1}", current.openedBy.name, cardMoment(current.openedAt))
          : t("Opened {0}", cardMoment(current.openedAt)),
        age: t(
          "{0, plural, one {# day open} other {# days open}}",
          Math.max(0, wholeDays(current.openedAt, now)),
        ),
      }
    : null;
  const closed = record.disputes
    .filter((dispute) => dispute.status !== "Open")
    .map((dispute): ClosedCase => {
      const outcome =
        dispute.status === "Withdrawn" || !dispute.resolution
          ? t("Withdrawn")
          : choiceLabel(invoiceDisputeResolutionChoices, dispute.resolution);
      const day = cardDay(dispute.resolvedAt);
      const who = dispute.resolvedBy
        ? dispute.status === "Withdrawn"
          ? t("Closed by {0} · {1}", dispute.resolvedBy.name, day)
          : t("Resolved by {0} · {1}", dispute.resolvedBy.name, day)
        : dispute.status === "Withdrawn"
          ? t("Closed {0}", day)
          : t("Resolved {0}", day);
      const notes = dispute.resolutionNotes?.trim() ?? "";
      return {
        key: dispute.id,
        title: t("{0} · {1}", reason(dispute), outcome),
        amount: dispute.disputedAmount === null ? null : money(dispute.disputedAmount, currency),
        line: notes ? t("{0} · {1}", who, notes) : who,
      };
    });
  return { open, closed };
}

/**
 * Why the invoice cannot be voided now, or null when it can: a voided one is
 * already gone, and cash or credit still applied to it must be taken off
 * first. A reversed payment and a credit taken back no longer count.
 */
export function voidBlocker(record: DeskInvoice, t: TranslateFn): string | null {
  const blocker = invoiceVoidBlocker(
    { status: record.status, appliedAmount: Number(record.appliedAmount) },
    {
      paymentApplications: record.paymentApplications.filter(
        (application) => application.payment?.status !== "Reversed",
      ).length,
      creditApplications: record.creditApplications.filter(
        (application) => application.status !== "Unapplied",
      ).length,
    },
  );
  return blocker === null ? null : t(blocker);
}

export type VoidNote = { reason: string; freight: string; when: string };

/** Why the invoice was voided, what became of its freight, and when and by whom. */
export function voidNote(record: DeskInvoice, t: TranslateFn): VoidNote | null {
  if (record.status !== "Voided") {
    return null;
  }
  const freight =
    record.voidDisposition === "Rebill"
      ? t("Back in the billing queue to bill again")
      : record.voidDisposition === "DoNotRebill"
        ? t("Its shipments settled without another bill")
        : "";
  const when = cardMoment(record.voidedAt);
  return {
    reason: record.voidReason.trim(),
    freight,
    when: record.voidedBy ? t("{0} by {1}", when, record.voidedBy.name) : when,
  };
}

export type ChannelTone = "ok" | "warn" | "idle" | "progress";

export type ChannelReading = {
  state: string;
  tone: ChannelTone;
  /** One line under the state, cut off to fit and told in full on hover. */
  detail: string;
  /** Whether the line says why it did not go. */
  why: boolean;
};

export type EmailAttemptReading = {
  toRecipients: readonly string[];
  sentAt: number | null | undefined;
  error: string | null | undefined;
};

export type EmailPlanReading = { to: readonly string[]; errors: readonly string[] };

/**
 * How the invoice went out by email: to whom and when from its last attempt,
 * why it failed, or, before it was sent, who it goes to or why it cannot go.
 */
export function emailChannel(
  record: DeskInvoice,
  read: { attempt: EmailAttemptReading | null; plan: EmailPlanReading | null },
  t: TranslateFn,
): ChannelReading {
  const { attempt, plan } = read;
  const recipients = attempt?.toRecipients.join(", ") ?? "";
  const error = attempt?.error?.trim() ?? "";
  switch (record.sendStatus) {
    case "Sent": {
      const when = cardMoment(attempt?.sentAt ?? record.sentAt);
      return {
        state: t("Sent"),
        tone: "ok",
        detail: recipients ? t("{0} to {1}", when, recipients) : when,
        why: false,
      };
    }
    case "PartiallySent":
      return { state: t("Partly sent"), tone: "warn", detail: error || recipients, why: !!error };
    case "Failed":
      return { state: t("Failed"), tone: "warn", detail: error, why: !!error };
    case "Sending":
      return {
        state: t("Sending"),
        tone: "progress",
        detail: recipients ? t("To {0}", recipients) : "",
        why: false,
      };
    case "NotSent":
      break;
  }
  if (record.status !== "Voided" && plan) {
    if (plan.errors.length > 0) {
      return { state: t("Cannot send"), tone: "warn", detail: plan.errors.join(" "), why: true };
    }
    if (plan.to.length > 0) {
      return {
        state: t("Not sent"),
        tone: "idle",
        detail: t("Goes to {0} when sent", plan.to.join(", ")),
        why: false,
      };
    }
  }
  return { state: t("Not sent"), tone: "idle", detail: "", why: false };
}

export type EdiReading = ChannelReading & { messageId: string | null };

/** How the invoice's EDI 210 went: sent where, sending, failed with why, or why it cannot go. */
export function ediChannel(record: DeskInvoice, t: TranslateFn): EdiReading {
  const plan = record.ediSendPlan;
  const partner = plan.partnerName.trim() || record.customer?.name || record.billToName;
  const messageId = plan.lastMessageId ?? record.lastEdiMessageId;
  const error = plan.lastError.trim() || record.lastEdiError.trim();
  if (!plan.enabled) {
    return {
      state: t("Not set up"),
      tone: "idle",
      detail: t("{0} does not take invoices by EDI", record.customer?.name ?? record.billToName),
      why: false,
      messageId: null,
    };
  }
  switch (plan.status) {
    case "Sent": {
      const when = cardMoment(plan.sentAt ?? record.ediSentAt);
      return {
        state: t("Sent"),
        tone: "ok",
        detail: plan.communicationMethod
          ? t("{0} to {1} by {2}", when, partner, plan.communicationMethod)
          : t("{0} to {1}", when, partner),
        why: false,
        messageId,
      };
    }
    case "Queued":
    case "Generated":
    case "Sending":
      return {
        state: t("Sending"),
        tone: "progress",
        detail: t("To {0}", partner),
        why: false,
        messageId,
      };
    case "Failed":
    case "DeadLettered":
    case "NotConfigured":
    case "NotSent":
      break;
  }
  if (plan.blockers.length > 0) {
    return {
      state: t("Cannot send"),
      tone: "warn",
      detail: plan.blockers.join(" "),
      why: true,
      messageId: null,
    };
  }
  switch (plan.status) {
    case "Failed":
      return { state: t("Failed"), tone: "warn", detail: error, why: !!error, messageId };
    case "DeadLettered":
      return { state: t("Not delivered"), tone: "warn", detail: error, why: !!error, messageId };
    case "NotConfigured":
      return {
        state: t("Not set up"),
        tone: "idle",
        detail: t("No 210 is set up for {0}", partner),
        why: false,
        messageId: null,
      };
    default:
      return {
        state: t("Not sent"),
        tone: "idle",
        detail: record.status === "Draft" ? t("Goes to {0} when posted", partner) : "",
        why: false,
        messageId: null,
      };
  }
}

export type BooksState = {
  providerName: string;
  record: Pick<
    AccountingSyncRecord,
    | "id"
    | "status"
    | "syncedAt"
    | "nextAttemptAt"
    | "resolution"
    | "errorMessage"
    | "attemptCount"
    | "skippedReason"
    | "externalDocNumber"
    | "externalUrl"
  >;
};

export type BooksReading = ChannelReading & {
  provider: string;
  externalUrl: string | null;
  recordId: string;
  needsAction: boolean;
};

const BOOKS_TONE: Record<AccountingSyncRecordStatus, ChannelTone> = {
  Synced: "ok",
  Queued: "progress",
  AwaitingApproval: "progress",
  InFlight: "progress",
  Retrying: "progress",
  Blocked: "warn",
  DeadLettered: "warn",
  Skipped: "idle",
  Superseded: "idle",
};

/** Where the invoice stands in the accounting system, named for whichever one the organization keeps. */
export function booksChannel(state: BooksState, t: TranslateFn): BooksReading {
  const record = state.record;
  const provider = state.providerName;
  const tone = BOOKS_TONE[record.status];
  const detail =
    record.status === "Synced"
      ? record.externalDocNumber
        ? t("As {0} · synced {1}", record.externalDocNumber, cardDay(record.syncedAt))
        : record.syncedAt
          ? t("Synced {0}", cardDay(record.syncedAt))
          : ""
      : accountingSyncStateDetail(state, t);
  return {
    state: accountingSyncStateLabel(record.status, provider, t),
    tone,
    detail,
    why: tone === "warn" && detail !== "",
    provider,
    externalUrl: record.status === "Synced" && record.externalUrl ? record.externalUrl : null,
    recordId: record.id,
    needsAction: accountingSyncNeedsAction(record.status),
  };
}

/** How many of what settled the invoice still stand, and how many were reversed or taken back. */
export function settledSummary(rows: readonly RemitRow[], credit: boolean, t: TranslateFn): string {
  const undone = rows.filter((row) => row.undone).length;
  const applied = t("{0} applied", rows.length - undone);
  if (undone === 0) {
    return applied;
  }
  return credit
    ? t("{0} · {1} taken back", applied, undone)
    : t("{0} · {1} reversed", applied, undone);
}

/** Another invoice the card names, as a row that opens it in the card's place. */
export function neighborRow(neighbor: DeskInvoiceNeighbor, t: TranslateFn): RemitRow {
  const name = (() => {
    switch (invoiceKind(neighbor).kind) {
      case "credit":
        return t("Credit memo {0}", neighbor.number);
      case "debit":
        return t("Debit memo {0}", neighbor.number);
      case "invoice":
        return t("Invoice {0}", neighbor.number);
    }
  })();
  const standing =
    neighbor.status === "Voided"
      ? t("Voided")
      : neighbor.status === "Draft"
        ? t("Draft")
        : cents(neighbor.openBalance) <= 0
          ? t("Settled")
          : neighbor.daysPastDue
            ? t("{0, plural, one {# day past due} other {# days past due}}", neighbor.daysPastDue)
            : t("{0} still owed", money(neighbor.openBalance, neighbor.currencyCode));
  return {
    key: neighbor.id,
    kind: "credit",
    title: name,
    tag: "",
    line: neighbor.billToName ? t("{0} · {1}", neighbor.billToName, standing) : standing,
    amount: centsMoney(Math.abs(cents(neighbor.totalAmount)), neighbor.currencyCode),
    undone: neighbor.status === "Voided",
    open: { entity: "invoice", id: neighbor.id },
    at: neighbor.postedAt ?? 0,
  };
}
