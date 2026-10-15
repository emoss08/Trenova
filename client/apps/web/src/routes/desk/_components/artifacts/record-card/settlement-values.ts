import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { Operation } from "@trenova/shared/types/permission";
import type { CardTrackStep } from "./card-track";
import type { CardStatusMark, CardTone } from "./record-card";
import { cardDay, cardWeekday, money } from "./record-card-values";

/**
 * A settlement's way from draft to paid. Worker and carrier settlements walk
 * the same one, and a void can stop either before it is paid.
 */
export type SettlementStatus =
  | "Draft"
  | "PendingApproval"
  | "Approved"
  | "Posted"
  | "Paid"
  | "Voided";

const STEPS = ["Draft", "PendingApproval", "Approved", "Posted", "Paid"] as const;

/** The parts of a settlement the shared helpers read, whichever kind it is. */
export type SettlementLife = {
  status: SettlementStatus;
  payDate: number;
  createdAt: number;
  submittedAt?: number | null;
  approvedAt?: number | null;
  postedAt?: number | null;
  paidAt?: number | null;
  voidedAt?: number | null;
  paymentMethod: string;
  paymentReference: string;
  submittedBy?: { name: string } | null;
  approvedBy?: { name: string } | null;
  postedBy?: { name: string } | null;
  paidBy?: { name: string } | null;
  voidedBy?: { name: string } | null;
};

/** An amount the server keeps in minor units, as the card writes it. */
export function minor(amount: number, currency: string): string {
  return money(amount / 100, currency);
}

/** An amount in minor units with its sign, a minus for what is taken off. */
export function signedMinor(amount: number, currency: string): string {
  return amount < 0 ? `−${minor(-amount, currency)}` : minor(amount, currency);
}

/** A period as its first and last day, "Oct 5 – Oct 11". */
export function periodLabel(start: number, end: number, t: TranslateFn): string {
  return t("{0} – {1}", cardDay(start), cardDay(end));
}

/**
 * The settlement's way as the track draws it: the steps behind it ticked, the
 * one it stands at, the ones ahead. A voided settlement leaves the whole way
 * struck off, wherever it had got to.
 */
export function settlementTrack(status: SettlementStatus, t: TranslateFn): CardTrackStep[] {
  const labels: Record<(typeof STEPS)[number], string> = {
    Draft: t("Draft"),
    PendingApproval: t("Submitted"),
    Approved: t("Approved"),
    Posted: t("Posted"),
    Paid: t("Paid"),
  };
  const at = (STEPS as readonly string[]).indexOf(status);
  return STEPS.map((key, index) => ({
    key,
    label: labels[key],
    state: status === "Voided" ? "off" : index < at ? "done" : index === at ? "now" : "next",
  }));
}

/** Where the settlement stands, as the pill at the card's head says it. */
export function settlementStanding(
  status: SettlementStatus,
  t: TranslateFn,
): { label: string; tone: CardTone; mark: CardStatusMark } {
  switch (status) {
    case "Draft":
      return { label: t("Draft"), tone: "draft", mark: "dot" };
    case "PendingApproval":
      return { label: t("Waiting on approval"), tone: "warn", mark: "dot" };
    case "Approved":
      return { label: t("Approved"), tone: "progress", mark: "dot" };
    case "Posted":
      return { label: t("Posted, not paid"), tone: "progress", mark: "dot" };
    case "Paid":
      return { label: t("Paid"), tone: "success", mark: "check" };
    case "Voided":
      return { label: t("Voided"), tone: "muted", mark: "x" };
  }
}

/**
 * The line under the track: who moved the settlement last and what comes
 * next. A voided one says whether it had reached the ledger.
 */
export function trackNote(life: SettlementLife, t: TranslateFn): string {
  const pays = cardWeekday(life.payDate);
  switch (life.status) {
    case "Draft":
      return t("Generated {0} · not submitted yet · pays {1}", cardDay(life.createdAt), pays);
    case "PendingApproval":
      return life.submittedBy
        ? t(
            "Submitted by {0} {1} · approve it to pay {2}",
            life.submittedBy.name,
            cardDay(life.submittedAt),
            pays,
          )
        : t("Submitted {0} · approve it to pay {1}", cardDay(life.submittedAt), pays);
    case "Approved":
      return life.approvedBy
        ? t(
            "Approved by {0} {1} · post it to send it to the ledger",
            life.approvedBy.name,
            cardDay(life.approvedAt),
          )
        : t("Approved {0} · post it to send it to the ledger", cardDay(life.approvedAt));
    case "Posted":
      return t("Posted {0} · pays {1}", cardDay(life.postedAt), pays);
    case "Paid":
      return paidNote(life, t);
    case "Voided":
      return life.postedAt
        ? t("Voided after it was posted · its ledger entry was reversed")
        : t("Voided before it was posted · nothing reached the ledger");
  }
}

