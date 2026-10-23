import { choiceLabel, detentionWaiverReasonChoices, stopTypeChoices } from "@/lib/choices";
import {
  deliveryStatusLabel,
  evidenceKindLabel,
  evidenceSourceLabel,
  noticeKindLabel,
} from "@/lib/detention-labels";
import type {
  DeskDetention,
  DeskDetentionEvidence,
  DeskDetentionNotice,
  DeskDetentionOccurrence,
} from "@/lib/graphql/detention-card";
import { formatList } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import type { CardTrackStep } from "../../record-card/card-track";
import type { CardStatusMark, CardTone } from "../../record-card/record-card";
import {
  cardDay,
  cardMoment,
  cardMomentAfter,
  cardTime,
  cityLine,
  money,
} from "../../record-card/record-card-values";

type Occurrence = DeskDetentionOccurrence;
type Status = Occurrence["status"];

const SECONDS = 60;

/** The facility the truck waited at, by name. */
export function placeName(occurrence: Occurrence, t: TranslateFn): string {
  return occurrence.locationName.trim() || t("Unknown location");
}

function stopTypeLabel(stopType: string): string {
  return choiceLabel(stopTypeChoices, stopType as never);
}

/** What heads the card: the load, the kind of stop, and the policy that priced it. */
export function detentionKicker(occurrence: Occurrence, t: TranslateFn): string[] {
  const kicker = [occurrence.shipmentProNumber.trim(), stopTypeLabel(occurrence.stopType)];
  const policy = occurrence.snapshot;
  if (policy) {
    kicker.push(
      policy.policyVersion
        ? t("{0} v{1}", policy.policyName, policy.policyVersion)
        : policy.policyName,
    );
  }
  return kicker.filter((part) => part !== "");
}

/** Under the facility: the customer and the appointment, when the stop had one. */
export function detentionSubtitle(occurrence: Occurrence, t: TranslateFn): string {
  const appointment = appointmentLabel(occurrence, t);
  return [occurrence.customerName.trim(), appointment].filter(Boolean).join(" · ");
}

function appointmentLabel(occurrence: Occurrence, t: TranslateFn): string {
  const { appointmentStart: start, appointmentEnd: end } = occurrence;
  if (!start) return "";
  if (end && end > start) {
    return t("appointment {0} – {1}", cardMoment(start), cardMomentAfter(start, end));
  }
  return t("appointment {0}", cardMoment(start));
}

/** Where the stop is, "Forest Park, GA", or nothing when the location could not be read. */
export function placeLine(occurrence: Occurrence): string {
  return occurrence.location ? cityLine(occurrence.location) : "";
}

/**
 * The rate the policy bills at, "$75.00/hr": the accessorial's rate, or the
 * one tier a single-tier policy has; nothing for a policy of several tiers,
 * whose rate the receipt gives step by step.
 */
function rateLabel(occurrence: Occurrence, t: TranslateFn): string {
  const policy = occurrence.snapshot;
  if (!policy) return "";
  const tiers = policy.tiers ?? [];
  const priced =
    policy.rateSource === "Accessorial"
      ? { rate: policy.flatRate, unit: policy.flatRateUnit }
      : tiers.length === 1
        ? { rate: tiers[0].rate, unit: tiers[0].rateUnit }
        : null;
  if (!priced || priced.rate <= 0) return "";
  const rate = money(priced.rate, occurrence.currency);
  switch (priced.unit) {
    case "Hour":
      return t("{0}/hr", rate);
    case "Day":
      return t("{0}/day", rate);
    case "Flat":
      return "";
  }
}

/**
 * The amount the card leads with and what it means where the charge stands: a
 * running clock's price as of now, the charge to bill, or a waived or
 * unbillable figure struck through.
 */
