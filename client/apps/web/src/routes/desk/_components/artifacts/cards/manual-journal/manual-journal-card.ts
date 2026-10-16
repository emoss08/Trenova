import type {
  DeskManualJournal,
  DeskManualJournalCard,
  JournalCardPeriod,
} from "@/lib/graphql/journal-cards";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CardTrackStep } from "../../record-card/card-track";
import type { JournalTotals, Standing } from "../../record-card/journal-values";
import { cardDay, cardMoment, minorMoney } from "../../record-card/record-card-values";
import type { Signoff } from "../../record-card/settlement-values";

export function journalStanding(journal: DeskManualJournal, t: TranslateFn): Standing {
  switch (journal.status) {
    case "Draft":
      return { label: t("Draft"), tone: "draft", mark: "dot" };
    case "PendingApproval":
      return { label: t("Waiting on approval"), tone: "progress", mark: "dot" };
    case "Approved":
      return { label: t("Approved, not posted"), tone: "progress", mark: "dot" };
    case "Posted":
      return { label: t("Posted"), tone: "success", mark: "check" };
    case "Rejected":
      return { label: t("Rejected"), tone: "danger", mark: "x" };
    case "Cancelled":
      return { label: t("Cancelled"), tone: "muted", mark: "dot" };
    default:
      return { label: journal.status, tone: "muted", mark: "dot" };
  }
}

/** A request that left the way to the ledger: refused or withdrawn. */
export function journalIsOff(journal: DeskManualJournal): boolean {
  return journal.status === "Rejected" || journal.status === "Cancelled";
}

export function journalKicker(card: DeskManualJournalCard, t: TranslateFn): string[] {
  return [card.journal.requestNumber, t("Manual journal"), card.journal.currencyCode];
}

export function journalSubtitle(card: DeskManualJournalCard, t: TranslateFn): string {
  const lines = t("{0, plural, one {# line} other {# lines}}", card.lines.length);
  return card.createdBy ? t("Requested by {0} · {1}", card.createdBy.name, lines) : lines;
}

/**
 * The request's figure: what each side comes to once it balances, or how far
 * off it is while it does not, so a draft's gap is the first thing read.
 */
export function journalFigure(
  card: DeskManualJournalCard,
  totals: JournalTotals,
  t: TranslateFn,
): { amount: string; caption: string; warn: boolean; struck: boolean } {
  const currency = card.journal.currencyCode;
  if (!totals.balanced) {
    return {
      amount: t("Off by {0}", minorMoney(Math.abs(totals.difference), currency)),
      caption: totals.difference > 0 ? t("debits exceed credits") : t("credits exceed debits"),
      warn: true,
      struck: false,
    };
  }
  const off = journalIsOff(card.journal);
  return {
    amount: minorMoney(totals.debit, currency),
    caption:
      card.journal.status === "Posted"
        ? t("each side · posted")
        : off
          ? t("each side · never posted")
          : t("each side · balanced"),
    warn: false,
    struck: off,
  };
}

const ORDER = ["Draft", "PendingApproval", "Approved", "Posted"] as const;

/** The request's way to the ledger: drafted, submitted, approved, posted. */
export function journalTrack(journal: DeskManualJournal, t: TranslateFn): CardTrackStep[] {
  const labels = [t("Draft"), t("Submitted"), t("Approved"), t("Posted")];
  const keys = ["draft", "submitted", "approved", "posted"];
  let reached: number;
  if (journal.status === "Rejected") {
    reached = 1;
  } else if (journal.status === "Cancelled") {
    reached = journal.approvedAt !== null ? 2 : 0;
  } else {
    reached = ORDER.indexOf(journal.status as (typeof ORDER)[number]);
  }
  const off = journalIsOff(journal);
  return labels.map((label, index) => ({
    key: keys[index],
    label,
    state:
      index < reached || (off && index === reached) ? "done" : index === reached ? "now" : "next",
  }));
}