function paidNote(life: SettlementLife, t: TranslateFn): string {
  const paidDay = cardDay(life.paidAt);
  if (!life.paidAt || paidDay === cardDay(life.payDate)) {
    return t("Paid {0} · on its pay date", paidDay);
  }
  return life.paidAt > life.payDate
    ? t("Paid {0} · after its pay date of {1}", paidDay, cardDay(life.payDate))
    : t("Paid {0} · before its pay date of {1}", paidDay, cardDay(life.payDate));
}

/** The furthest step a voided settlement had reached before it was voided. */
export function reachedStep(life: SettlementLife, t: TranslateFn): string {
  if (life.postedAt) return t("Posted · its ledger entry was reversed");
  if (life.approvedAt) return t("Approved · never posted");
  if (life.submittedAt) return t("Submitted · never approved");
  return t("Draft · never submitted");
}

/** One sign-off on the way: the step, who took it, and when (with how, for the payment). */
export type Signoff = { key: string; step: string; who: string; when: string };

/** Who moved the settlement through each step it has passed, in the order they came. */
export function signoffs(life: SettlementLife, t: TranslateFn): Signoff[] {
  const rows: Signoff[] = [];
  const add = (
    key: string,
    step: string,
    by: { name: string } | null | undefined,
    at: number | null | undefined,
    how = "",
  ) => {
    if (!at) return;
    rows.push({
      key,
      step,
      who: by?.name || t("Someone"),
      when: how ? t("{0} · {1}", cardDay(at), how) : cardDay(at),
    });
  };
  add("submitted", t("Submitted"), life.submittedBy, life.submittedAt);
  add("approved", t("Approved"), life.approvedBy, life.approvedAt);
  add("posted", t("Posted"), life.postedBy, life.postedAt);
  add("paid", t("Paid"), life.paidBy, life.paidAt, life.paymentMethod);
  add("voided", t("Voided"), life.voidedBy, life.voidedAt);
  return rows;
}

/** A step someone takes on a settlement from the card. */
export type SettlementAction =
  | "submit"
  | "recalculate"
  | "approve"
  | "reject"
  | "post"
  | "pay"
  | "void";

/** The permission the server checks for each step, the same for worker and carrier settlements. */
export const SETTLEMENT_ACTION_OPERATION: Record<SettlementAction, number> = {
  submit: Operation.Submit,
  recalculate: Operation.Update,
  approve: Operation.Approve,
  reject: Operation.Reject,
  post: Operation.Approve,
  pay: Operation.Update,
  void: Operation.Cancel,
};

/**
 * What can be done to the settlement where it stands: the one step that moves
 * it on, and the others beside it. A paid or voided settlement is done with.
 */
export function settlementActions(status: SettlementStatus): {
  primary: SettlementAction | null;
  more: SettlementAction[];
} {
  switch (status) {
    case "Draft":
      return { primary: "submit", more: ["recalculate", "void"] };
    case "PendingApproval":
      return { primary: "approve", more: ["reject", "void"] };
    case "Approved":
      return { primary: "post", more: ["void"] };
    case "Posted":
      return { primary: "pay", more: ["void"] };
    case "Paid":
    case "Voided":
      return { primary: null, more: [] };
  }
}

/** What each step's button says. */
export function settlementActionLabel(
  action: SettlementAction,
  amount: string,
  t: TranslateFn,
): string {
  switch (action) {
    case "submit":
      return t("Submit for approval");
    case "recalculate":
      return t("Recalculate");
    case "approve":
      return t("Approve {0}", amount);
    case "reject":
      return t("Send back");
    case "post":
      return t("Post to the ledger");
    case "pay":
      return t("Mark {0} paid", amount);
    case "void":
      return t("Void");
  }
}