export function detentionFigure(
  detail: DeskDetention,
  now: number,
  t: TranslateFn,
): { amount: string; caption: string; struck: boolean; quiet: boolean } {
  const { occurrence } = detail;
  const currency = occurrence.currency;
  const billed = money(occurrence.billableAmount, currency);
  switch (occurrence.status) {
    case "Accruing": {
      const live = detail.currentlyBillable;
      const amount = money(live?.billableAmount ?? occurrence.billableAmount, currency);
      return now >= occurrence.freeTimeExpiresAt
        ? {
            amount,
            caption: t("so far · billing since {0}", cardTime(occurrence.freeTimeExpiresAt)),
            struck: false,
            quiet: false,
          }
        : {
            amount,
            caption: t("free time ends {0}", cardTime(occurrence.freeTimeExpiresAt)),
            struck: false,
            quiet: true,
          };
    }
    case "Waived":
      return {
        amount: money(
          Number(occurrence.waivedAmount) || Number(occurrence.billableAmount),
          currency,
        ),
        caption: t("waived · {0}", cardDay(occurrence.waivedAt)),
        struck: true,
        quiet: false,
      };
    case "NotBillable":
      return {
        amount: billed,
        caption: occurrence.suppressedByGate
          ? t("not billable · notice not sent")
          : t("not billable"),
        struck: Number(occurrence.billableAmount) > 0,
        quiet: true,
      };
    case "Disputed":
      return {
        amount: billed,
        caption: t("in dispute since {0}", cardDay(occurrence.disputedAt)),
        struck: false,
        quiet: false,
      };
    case "Billed":
      return { amount: billed, caption: t("billed"), struck: false, quiet: false };
    case "Pending":
    case "Approved": {
      const rate = rateLabel(occurrence, t);
      const minutes = formatDetentionMinutes(occurrence.roundedMinutes);
      return {
        amount: billed,
        caption: rate ? t("to bill · {0} at {1}", minutes, rate) : t("to bill · {0}", minutes),
        struck: false,
        quiet: false,
      };
    }
  }
}

const STEPS = ["Accruing", "Pending", "Approved", "Billed"] as const;

/** The charge's way from the dock to the invoice; a waiver, dispute or gate takes it off the way. */
export function detentionTrack(status: Status, t: TranslateFn): CardTrackStep[] {
  const labels: Record<(typeof STEPS)[number], string> = {
    Accruing: t("At the dock"),
    Pending: t("Waiting on approval"),
    Approved: t("Approved"),
    Billed: t("Billed"),
  };
  const at = (STEPS as readonly string[]).indexOf(status);
  return STEPS.map((key, index) => ({
    key,
    label: labels[key],
    state: at < 0 ? "off" : index < at ? "done" : index === at ? "now" : "next",
  }));
}

/** Whether the charge has left the way to the invoice: waived, disputed or not billable. */
export function isOffTrack(status: Status): boolean {
  return status === "Waived" || status === "Disputed" || status === "NotBillable";
}

/** Where the charge stands, as the pill at the card's head says it. */
export function detentionStanding(
  status: Status,
  t: TranslateFn,
): { label: string; tone: CardTone; mark: CardStatusMark } {
  switch (status) {
    case "Accruing":
      return { label: t("At the dock"), tone: "progress", mark: "dot" };
    case "Pending":
      return { label: t("Waiting on approval"), tone: "warn", mark: "dot" };
    case "Approved":
      return { label: t("Approved"), tone: "progress", mark: "dot" };
    case "Billed":
      return { label: t("Billed"), tone: "success", mark: "check" };
    case "Disputed":
      return { label: t("Disputed"), tone: "warn", mark: "dot" };
    case "Waived":
      return { label: t("Waived"), tone: "muted", mark: "x" };
    case "NotBillable":
      return { label: t("Not billable"), tone: "muted", mark: "x" };
  }
}

