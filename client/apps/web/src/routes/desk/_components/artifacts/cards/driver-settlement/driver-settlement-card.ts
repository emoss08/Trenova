import {
  choiceLabel,
  payComponentKindChoices,
  settlementDisputeCategoryChoices,
} from "@/lib/choices";
import type {
  DeskDriverSettlement,
  DeskDriverSettlementDispute,
  DeskDriverSettlementGroup,
} from "@/lib/graphql/settlement-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ReferenceKind } from "@trenova/shared/types/generated/reference-views";
import type { StatementGroup, StatementRow } from "../../record-card/card-settlement";
import { cardDay, cardMoment, cardWeekday, money } from "../../record-card/record-card-values";
import { minor, periodLabel, signedMinor } from "../../record-card/settlement-values";

const DAY_SECONDS = 86_400;

/** The worker the settlement pays, by name. */
export function workerName(record: DeskDriverSettlement, t: TranslateFn): string {
  const worker = record.worker;
  if (!worker) return t("Unknown worker");
  return (
    worker.wholeName.trim() ||
    [worker.firstName, worker.lastName].filter(Boolean).join(" ") ||
    t("Unknown worker")
  );
}

/** What heads the card: the number, the period, and how the worker is paid. */
export function settlementKicker(record: DeskDriverSettlement, t: TranslateFn): string[] {
  return [
    record.settlementNumber,
    periodLabel(record.periodStart, record.periodEnd, t),
    record.classification === "OwnerOperator" ? t("Owner-operator") : t("Company"),
  ];
}

/** Under the worker's name: how many loads and miles it pays, and on which pay profile. */
export function settlementSubtitle(record: DeskDriverSettlement, t: TranslateFn): string {
  const miles = Number(record.totalMiles);
  return [
    record.shipmentCount > 0
      ? t("{0, plural, one {Pay for # load} other {Pay for # loads}}", record.shipmentCount)
      : t("No loads this period"),
    miles > 0 ? t("{0} mi", Math.round(miles).toLocaleString()) : "",
    record.payProfileName.trim(),
  ]
    .filter((part) => part !== "")
    .join(" · ");
}

/** The figure beside the name: the net pay, and what it means where the settlement stands. */
export function settlementFigure(
  record: DeskDriverSettlement,
  t: TranslateFn,
): { amount: string; caption: string; tone: "plain" | "zero" | "struck" } {
  const amount = minor(record.netPayMinor, record.currencyCode);
  if (record.status === "Voided") {
    return { amount, caption: t("net pay · voided"), tone: "struck" };
  }
  if (record.status === "Paid") {
    return { amount, caption: t("net pay · paid {0}", cardDay(record.paidAt)), tone: "plain" };
  }
  if (record.carryForwardOutMinor < 0) {
    return {
      amount,
      caption: t(
        "net pay · {0} carries forward",
        minor(-record.carryForwardOutMinor, record.currencyCode),
      ),
      tone: "zero",
    };
  }
  return {
    amount,
    caption: t("net pay · pays {0}", cardWeekday(record.payDate)),
    tone: record.netPayMinor === 0 ? "zero" : "plain",
  };
}

/** The facts beside the status pill: the flags to review and the disputes still open. */
export function headFacts(record: DeskDriverSettlement, t: TranslateFn): string[] {
  const facts: string[] = [];
  const flags = record.exceptions?.length ?? 0;
  if (flags > 0 && record.status !== "Paid" && record.status !== "Voided") {
    facts.push(t("{0, plural, one {# flag to review} other {# flags to review}}", flags));
  }
  const open = record.disputes.filter(isOpenDispute).length;
  if (open > 0) {
    facts.push(t("{0, plural, one {# open dispute} other {# open disputes}}", open));
  }
  if (record.status === "Paid" && record.paymentReference.trim()) {
    facts.push(t("{0} · {1}", record.paymentMethod, record.paymentReference.trim()));
  }
  return facts;
}

/** A flag the settlement run raised, titled by what it is and told in the server's words. */
export type SettlementFlag = {
  key: string;
  severity: "critical" | "warning";
  title: string;
  body: string;
};

