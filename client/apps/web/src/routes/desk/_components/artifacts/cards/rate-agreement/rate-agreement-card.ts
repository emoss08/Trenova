import {
  accessorialChargeMethodChoices,
  choiceLabel,
  rateAgreementStatusChoices,
  rateRoundingModeChoices,
  rateUnitChoices,
} from "@/lib/choices";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatNumber } from "@trenova/shared/i18n/format";
import type {
  RateAgreement,
  RateAgreementAccessorial,
  RateAgreementFuelBinding,
  RateAgreementRule,
  RateAgreementRuleBreak,
} from "@trenova/shared/types/rate";
import type { CardTone } from "../../record-card/record-card";
import { money } from "../../record-card/record-card-values";

const DAY_SECONDS = 86_400;
/** Rates are quoted to the cent of a cent: "2.1025" a mile. */
const RATE_DIGITS = 4;

/** A decimal the API sends as a string, or null when it is not there or not a number. */
export function decimal(value: string | number | null | undefined): number | null {
  if (value === null || value === undefined || value === "") return null;
  const figure = Number(value);
  return Number.isFinite(figure) ? figure : null;
}

function figure(value: number, digits = RATE_DIGITS): string {
  return formatNumber(value, { maximumFractionDigits: digits });
}

/** Where an agreement stands in its life, as the card's head says it. */
export type AgreementStage = { tone: CardTone; mark: "dot" | "check" | "pause"; label: string };

/**
 * Only an active agreement prices freight. One in review is moving toward it,
 * a draft has not started, a suspended one is held, and an expired or archived
 * one is done.
 */
export function agreementStage(status: RateAgreement["status"]): AgreementStage {
  const label = choiceLabel(rateAgreementStatusChoices, status);
  switch (status) {
    case "Active":
      return { tone: "success", mark: "check", label };
    case "InReview":
      return { tone: "progress", mark: "dot", label };
    case "Suspended":
      return { tone: "warn", mark: "pause", label };
    case "Expired":
      return { tone: "muted", mark: "dot", label };
    case "Archived":
      return { tone: "muted", mark: "pause", label };
    default:
      return { tone: "draft", mark: "dot", label };
  }
}

/**
 * How an agreement's term reads on its panel: pricing freight, inside the
 * renewal notice that needs someone, not started or not yet approved, held,
 * or over. The panel takes its colour from it.
 */
export type TermState = "live" | "notice" | "pending" | "held" | "ended";

export function termState(
  agreement: Pick<
    RateAgreement,
    "status" | "effectiveFrom" | "effectiveTo" | "autoRenew" | "renewalNoticeDays"
  >,
  now: number,
): TermState {
  switch (agreement.status) {
    case "Suspended":
      return "held";
    case "Expired":
    case "Archived":
      return "ended";
    case "Active":
      break;
    default:
      return "pending";
  }
  if (agreement.effectiveFrom > now) return "pending";
  const end = agreement.effectiveTo;
  if (end === null || end === undefined) return "live";
  if (end < now) return "ended";
  const daysLeft = Math.ceil((end - now) / DAY_SECONDS);
  return !agreement.autoRenew && daysLeft <= agreement.renewalNoticeDays ? "notice" : "live";
}

/** Where an agreement's term stands on its bar: today's place and the renewal notice's share. */
export type TermSpan = {
  /** How far through the term today is, from 0 (its first day) to 1 (its last). */
  today: number;
  /** The share of the term at its end taken by the renewal notice. */
  notice: number;
  /** Whether today falls before the term, inside it, or after it. */
  when: "before" | "during" | "after";
};

/** An agreement's term laid along a bar, or null for one with no end to lay it to. */
export function termSpan(
  agreement: Pick<RateAgreement, "effectiveFrom" | "effectiveTo" | "renewalNoticeDays">,
  now: number,
): TermSpan | null {
  const end = agreement.effectiveTo;
  if (end === null || end === undefined || end <= agreement.effectiveFrom) return null;
  const length = end - agreement.effectiveFrom;
  const share = (now - agreement.effectiveFrom) / length;
  return {
    today: Math.min(1, Math.max(0, share)),
    notice: Math.min(1, Math.max(0, (agreement.renewalNoticeDays * DAY_SECONDS) / length)),
    when: share < 0 ? "before" : share > 1 ? "after" : "during",
  };
}

/** One step of an agreement's way to pricing freight: drafted, submitted, approved, and how it ended. */
export type LifecycleStep = {
  key: "drafted" | "submitted" | "approved" | "ended";
  label: string;
  /** When it happened, or null for a step not yet taken. */
  at: number | null;
  /** Who took it, by user id, when the agreement records that. */
  byId: string | null;
  state: "done" | "now" | "todo";
  /** How an ended step reads: a hold that can lift, or a close that cannot. */
  tone: "held" | "closed" | null;
};