/** The line under the track: what happened last and what comes next. */
export function detentionTrackNote(occurrence: Occurrence, now: number, t: TranslateFn): string {
  const pro = occurrence.shipmentProNumber;
  switch (occurrence.status) {
    case "Accruing":
      return t(
        "On the clock {0} · it settles when the truck departs",
        formatDetentionMinutes(minutesBetween(occurrence.clockStartAt, now)),
      );
    case "Pending":
      return t(
        "Departed {0} · approve it to post the charge to {1}",
        cardMoment(occurrence.departedAt ?? occurrence.clockStopAt),
        pro,
      );
    case "Approved":
      return occurrence.approvedBy
        ? t(
            "Approved by {0} {1} · it posts to {2} with the next billing run",
            occurrence.approvedBy.name,
            cardDay(occurrence.approvedAt),
            pro,
          )
        : t("Approved automatically · it posts to {0} with the next billing run", pro);
    case "Billed":
      return t("Posted to {0} as a detention charge", pro);
    case "Waived":
      return occurrence.approvedAt
        ? t("Waived after it was approved · nothing was posted to {0}", pro)
        : t("Waived before approval · nothing was posted to {0}", pro);
    case "Disputed":
      return t(
        "The customer disputed it {0} · it stays off the invoice until it is settled",
        cardDay(occurrence.disputedAt),
      );
    case "NotBillable":
      return occurrence.suppressedByGate
        ? t("The required notice was not sent, so the policy held the charge back")
        : t("No billable time after free time and rounding");
  }
}

/** The facts beside the pill: what to look at before acting. */
export function detentionFacts(detail: DeskDetention, now: number, t: TranslateFn): string[] {
  const { occurrence } = detail;
  const facts: string[] = [];
  const threshold = occurrence.snapshot?.requireApprovalOverAmount;
  if (occurrence.status === "Pending" && threshold) {
    facts.push(t("Over the {0} approval line", money(threshold, occurrence.currency)));
  }
  const notice = noticeReading(occurrence, now);
  if (notice === "overdue") facts.push(t("Notice overdue"));
  if (notice === "due") facts.push(t("Notice due {0}", cardTime(occurrence.noticeDueAt)));
  if (notice === "sent") facts.push(t("Notice sent in window"));
  if (notice === "late") facts.push(t("Notice sent late"));
  if (occurrence.arrivedLate && occurrence.lateByMinutes > 0) {
    facts.push(t("Arrived {0} late", formatDetentionMinutes(occurrence.lateByMinutes)));
  }
  if (occurrence.convertedToLayover) facts.push(t("Turned into layover"));
  return facts.slice(0, 2);
}

/** How the notice to the customer stands: none needed, due, overdue, sent in its window, late or failed. */
export type NoticeReading = "none" | "due" | "overdue" | "sent" | "late" | "failed";

export function noticeReading(occurrence: Occurrence, now: number): NoticeReading {
  switch (occurrence.notificationStatus) {
    case "NotRequired":
      return "none";
    case "Sent":
      return "sent";
    case "Late":
      return "late";
    case "Failed":
      return "failed";
    case "Missed":
      return "overdue";
    case "Pending":
      return occurrence.noticeDueAt && now > occurrence.noticeDueAt ? "overdue" : "due";
  }
}

function minutesBetween(from: number, to: number): number {
  return Math.max(0, Math.round((to - from) / SECONDS));
}

/** One stretch of the ruler: before the clock, free time, or billable. */
export type ClockSpan = { kind: "pre" | "free" | "bill"; from: number; to: number };
/** A moment under the ruler, set on the lower row when it would crowd the one before. */
export type ClockTick = { key: string; label: string; time: string; at: number; low: boolean };
/** The notice's moment above the ruler: sent, or still due. */
export type ClockFlag = { label: string; at: number; tone: "ok" | "warn" };

export type ClockModel = {
  from: number;
  to: number;
  live: boolean;
  spans: ClockSpan[];
  ticks: ClockTick[];
  flag: ClockFlag | null;
  dwellMinutes: number;
  freeMinutes: number;
  billable: { minutes: number; rounded: number };
};

/** How close two moments may sit, as a share of the ruler, before the second drops a row. */
const CROWDED = 0.16;

/**
 * The stop's clock as the ruler draws it: from the first moment that matters
 * (the appointment or the arrival) to the departure, or to now while the truck
 * is still there; the time before the clock starts, free time, and billable
 * time; each moment labelled beneath, every other crowded one a row lower; and
 * the notice above it.
 */