function flagTitle(code: string, t: TranslateFn): string {
  switch (code) {
    case "NegativeNet":
      return t("Deductions ran past earnings");
    case "HighVariance":
      return t("Far from this worker's usual pay");
    case "NoActivity":
      return t("No pay this period");
    case "MissingPayProfile":
      return t("No pay profile");
    case "GuaranteeApplied":
      return t("The guaranteed minimum was paid");
    case "DeductionCapped":
      return t("A deduction was capped");
    case "EarningCapped":
      return t("Earnings were capped");
    case "ManualAdjustment":
      return t("Pay added by hand");
    default:
      return t("Review before approving");
  }
}

/** The flags to review, the critical ones first. */
export function settlementFlags(record: DeskDriverSettlement, t: TranslateFn): SettlementFlag[] {
  return (record.exceptions ?? [])
    .map((exception) => ({
      key: exception.code,
      severity: exception.severity === "Critical" ? ("critical" as const) : ("warning" as const),
      title: flagTitle(exception.code, t),
      body: exception.message,
    }))
    .sort((a, b) => Number(b.severity === "critical") - Number(a.severity === "critical"));
}

/** One step of the waterfall: where the running total stood and where it lands. */
export type FallStep = {
  key: "earnings" | "reimbursements" | "carriedIn" | "deductions" | "net" | "carriesForward";
  label: string;
  note: string;
  amount: number;
  from: number;
  to: number;
  sign: "in" | "out" | "total" | "carry";
};

/**
 * How the statement adds up, as the waterfall draws it: earnings, then what
 * was reimbursed and carried in, then the deductions taken off, landing on the
 * net pay; a shortfall carried to the next settlement hangs below zero.
 * Amounts are minor units; `from` and `to` are where each bar starts and ends.
 */
export function waterfall(record: DeskDriverSettlement, t: TranslateFn): FallStep[] {
  const steps: FallStep[] = [];
  let run = 0;
  const move = (
    key: FallStep["key"],
    label: string,
    note: string,
    amount: number,
    sign: FallStep["sign"],
  ) => {
    const from = run;
    run += amount;
    steps.push({
      key,
      label,
      note,
      amount,
      from: Math.min(from, run),
      to: Math.max(from, run),
      sign,
    });
  };
  move(
    "earnings",
    t("Earnings"),
    record.shipmentCount > 0
      ? t("{0, plural, one {# load} other {# loads}}", record.shipmentCount)
      : "",
    record.grossEarningsMinor,
    "in",
  );
  if (record.reimbursementsMinor !== 0) {
    move("reimbursements", t("Reimbursements"), "", record.reimbursementsMinor, "in");
  }
  if (record.carryForwardInMinor !== 0) {
    move(
      "carriedIn",
      t("Carried in"),
      t("from the last settlement"),
      record.carryForwardInMinor,
      record.carryForwardInMinor > 0 ? "in" : "out",
    );
  }
  if (record.deductionsMinor !== 0) {
    move("deductions", t("Deductions"), "", -record.deductionsMinor, "out");
  }
  steps.push({
    key: "net",
    label: t("Net pay"),
    note: "",
    amount: record.netPayMinor,
    from: 0,
    to: record.netPayMinor,
    sign: "total",
  });
  if (record.carryForwardOutMinor < 0) {
    steps.push({
      key: "carriesForward",
      label: t("Carries forward"),
      note: t("to the next settlement"),
      amount: record.carryForwardOutMinor,
      from: record.carryForwardOutMinor,
      to: 0,
      sign: "carry",
    });
  }
  return steps;
}

/** The waterfall's scale: the lowest and highest point any bar reaches, zero always within it. */
export function fallScale(steps: readonly FallStep[]): { low: number; high: number } {
  let low = 0;
  let high = 0;
  for (const step of steps) {
    low = Math.min(low, step.from);
    high = Math.max(high, step.to);
  }
  return { low, high: high === low ? low + 1 : high };
}

type Section = "earnings" | "reimbursements" | "carried" | "deductions";

function sectionOf(group: DeskDriverSettlementGroup): Section {
  switch (group.category) {
    case "Earning":
    case "GuaranteeTopUp":
      return "earnings";
    case "Reimbursement":
      return "reimbursements";
    case "CarryForward":
      return "carried";
    case "Adjustment":
      return group.amountMinor < 0 ? "deductions" : "earnings";
    default:
      return "deductions";
  }
}