/**
 * The approval path an agreement has walked. A draft is where it is; one in
 * review waits on approval; an active one has taken every step; a suspended,
 * expired or archived one adds how it ended.
 */
export function lifecycleSteps(
  agreement: Pick<
    RateAgreement,
    "status" | "createdAt" | "submittedAt" | "submittedById" | "approvedAt" | "approvedById"
  >,
  t: TranslateFn,
): LifecycleStep[] {
  const submitted = agreement.submittedAt ?? null;
  const approved = agreement.approvedAt ?? null;
  const status = agreement.status;
  const steps: LifecycleStep[] = [
    {
      key: "drafted",
      label: t("Drafted"),
      at: agreement.createdAt ?? null,
      byId: null,
      state: status === "Draft" ? "now" : "done",
      tone: null,
    },
    {
      key: "submitted",
      label: t("Submitted for review"),
      at: submitted,
      byId: agreement.submittedById ?? null,
      state: status === "InReview" ? "now" : submitted !== null ? "done" : "todo",
      tone: null,
    },
    {
      key: "approved",
      label: t("Approved"),
      at: approved,
      byId: agreement.approvedById ?? null,
      state: approved !== null ? "done" : "todo",
      tone: null,
    },
  ];
  if (status === "Suspended" || status === "Expired" || status === "Archived") {
    steps.push({
      key: "ended",
      label: choiceLabel(rateAgreementStatusChoices, status),
      at: null,
      byId: null,
      state: "now",
      tone: status === "Suspended" ? "held" : "closed",
    });
  }
  return steps;
}

/** How a lane is priced: by a formula template at a rate, or by a rate table's cells. */
export type LanePricing =
  | { by: "formula"; templateId: string; rate: string | null }
  | { by: "matrix"; matrixId: string }
  | { by: "none" };

export function lanePricing(rule: RateAgreementRule): LanePricing {
  if (rule.rateMatrixId) return { by: "matrix", matrixId: rule.rateMatrixId };
  if (rule.formulaTemplateId) {
    const rate = decimal(rule.rate);
    return {
      by: "formula",
      templateId: rule.formulaTemplateId,
      rate: rate === null ? null : figure(rate),
    };
  }
  return { by: "none" };
}

/** A percentage the API sends as a whole number of percent, "12.5" for 12.5%. */
function percent(value: number): string {
  return formatNumber(value / 100, { style: "percent", maximumFractionDigits: 2 });
}

/** A range of weight or distance, open at either end. */
function span(low: number | null, high: number | null, unit: "lb" | "mi", t: TranslateFn): string {
  const from = low === null ? "" : figure(low, 0);
  const to = high === null ? "" : figure(high, 0);
  if (unit === "lb") {
    if (from && to) return t("{0}–{1} lb", from, to);
    if (from) return t("Over {0} lb", from);
    return to ? t("Up to {0} lb", to) : "";
  }
  if (from && to) return t("{0}–{1} mi", from, to);
  if (from) return t("Over {0} mi", from);
  return to ? t("Up to {0} mi", to) : "";
}

/**
 * What narrows or bounds a lane's price beyond its places: its charge limits,
 * discount, the loads it applies to, and which way it runs.
 */
export function laneTerms(rule: RateAgreementRule, currency: string, t: TranslateFn): string[] {
  const minCharge = decimal(rule.minCharge);
  const maxCharge = decimal(rule.maxCharge);
  const floor = decimal(rule.absoluteMinCharge);
  const discount = decimal(rule.discountPercent);
  const billable = decimal(rule.minBillableDistance);
  const breaks = rule.breaks.length;
  return [
    minCharge !== null ? t("Min {0}", money(minCharge, currency)) : "",
    maxCharge !== null ? t("Max {0}", money(maxCharge, currency)) : "",
    floor !== null ? t("Never under {0}", money(floor, currency)) : "",
    discount !== null && discount !== 0 ? t("{0} discount", percent(discount)) : "",
    billable !== null ? t("Bills at least {0} mi", figure(billable, 0)) : "",
    breaks > 0 ? t("{0, plural, one {# weight break} other {# weight breaks}}", breaks) : "",
    span(decimal(rule.minWeight), decimal(rule.maxWeight), "lb", t),
    span(decimal(rule.minDistance), decimal(rule.maxDistance), "mi", t),
    rule.hazmatOnly ? t("Hazmat only") : "",
    rule.tempControlOnly ? t("Temperature controlled only") : "",
    rule.direction === "Bidirectional" ? t("Both ways") : "",
  ].filter((term) => term !== "");
}

/** A weight break as its lane lists it: its own label, or the weights it covers. */
export function breakLabel(weightBreak: RateAgreementRuleBreak, t: TranslateFn): string {
  if (weightBreak.label.trim() !== "") return weightBreak.label.trim();
  return span(decimal(weightBreak.fromWeight), decimal(weightBreak.toWeight), "lb", t);
}