export function clockModel(detail: DeskDetention, now: number, t: TranslateFn): ClockModel {
  const { occurrence } = detail;
  const live = occurrence.isOpen && occurrence.status === "Accruing";
  const starts = [
    occurrence.appointmentStart,
    occurrence.arrivedAt,
    occurrence.clockStartAt,
  ].filter((value): value is number => typeof value === "number" && value > 0);
  const from = Math.min(...starts);
  const end = live ? now : (occurrence.departedAt ?? occurrence.clockStopAt ?? now);
  const to = Math.max(end, from + SECONDS);
  const freeEnd = Math.min(occurrence.freeTimeExpiresAt, to);

  const spans: ClockSpan[] = [];
  if (occurrence.clockStartAt > from) {
    spans.push({ kind: "pre", from, to: Math.min(occurrence.clockStartAt, to) });
  }
  if (freeEnd > occurrence.clockStartAt) {
    spans.push({ kind: "free", from: occurrence.clockStartAt, to: freeEnd });
  }
  if (to > occurrence.freeTimeExpiresAt) {
    spans.push({ kind: "bill", from: occurrence.freeTimeExpiresAt, to });
  }

  const moments: Omit<ClockTick, "low">[] = [];
  if (occurrence.appointmentStart && occurrence.appointmentStart !== occurrence.arrivedAt) {
    moments.push({
      key: "appointment",
      label: t("Appointment"),
      time: cardTime(occurrence.appointmentStart),
      at: occurrence.appointmentStart,
    });
  }
  if (occurrence.arrivedAt) {
    moments.push({
      key: "arrived",
      label: t("Arrived"),
      time: cardTime(occurrence.arrivedAt),
      at: occurrence.arrivedAt,
    });
  }
  if (!starts.slice(0, -1).includes(occurrence.clockStartAt)) {
    moments.push({
      key: "clock",
      label: t("Clock starts"),
      time: cardTime(occurrence.clockStartAt),
      at: occurrence.clockStartAt,
    });
  }
  if (occurrence.freeTimeExpiresAt <= to) {
    moments.push({
      key: "free",
      label: t("Free time ends"),
      time: cardTime(occurrence.freeTimeExpiresAt),
      at: occurrence.freeTimeExpiresAt,
    });
  }
  moments.push(
    live
      ? { key: "end", label: t("Now"), time: cardTime(now), at: to }
      : { key: "end", label: t("Departed"), time: cardTime(end), at: to },
  );

  const span = to - from;
  let last = Number.NEGATIVE_INFINITY;
  let low = false;
  const ticks = [...moments]
    .sort((a, b) => a.at - b.at)
    .map((moment) => {
      const share = (moment.at - from) / span;
      low = share - last < CROWDED ? !low : false;
      last = share;
      return { ...moment, low };
    });

  let flag: ClockFlag | null = null;
  if (occurrence.noticeSentAt) {
    flag = {
      label: t("Notice {0}", cardTime(occurrence.noticeSentAt)),
      at: occurrence.noticeSentAt,
      tone: "ok",
    };
  } else if (occurrence.noticeDueAt && noticeReading(occurrence, now) !== "none") {
    flag = {
      label: t("Notice due {0}", cardTime(occurrence.noticeDueAt)),
      at: occurrence.noticeDueAt,
      tone: "warn",
    };
  }
  if (flag && (flag.at < from || flag.at > to)) flag = null;

  return {
    from,
    to,
    live,
    spans,
    ticks,
    flag,
    dwellMinutes: live
      ? minutesBetween(occurrence.arrivedAt ?? occurrence.clockStartAt, now)
      : occurrence.rawDwellMinutes,
    freeMinutes: occurrence.freeMinutesGranted,
    billable: live
      ? {
          minutes: minutesBetween(occurrence.freeTimeExpiresAt, now),
          rounded: detail.currentlyBillable?.billableMinutes ?? occurrence.roundedMinutes,
        }
      : { minutes: occurrence.billableMinutes, rounded: occurrence.roundedMinutes },
  };
}