function count(value: string): string {
  return Number(value).toLocaleString(undefined, { maximumFractionDigits: 2 });
}

/** How a row of pay was worked out: its quantity against its rate, in the method's own units. */
export function lineBasis(
  group: DeskDriverSettlementGroup,
  currency: string,
  t: TranslateFn,
): string {
  const rate = group.rate === null || group.rate === undefined ? null : money(group.rate, currency);
  const quantity = Number(group.quantity);
  const per = (unit: string) =>
    rate !== null && quantity > 0 ? t("{0} {1} × {2}", count(group.quantity), unit, rate) : "";
  switch (group.method) {
    case "PerLoadedMile":
    case "PerEmptyMile":
    case "PerTotalMile":
      return per(t("mi")) || t("{0} mi", count(group.quantity));
    case "PerStop":
      return rate !== null && quantity > 0
        ? t("{0, plural, one {# stop} other {# stops}} × {1}", quantity, rate)
        : t("{0, plural, one {# stop} other {# stops}}", quantity);
    case "PerHour":
      return per(t("hr")) || t("{0} hr", count(group.quantity));
    case "PerDay":
      return rate !== null && quantity > 0
        ? t("{0, plural, one {# day} other {# days}} × {1}", quantity, rate)
        : t("{0, plural, one {# day} other {# days}}", quantity);
    case "PercentOfRevenue":
      return group.rate ? t("{0}% of revenue", count(group.rate)) : t("Share of revenue");
    case "FlatPerShipment":
      return rate !== null ? t("{0} a load", rate) : t("Flat a load");
    case "PerEvent":
      return rate !== null && quantity > 0
        ? t("{0} × {1}", count(group.quantity), rate)
        : t("Flat");
    default:
      if (group.category === "AdvanceRecovery" && rate !== null) {
        return t("{0} a settlement", rate);
      }
      if (group.category === "Reimbursement") {
        return t("As receipted");
      }
      return t("Flat");
  }
}

function lineName(group: DeskDriverSettlementGroup, t: TranslateFn): string {
  if (group.componentKind && group.componentKind !== "Custom") {
    return choiceLabel(payComponentKindChoices, group.componentKind);
  }
  if (group.description.trim()) return group.description.trim();
  switch (group.category) {
    case "AdvanceRecovery":
      return t("Advance recovery");
    case "EscrowContribution":
      return t("Escrow");
    case "GuaranteeTopUp":
      return t("Guaranteed minimum");
    case "CarryForward":
      return t("Carried in");
    case "Reimbursement":
      return t("Reimbursement");
    case "Deduction":
      return t("Deduction");
    default:
      return t("Adjustment");
  }
}

/** The kind whose read a line's detail needs when the server withheld it. */
function lineWithheld(group: DeskDriverSettlementGroup): ReferenceKind | undefined {
  if (group.advance) {
    return group.advance.outstandingMinor === null ? "pay_advance" : undefined;
  }
  if (group.escrowAccount) {
    const { balanceMinor, targetAmountMinor } = group.escrowAccount;
    return balanceMinor === null || targetAmountMinor === null ? "escrow_account" : undefined;
  }
  return undefined;
}

function lineDetail(group: DeskDriverSettlementGroup, currency: string, t: TranslateFn): string {
  if (group.advance) {
    const reference = group.advance.reference.trim();
    const named = reference !== "" && !group.description.includes(reference);
    const outstanding = group.advance.outstandingMinor;
    if (outstanding === null) {
      return named ? t("Advance {0}", reference) : "";
    }
    return named
      ? t("Advance {0} · {1} still owed", reference, minor(outstanding, currency))
      : t("{0} still owed on the advance", minor(outstanding, currency));
  }
  if (group.escrowAccount) {
    const { balanceMinor, targetAmountMinor } = group.escrowAccount;
    if (balanceMinor === null || targetAmountMinor === null) {
      return "";
    }
    return t("{0} of {1} held", minor(balanceMinor, currency), minor(targetAmountMinor, currency));
  }
  if (group.shipmentCount > 0) {
    return t("{0, plural, one {# load} other {# loads}}", group.shipmentCount);
  }
  if (group.recurringDeductionId) {
    return t("Every settlement");
  }
  if (group.lineCount > 1) {
    return t("{0, plural, one {# line} other {# lines}}", group.lineCount);
  }
  return "";
}