/** An accessorial's terms as a biller reads them: what it charges, and what changes that. */
export function accessorialTerms(
  accessorial: RateAgreementAccessorial,
  currency: string,
  t: TranslateFn,
): string[] {
  if (accessorial.waived) return [t("Waived")];
  const amount = decimal(accessorial.amount);
  const cap = decimal(accessorial.maxAmount);
  const charge =
    amount === null
      ? choiceLabel(accessorialChargeMethodChoices, accessorial.method)
      : accessorial.method === "Percentage"
        ? percent(amount)
        : accessorial.method === "PerUnit" && accessorial.rateUnit
          ? t(
              "{0} per {1}",
              money(amount, currency),
              choiceLabel(rateUnitChoices, accessorial.rateUnit).toLowerCase(),
            )
          : money(amount, currency);
  return [
    charge,
    accessorial.freeUnits ? t("{0} free", figure(accessorial.freeUnits, 0)) : "",
    cap !== null ? t("Up to {0}", money(cap, currency)) : "",
    accessorial.autoApply ? t("Added automatically") : "",
  ].filter((term) => term !== "");
}

/** What the agreement does to its fuel program's surcharge. */
export function fuelTerms(
  binding: RateAgreementFuelBinding,
  currency: string,
  t: TranslateFn,
): string[] {
  if (binding.waived) return [t("Fuel surcharge waived")];
  const peg = decimal(binding.pegPriceOverride);
  const increment = decimal(binding.incrementRateOverride);
  const cap = decimal(binding.capAmount);
  const terms = [
    peg !== null ? t("Peg {0}", money(peg, currency)) : "",
    increment !== null ? t("{0} per increment", money(increment, currency)) : "",
    cap !== null ? t("Capped at {0}", money(cap, currency)) : "",
  ].filter((term) => term !== "");
  return terms.length > 0 ? terms : [t("The program's own terms")];
}

/** One of the limits the agreement holds every price inside. */
export type Guardrail = { key: string; label: string; value: string };

/**
 * The agreement's default charge limits and rounding, and the margin it keeps:
 * a floor on margin for what it sells, a cap on pay for what it buys.
 */
export function guardrails(agreement: RateAgreement, t: TranslateFn): Guardrail[] {
  const minCharge = decimal(agreement.defaultMinCharge);
  const maxCharge = decimal(agreement.defaultMaxCharge);
  const marginFloor = decimal(agreement.marginFloorPercent);
  const payCap = decimal(agreement.maxPayPercentOfSell);
  const rows: Guardrail[] = [
    {
      key: "min",
      label: t("Minimum charge"),
      value: minCharge === null ? t("None") : money(minCharge, agreement.currency),
    },
    {
      key: "max",
      label: t("Maximum charge"),
      value: maxCharge === null ? t("None") : money(maxCharge, agreement.currency),
    },
  ];
  if (marginFloor !== null) {
    rows.push({
      key: "margin",
      label: t("Margin floor"),
      value: percent(marginFloor),
    });
  }
  if (payCap !== null) {
    rows.push({
      key: "pay",
      label: t("Pays at most"),
      value: t("{0} of sell", percent(payCap)),
    });
  }
  rows.push({
    key: "rounding",
    label: t("Rounding"),
    value: t(
      "{0} · {1, plural, one {# place} other {# places}}",
      choiceLabel(rateRoundingModeChoices, agreement.roundingMode),
      agreement.roundingPrecision,
    ),
  });
  return rows;
}

/** Every id the card names by its record, per kind, sorted so the lookup's key holds still. */
export type AgreementReferences = {
  templates: string[];
  matrices: string[];
  charges: string[];
  programs: string[];
  customers: string[];
  carriers: string[];
  users: string[];
};

function sortedIds(ids: Iterable<string | null | undefined>): string[] {
  return [...new Set([...ids].filter((id): id is string => Boolean(id)))].sort();
}

export function agreementReferences(
  agreement: RateAgreement,
  lanes: readonly RateAgreementRule[],
  versionAuthors: readonly (string | null | undefined)[],
): AgreementReferences {
  return {
    templates: sortedIds([
      ...lanes.map((lane) => lane.formulaTemplateId),
      ...agreement.accessorials.map((accessorial) => accessorial.formulaTemplateId),
    ]),
    matrices: sortedIds(lanes.map((lane) => lane.rateMatrixId)),
    charges: sortedIds(
      agreement.accessorials.map((accessorial) => accessorial.accessorialChargeId),
    ),
    programs: sortedIds([agreement.fuelBinding?.fuelSurchargeProgramId]),
    customers: sortedIds([agreement.customerId, agreement.billToCustomerId]),
    carriers: sortedIds([agreement.carrierId]),
    users: sortedIds([agreement.submittedById, agreement.approvedById, ...versionAuthors]),
  };
}