export function journalTrackNote(card: DeskManualJournalCard, t: TranslateFn): string {
  const { journal } = card;
  switch (journal.status) {
    case "Draft":
      return t("Started {0} · not submitted yet", cardDay(journal.createdAt));
    case "PendingApproval":
      return t(
        "Submitted {0} · waiting on someone who may approve manual journals",
        cardDay(journal.updatedAt),
      );
    case "Approved":
      return card.approvedBy
        ? t(
            "Approved {0} by {1} · waiting to be posted",
            cardMoment(journal.approvedAt),
            card.approvedBy.name,
          )
        : t("Approved {0} · waiting to be posted", cardMoment(journal.approvedAt));
    case "Posted": {
      const entry = card.entries.find((candidate) => candidate.isPosted && !candidate.isReversal);
      const when = cardDay(card.postedAt);
      return entry ? t("Posted {0} as {1}", when, entry.entryNumber) : t("Posted {0}", when);
    }
    case "Rejected":
      return card.rejectedBy
        ? t(
            "Rejected {0} by {1}; nothing reached the general ledger",
            cardDay(journal.rejectedAt),
            card.rejectedBy.name,
          )
        : t("Rejected {0}; nothing reached the general ledger", cardDay(journal.rejectedAt));
    case "Cancelled":
      return card.cancelledBy
        ? t(
            "Cancelled {0} by {1}; nothing reached the general ledger",
            cardDay(journal.cancelledAt),
            card.cancelledBy.name,
          )
        : t("Cancelled {0}; nothing reached the general ledger", cardDay(journal.cancelledAt));
    default:
      return "";
  }
}

/** Who asked for the journal, who approved it and who posted it, under its track. */
export function journalSignoffs(card: DeskManualJournalCard, t: TranslateFn): Signoff[] {
  const { journal } = card;
  if (journal.status === "Draft") return [];
  const notYet = t("Not yet");
  return [
    {
      key: "requested",
      step: t("Requested"),
      who: card.createdBy?.name ?? "—",
      when: cardMoment(journal.createdAt),
    },
    {
      key: "approved",
      step: t("Approved"),
      who: card.approvedBy?.name ?? "—",
      when: journal.approvedAt ? cardMoment(journal.approvedAt) : notYet,
    },
    {
      key: "posted",
      step: t("Posted"),
      who: card.postedBy?.name ?? "—",
      when: card.postedAt ? cardMoment(card.postedAt) : notYet,
    },
  ];
}

/** Why a refused or withdrawn request stands as it does, kept as written. */
export function journalSettled(
  card: DeskManualJournalCard,
  t: TranslateFn,
): { title: string; reason: string; by: string } | null {
  const { journal } = card;
  if (journal.status === "Rejected") {
    return {
      title: t("Why it was rejected"),
      reason: journal.rejectionReason?.trim() || t("No reason was given."),
      by: card.rejectedBy
        ? t("{0}, {1}", card.rejectedBy.name, cardDay(journal.rejectedAt))
        : cardDay(journal.rejectedAt),
    };
  }
  if (journal.status === "Cancelled") {
    return {
      title: t("Why it was cancelled"),
      reason: journal.cancelReason?.trim() || t("No reason was given."),
      by: card.cancelledBy
        ? t("{0}, {1}", card.cancelledBy.name, cardDay(journal.cancelledAt))
        : cardDay(journal.cancelledAt),
    };
  }
  return null;
}

/**
 * Where the request will post: on the day asked for in the period asked for,
 * on the first open day after a closed period under the organization's
 * policy, or nowhere until its period is reopened.
 */
export type PostingPlace =
  | { kind: "asked"; period: JournalCardPeriod; date: number }
  | { kind: "moved"; from: JournalCardPeriod | null; to: JournalCardPeriod; date: number }
  | { kind: "refused"; from: JournalCardPeriod | null; reason: string };

export function postingPlace(card: DeskManualJournalCard): PostingPlace | null {
  const { posting, requestedPeriod } = card;
  if (!posting) return null;
  if (posting.refusal || !posting.period) {
    return { kind: "refused", from: requestedPeriod, reason: posting.refusal };
  }
  const date = posting.accountingDate ?? card.journal.accountingDate;
  if (requestedPeriod && requestedPeriod.id === posting.period.id) {
    return { kind: "asked", period: posting.period, date };
  }
  return { kind: "moved", from: requestedPeriod, to: posting.period, date };
}

/** The steps a person can take on the request now, before checking their permissions. */
export type JournalActions = {
  submit: boolean;
  approveAndPost: boolean;
  post: boolean;
  reject: boolean;
  cancel: boolean;
};

export function journalActions(journal: DeskManualJournal): JournalActions {
  const status = journal.status;
  return {
    submit: status === "Draft",
    approveAndPost: status === "PendingApproval",
    post: status === "Approved",
    reject: status === "PendingApproval",
    cancel: status === "Draft" || status === "PendingApproval" || status === "Approved",
  };
}

/** Why a draft cannot be submitted yet, or null when it can. */
export function submitBlocker(lines: number, totals: JournalTotals, t: TranslateFn): string | null {
  if (lines < 2) return t("Add at least two lines on its page before submitting");
  if (!totals.balanced) return t("Balance the lines on its page before submitting");
  return null;
}