/** One step of the calculation as the receipt reads it. */
export type ReceiptRow = {
  key: string;
  label: string;
  detail: string;
  term: string;
  value: string;
  cut: boolean;
};

/**
 * The calculation as the server recorded it, step by step, with the minutes,
 * money or moment each step settled on; the final step is the charge. The
 * step's words are the engine's own, the sentence a clerk can paste. The
 * worker's detention pay is not part of the charge, so it is left to the
 * split beside the receipt.
 */
export function receiptRows(
  occurrence: Occurrence,
): { rows: ReceiptRow[]; total: string | null; engine: string; computedAt: number } | null {
  const trace = occurrence.trace;
  if (!trace || trace.steps.length === 0) return null;
  const rows: ReceiptRow[] = [];
  let total: string | null = null;
  trace.steps.forEach((step, index) => {
    if (step.kind === "DriverPay") return;
    if (step.kind === "Final") {
      total =
        step.amount === null || step.amount === undefined
          ? null
          : money(step.amount, occurrence.currency);
      return;
    }
    const value =
      step.amount !== null && step.amount !== undefined
        ? `${step.isReduction ? "−" : ""}${money(Math.abs(step.amount), occurrence.currency)}`
        : step.minutes !== null && step.minutes !== undefined
          ? `${step.isReduction ? "−" : ""}${formatDetentionMinutes(step.minutes)}`
          : step.timestamp
            ? cardTime(step.timestamp)
            : "";
    rows.push({
      key: `${step.kind}:${index}`,
      label: step.label,
      detail: step.detail,
      term: step.term,
      value,
      cut: step.isReduction,
    });
  });
  return {
    rows,
    total: total ?? money(occurrence.billableAmount, occurrence.currency),
    engine: trace.engineVersion,
    computedAt: trace.computedAt,
  };
}

function amountOrNull(value: string | null | undefined): number | null {
  return value === null || value === undefined ? null : Number(value) || 0;
}

/**
 * What the customer is billed, what the worker is paid for the wait, and what is kept. The pay
 * and what is kept are null to a reader without the margins field group: beside the bill the
 * pay gives the margin away.
 */
export function paySplit(
  occurrence: Occurrence,
  t: TranslateFn,
): {
  bill: number;
  pay: number | null;
  margin: number | null;
  payNote: string;
  payShare: number;
} {
  const bill = Number(occurrence.billableAmount) || 0;
  const pay = amountOrNull(occurrence.driverPayAmount);
  const margin = amountOrNull(occurrence.netMargin);
  return {
    bill,
    pay,
    margin,
    payNote: t("{0} payable", formatDetentionMinutes(occurrence.driverPayMinutes)),
    payShare: bill > 0 && pay !== null ? Math.min(1, Math.max(0, pay / bill)) : 0,
  };
}

/** What the policy asks of the notice, in a sentence, or nothing when it asks for none. */
export function noticeRule(occurrence: Occurrence, t: TranslateFn): string {
  const policy = occurrence.snapshot;
  if (!policy || policy.notificationRequirement === "None") return "";
  const lead = policy.notificationLeadMinutes ?? 0;
  const ask =
    lead > 0
      ? t(
          "{0} asks for notice {1} before free time runs out",
          policy.policyName,
          formatDetentionMinutes(lead),
        )
      : t("{0} asks for notice before free time runs out", policy.policyName);
  if (policy.notificationRequirement === "Advisory") return t("{0}; it is advisory.", ask);
  switch (policy.unnotifiedBehavior) {
    case "Suppress":
      return t("{0}; without it the charge won't bill.", ask);
    case "Flag":
      return t("{0}; without it the charge is held for review.", ask);
    case "Bill":
      return t("{0}.", ask);
  }
}