/**
 * The statement as the card reads it: one row per way of paying, under
 * earnings, reimbursements, what was carried in and deductions, each group
 * with its total. A deduction reads as taken off.
 */
export function statementGroups(record: DeskDriverSettlement, t: TranslateFn): StatementGroup[] {
  const currency = record.currencyCode;
  const titles: Record<Section, string> = {
    earnings: t("Earnings"),
    reimbursements: t("Reimbursements"),
    carried: t("Carried in"),
    deductions: t("Deductions"),
  };
  const rows: Record<Section, StatementRow[]> = {
    earnings: [],
    reimbursements: [],
    carried: [],
    deductions: [],
  };
  const totals: Record<Section, number> = {
    earnings: 0,
    reimbursements: 0,
    carried: 0,
    deductions: 0,
  };
  for (const [index, group] of (record.lineSummary ?? []).entries()) {
    const section = sectionOf(group);
    totals[section] += group.amountMinor;
    rows[section].push({
      key: `${group.category}:${index}`,
      name: lineName(group, t),
      detail: lineDetail(group, currency, t),
      withheld: lineWithheld(group),
      basis: lineBasis(group, currency, t),
      amount: signedMinor(group.amountMinor, currency),
      manual: group.manual,
    });
  }
  return (["earnings", "reimbursements", "carried", "deductions"] as const)
    .filter((section) => rows[section].length > 0)
    .map((section) => ({
      key: section,
      title: titles[section],
      total: signedMinor(totals[section], currency),
      rows: rows[section],
    }));
}

/** How many lines the statement holds in all, for the section's aside. */
export function lineTotal(record: DeskDriverSettlement): number {
  return (record.lineSummary ?? []).reduce((sum, group) => sum + group.lineCount, 0);
}

function isOpenDispute(dispute: DeskDriverSettlementDispute): boolean {
  return dispute.status === "Open" || dispute.status === "InReview";
}

/** A dispute the worker raised, as the card tells it. */
export type DisputeCase = {
  id: string;
  open: boolean;
  title: string;
  status: string;
  body: string;
  meta: string;
};

/** The worker's disputes against the statement, the open ones first, with how long each has been open. */
export function disputeCases(
  record: DeskDriverSettlement,
  now: number,
  t: TranslateFn,
): DisputeCase[] {
  return [...record.disputes]
    .sort(
      (a, b) => Number(isOpenDispute(b)) - Number(isOpenDispute(a)) || b.createdAt - a.createdAt,
    )
    .map((dispute) => {
      const open = isOpenDispute(dispute);
      const line = dispute.settlementLine;
      const about = line
        ? [line.description.trim(), line.proNumber.trim()].filter(Boolean).join(" · ")
        : "";
      const category = choiceLabel(settlementDisputeCategoryChoices, dispute.category);
      return {
        id: dispute.id,
        open,
        title: about ? t("{0} · {1}", category, about) : category,
        status: disputeStatusLabel(dispute.status, t),
        body: open ? dispute.description : dispute.resolutionNote.trim() || dispute.description,
        meta: open
          ? raisedLabel(dispute.createdAt, now, t)
          : t("Closed {0}", cardDay(dispute.resolvedAt)),
      };
    });
}

/** When an open dispute was raised, with the whole days it has been open once there is one. */
function raisedLabel(createdAt: number, now: number, t: TranslateFn): string {
  const days = Math.floor((now - createdAt) / DAY_SECONDS);
  return days >= 1
    ? t(
        "Raised {0} · {1, plural, one {# day open} other {# days open}}",
        cardMoment(createdAt),
        days,
      )
    : t("Raised {0}", cardMoment(createdAt));
}

function disputeStatusLabel(status: string, t: TranslateFn): string {
  switch (status) {
    case "Open":
      return t("Open");
    case "InReview":
      return t("In review");
    case "Resolved":
      return t("Resolved");
    case "Denied":
      return t("Denied");
    default:
      return t("Withdrawn");
  }
}