/** The warning above the notice, when one is overdue or failed: what happened and by when to act. */
export function noticeSlip(
  occurrence: Occurrence,
  now: number,
  t: TranslateFn,
): { title: string; body: string } | null {
  const reading = noticeReading(occurrence, now);
  if (reading === "failed") {
    return {
      title: t("The notice could not be delivered"),
      body: t("Check the customer's detention contacts and send it again."),
    };
  }
  if (reading !== "overdue" || !occurrence.noticeDueAt) return null;
  const late = formatDetentionMinutes(minutesBetween(occurrence.noticeDueAt, now));
  const suppressed = occurrence.snapshot?.unnotifiedBehavior === "Suppress";
  if (occurrence.noticeDeadlineAt && now < occurrence.noticeDeadlineAt) {
    return {
      title: t("Notice overdue by {0}", late),
      body: suppressed
        ? t(
            "It was due at {0}. Send it before {1}, or this policy holds the charge back and it won't bill.",
            cardTime(occurrence.noticeDueAt),
            cardTime(occurrence.noticeDeadlineAt),
          )
        : t(
            "It was due at {0}. Send it before {1}.",
            cardTime(occurrence.noticeDueAt),
            cardTime(occurrence.noticeDeadlineAt),
          ),
    };
  }
  return {
    title: t("Notice overdue by {0}", late),
    body: suppressed
      ? t(
          "It was due at {0}. Without it this policy holds the charge back.",
          cardTime(occurrence.noticeDueAt),
        )
      : t("It was due at {0}.", cardTime(occurrence.noticeDueAt)),
  };
}

/** The evidence trail in the order it happened. */
export function evidenceTrail(evidence: readonly DeskDetentionEvidence[]): DeskDetentionEvidence[] {
  return [...evidence].sort((a, b) => a.observedAt - b.observedAt);
}

/** Why the charge was waived or disputed, and by whom, for the slip at the card's top. */
export function settledSlip(
  occurrence: Occurrence,
  t: TranslateFn,
): { title: string; note: string; facts: { key: string; label: string; value: string }[] } | null {
  if (occurrence.status === "Waived") {
    return {
      title: t("Why it was waived"),
      note: occurrence.waiverNote.trim(),
      facts: [
        {
          key: "reason",
          label: t("Reason"),
          value: occurrence.waiverReason
            ? choiceLabel(detentionWaiverReasonChoices, occurrence.waiverReason)
            : t("Not given"),
        },
        { key: "by", label: t("Waived by"), value: occurrence.waivedBy?.name ?? t("Someone") },
        { key: "when", label: t("When"), value: cardMoment(occurrence.waivedAt) },
        {
          key: "reached",
          label: t("Had reached"),
          value: occurrence.approvedAt ? t("Approved") : t("Waiting on approval"),
        },
      ],
    };
  }
  if (occurrence.status === "Disputed") {
    return {
      title: t("What the customer disputes"),
      note: occurrence.disputeNote.trim(),
      facts: [
        { key: "when", label: t("Disputed"), value: cardMoment(occurrence.disputedAt) },
        {
          key: "amount",
          label: t("Amount"),
          value: money(occurrence.billableAmount, occurrence.currency),
        },
      ],
    };
  }
  return null;
}

/** One notice sent or due for the stop, as the notice list reads it. */
export type NoticeRow = {
  key: string;
  title: string;
  to: string;
  when: string;
  tone: "ok" | "warn" | "plain";
};

/** The notices for the stop, latest last: what each was, where it went, and how it landed. */
export function noticeRows(notices: readonly DeskDetentionNotice[], t: TranslateFn): NoticeRow[] {
  return [...notices]
    .sort((a, b) => (a.sentAt ?? a.scheduledFor) - (b.sentAt ?? b.scheduledFor))
    .map((notice) => {
      const failed = notice.deliveryStatus === "Failed" || notice.deliveryStatus === "Bounced";
      const recipients = notice.recipients ?? [];
      return {
        key: notice.id,
        title: t(
          "{0} · {1}",
          noticeKindLabel(notice.kind, t),
          deliveryStatusLabel(notice.deliveryStatus, t),
        ),
        to:
          failed && notice.failureReason
            ? notice.failureReason
            : recipients.length > 0
              ? t("To {0}", formatList(recipients, "conjunction"))
              : t("No recipients recorded"),
        when: notice.sentAt
          ? cardMoment(notice.sentAt)
          : t("due {0}", cardMoment(notice.scheduledFor)),
        tone: failed ? "warn" : notice.sentAt && notice.satisfiesRequirement ? "ok" : "plain",
      };
    });
}

/** A factor of the collectability score, and what would earn the points it is short. */
export type ProofFactor = {
  key: string;
  label: string;
  points: string;
  remedy: string;
  short: boolean;
};

/** One piece of evidence on the chain, as the trail reads it. */
export type ProofEvent = { key: string; title: string; summary: string; when: string };

/**
 * Whether the charge would stand up to a dispute: the score out of 100 and
 * its band, each factor with what is missing, the evidence in the order it
 * happened, and whether every record still matches the hash it was written
 * with.
 */
export function proofReading(
  detail: DeskDetention,
  t: TranslateFn,
): {
  score: number;
  band: string;
  weak: boolean;
  factors: ProofFactor[];
  events: ProofEvent[];
  aside: string;
  chain: string;
  chainValid: boolean;
} {
  const { collectability, evidence } = detail;
  const score = Math.max(0, Math.min(100, Math.round(collectability.score)));
  const count = evidence.length;
  return {
    score,
    band: scoreBandLabel(collectability.band, t),
    weak: collectability.band === "Weak" || collectability.band === "AtRisk",
    factors: collectability.factors.map((factor) => {
      const short = factor.earned < factor.possible;
      return {
        key: factor.key,
        label: factor.label,
        points: `${factor.earned}/${factor.possible}`,
        remedy: short ? (factor.remedy ?? "").trim() : "",
        short,
      };
    }),
    events: evidenceTrail(evidence).map((item) => ({
      key: item.id,
      title: t("{0} · {1}", evidenceKindLabel(item.kind, t), evidenceSourceLabel(item.source, t)),
      summary: item.summary,
      when: cardTime(item.observedAt),
    })),
    aside: collectability.chainValid
      ? t("{0, plural, one {# record} other {# records}} · chain verified", count)
      : t("{0, plural, one {# record} other {# records}} · chain broken", count),
    chain: collectability.chainValid
      ? t("Every record still matches the hash it was written with")
      : t("A record changed after it was written; the evidence chain no longer verifies"),
    chainValid: collectability.chainValid,
  };
}

function scoreBandLabel(band: DeskDetention["collectability"]["band"], t: TranslateFn): string {
  switch (band) {
    case "Strong":
      return t("of 100 · strong");
    case "Adequate":
      return t("of 100 · adequate");
    case "Weak":
      return t("of 100 · weak");
    case "AtRisk":
      return t("of 100 · at risk");
  }
}

/** The steps the card offers, under the permission the server checks, with the one that leads. */
export type DetentionAction = "approve" | "notice" | "dispute" | "waive";

/**
 * What can still be done to the charge, as the detention desk offers it:
 * approve one waiting on approval, send the notice while one is owed, record
 * a dispute against a billable charge, and waive one that is not already
 * waived or unbillable. Approval leads, then the notice.
 */
export function detentionActions(occurrence: Occurrence): {
  primary: DetentionAction | null;
  more: DetentionAction[];
} {
  const actions: DetentionAction[] = [];
  if (occurrence.status === "Pending") actions.push("approve");
  if (occurrence.notificationStatus !== "NotRequired" && occurrence.notificationStatus !== "Sent") {
    actions.push("notice");
  }
  if (occurrence.status !== "Disputed" && Number(occurrence.billableAmount) > 0) {
    actions.push("dispute");
  }
  if (occurrence.status !== "Waived" && occurrence.status !== "NotBillable") {
    actions.push("waive");
  }
  const primary = actions.find((action) => action === "approve" || action === "notice") ?? null;
  return { primary, more: actions.filter((action) => action !== primary) };
}
