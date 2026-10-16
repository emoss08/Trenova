import type { DeskManualJournalCard } from "@/lib/graphql/journal-cards";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { journalTotals } from "../../../record-card/journal-values";
import { cardDay, cardMoment } from "../../../record-card/record-card-values";
import {
  journalActions,
  journalFigure,
  journalIsOff,
  journalKicker,
  journalSettled,
  journalSignoffs,
  journalStanding,
  journalSubtitle,
  journalTrack,
  journalTrackNote,
  postingPlace,
  submitBlocker,
} from "../manual-journal-card";
import draftFixture from "./mj-draft.json";
import pendingFixture from "./mj-pending.json";
import postedFixture from "./mj-posted.json";
import rejectedFixture from "./mj-rejected.json";

const t = translate as TranslateFn;

const pending = pendingFixture.card as unknown as DeskManualJournalCard;
const draft = draftFixture.card as unknown as DeskManualJournalCard;
const posted = postedFixture.card as unknown as DeskManualJournalCard;
const rejected = rejectedFixture.card as unknown as DeskManualJournalCard;

function withCard(card: DeskManualJournalCard, change: (copy: DeskManualJournalCard) => void) {
  const copy = structuredClone(card);
  change(copy);
  return copy;
}

const cancelled = withCard(pending, (copy) => {
  copy.journal.status = "Cancelled";
  copy.journal.cancelledAt = 1792081200;
  copy.journal.cancelReason = "Statements arrived early; booked from them instead.";
  copy.cancelledBy = { id: "usr_dana", name: "Dana Ruiz" };
  copy.posting = null;
});

describe("journalStanding", () => {
  it("reads each status as its pill", () => {
    const standing = (card: DeskManualJournalCard) => journalStanding(card.journal, t);
    expect(standing(draft)).toEqual({ label: "Draft", tone: "draft", mark: "dot" });
    expect(standing(pending)).toEqual({
      label: "Waiting on approval",
      tone: "progress",
      mark: "dot",
    });
    expect(standing(withCard(pending, (copy) => (copy.journal.status = "Approved")))).toEqual({
      label: "Approved, not posted",
      tone: "progress",
      mark: "dot",
    });
    expect(standing(posted)).toEqual({ label: "Posted", tone: "success", mark: "check" });
    expect(standing(rejected)).toEqual({ label: "Rejected", tone: "danger", mark: "x" });
    expect(standing(cancelled)).toEqual({ label: "Cancelled", tone: "muted", mark: "dot" });
  });

  it("reads only rejected and cancelled requests as off the way to the ledger", () => {
    expect(
      [draft, pending, posted, rejected, cancelled].map((card) => journalIsOff(card.journal)),
    ).toEqual([false, false, false, true, true]);
  });
});

describe("the journal's head", () => {
  it("names the request, what it is and its currency", () => {
    expect(journalKicker(pending, t)).toEqual(["MJ-2610-0007", "Manual journal", "USD"]);
  });

  it("names who asked for it and counts the lines", () => {
    expect(journalSubtitle(pending, t)).toBe("Requested by Dana Ruiz · 3 lines");
    expect(
      journalSubtitle(
        withCard(pending, (copy) => (copy.createdBy = null)),
        t,
      ),
    ).toBe("3 lines");
  });

  it("gives what each side comes to while it balances", () => {
    expect(journalFigure(pending, journalTotals(pending.lines), t)).toEqual({
      amount: "$24,535.50",
      caption: "each side · balanced",
      warn: false,
      struck: false,
    });
    expect(journalFigure(posted, journalTotals(posted.lines), t)).toEqual({
      amount: "$312.00",
      caption: "each side · posted",
      warn: false,
      struck: false,
    });
  });

  it("strikes the figure of a request that never posted", () => {
    expect(journalFigure(rejected, journalTotals(rejected.lines), t)).toEqual({
      amount: "$250.00",
      caption: "each side · never posted",
      warn: false,
      struck: true,
    });
  });

  it("leads with how far off a journal is, and which side is over", () => {
    expect(journalFigure(draft, journalTotals(draft.lines), t)).toEqual({
      amount: "Off by $40.00",
      caption: "debits exceed credits",
      warn: true,
      struck: false,
    });
    const creditHeavy = withCard(draft, (copy) => (copy.lines[1]!.creditAmount = 130000));
    expect(journalFigure(creditHeavy, journalTotals(creditHeavy.lines), t)).toMatchObject({
      amount: "Off by $40.00",
      caption: "credits exceed debits",
    });
  });
});

describe("journalTrack", () => {
  const states = (card: DeskManualJournalCard) =>
    journalTrack(card.journal, t).map((step) => `${step.label}:${step.state}`);

  it("stands on each step in turn on the way to the ledger", () => {
    expect(states(draft)).toEqual(["Draft:now", "Submitted:next", "Approved:next", "Posted:next"]);
    expect(states(pending)).toEqual([
      "Draft:done",
      "Submitted:now",
      "Approved:next",
      "Posted:next",
    ]);
    expect(states(posted)).toEqual(["Draft:done", "Submitted:done", "Approved:done", "Posted:now"]);
  });

  it("stops a rejected request at the submission it was refused on", () => {
    expect(states(rejected)).toEqual([
      "Draft:done",
      "Submitted:done",
      "Approved:next",
      "Posted:next",
    ]);
  });

  it("stops a cancelled request where it got to: approved, or never past draft", () => {
    const approvedThenCancelled = withCard(cancelled, (copy) => {
      copy.journal.approvedAt = 1791882120;
    });
    expect(states(approvedThenCancelled)).toEqual([
      "Draft:done",
      "Submitted:done",
      "Approved:done",
      "Posted:next",
    ]);
    expect(states(cancelled)).toEqual([
      "Draft:done",
      "Submitted:next",
      "Approved:next",
      "Posted:next",
    ]);
  });
});

describe("journalTrackNote", () => {
  it("says when a draft was started and a request submitted", () => {
    expect(journalTrackNote(draft, t)).toBe(
      `Started ${cardDay(draft.journal.createdAt)} · not submitted yet`,
    );
    expect(journalTrackNote(pending, t)).toBe(
      `Submitted ${cardDay(pending.journal.updatedAt)} · waiting on someone who may approve manual journals`,
    );
  });

  it("names who approved a request waiting to post", () => {
    const approved = withCard(pending, (copy) => {
      copy.journal.status = "Approved";
      copy.journal.approvedAt = 1792143000;
      copy.approvedBy = { id: "usr_priya", name: "Priya Shah" };
    });
    expect(journalTrackNote(approved, t)).toBe(
      `Approved ${cardMoment(1792143000)} by Priya Shah · waiting to be posted`,
    );
  });

  it("names the entry a posted request became", () => {
    expect(journalTrackNote(posted, t)).toBe(`Posted ${cardDay(posted.postedAt)} as JE-2610-00422`);
    expect(
      journalTrackNote(
        withCard(posted, (copy) => (copy.entries = [])),
        t,
      ),
    ).toBe(`Posted ${cardDay(posted.postedAt)}`);
  });

  it("names who rejected or cancelled a request and that nothing reached the ledger", () => {
    expect(journalTrackNote(rejected, t)).toBe(
      `Rejected ${cardDay(rejected.journal.rejectedAt)} by Priya Shah; nothing reached the general ledger`,
    );
    expect(journalTrackNote(cancelled, t)).toBe(
      `Cancelled ${cardDay(1792081200)} by Dana Ruiz; nothing reached the general ledger`,
    );
  });
});

describe("journalSignoffs", () => {
  it("lists nobody for a draft no one has submitted", () => {
    expect(journalSignoffs(draft, t)).toEqual([]);
  });

  it("names who asked, approved and posted, and what has not happened yet", () => {
    expect(journalSignoffs(posted, t)).toEqual([
      {
        key: "requested",
        step: "Requested",
        who: "Dana Ruiz",
        when: cardMoment(posted.journal.createdAt),
      },
      {
        key: "approved",
        step: "Approved",
        who: "Priya Shah",
        when: cardMoment(posted.journal.approvedAt),
      },
      { key: "posted", step: "Posted", who: "Priya Shah", when: cardMoment(posted.postedAt) },
    ]);
    expect(journalSignoffs(pending, t).map((row) => `${row.step}:${row.who}:${row.when}`)).toEqual([
      `Requested:Dana Ruiz:${cardMoment(pending.journal.createdAt)}`,
      "Approved:—:Not yet",
      "Posted:—:Not yet",
    ]);
  });
});

describe("journalSettled", () => {
  it("keeps the reason a request was rejected, with who and when", () => {
    expect(journalSettled(rejected, t)).toEqual({
      title: "Why it was rejected",
      reason:
        "Waive it under the detention policy instead of a manual write-off, so the stop's record shows why.",
      by: `Priya Shah, ${cardDay(rejected.journal.rejectedAt)}`,
    });
  });

  it("keeps the reason a request was cancelled, and says when none was given", () => {
    expect(journalSettled(cancelled, t)).toEqual({
      title: "Why it was cancelled",
      reason: "Statements arrived early; booked from them instead.",
      by: `Dana Ruiz, ${cardDay(1792081200)}`,
    });
    const unexplained = withCard(cancelled, (copy) => {
      copy.journal.cancelReason = "  ";
      copy.cancelledBy = null;
    });
    expect(journalSettled(unexplained, t)).toEqual({
      title: "Why it was cancelled",
      reason: "No reason was given.",
      by: cardDay(1792081200),
    });
  });

  it("has nothing to say of a request still on its way", () => {
    expect(journalSettled(pending, t)).toBeNull();
    expect(journalSettled(posted, t)).toBeNull();
  });
});

describe("postingPlace", () => {
  it("posts on the day asked for when its period takes postings", () => {
    expect(postingPlace(pending)).toEqual({
      kind: "asked",
      period: pending.requestedPeriod,
      date: 1793404800,
    });
  });

  it("moves to the first open day when the period asked for is shut", () => {
    expect(postingPlace(draft)).toEqual({
      kind: "moved",
      from: draft.requestedPeriod,
      to: draft.posting!.period,
      date: 1790812800,
    });
  });

  it("gives the refusal when it cannot post anywhere", () => {
    const refused = withCard(draft, (copy) => {
      copy.posting = {
        period: null,
        accountingDate: null,
        refusal: "September 2026 is closed and your policy refuses late postings",
      };
    });
    expect(postingPlace(refused)).toEqual({
      kind: "refused",
      from: draft.requestedPeriod,
      reason: "September 2026 is closed and your policy refuses late postings",
    });
  });

  it("falls back to the date asked for when the server gives no posting date", () => {
    const undated = withCard(pending, (copy) => (copy.posting!.accountingDate = null));
    expect(postingPlace(undated)).toMatchObject({ kind: "asked", date: 1793404800 });
  });

  it("has no place for a request already posted or settled", () => {
    expect(postingPlace(posted)).toBeNull();
    expect(postingPlace(rejected)).toBeNull();
  });
});

describe("journalActions", () => {
  it("offers each status only the steps the server takes from it", () => {
    expect(journalActions(draft.journal)).toEqual({
      submit: true,
      approveAndPost: false,
      post: false,
      reject: false,
      cancel: true,
    });
    expect(journalActions(pending.journal)).toEqual({
      submit: false,
      approveAndPost: true,
      post: false,
      reject: true,
      cancel: true,
    });
    expect(journalActions({ ...pending.journal, status: "Approved" })).toEqual({
      submit: false,
      approveAndPost: false,
      post: true,
      reject: false,
      cancel: true,
    });
    for (const card of [posted, rejected, cancelled]) {
      expect(Object.values(journalActions(card.journal))).toEqual([
        false,
        false,
        false,
        false,
        false,
      ]);
    }
  });
});

describe("submitBlocker", () => {
  it("asks for two lines, then for them to balance, before a draft goes in", () => {
    expect(submitBlocker(1, journalTotals(draft.lines.slice(0, 1)), t)).toBe(
      "Add at least two lines on its page before submitting",
    );
    expect(submitBlocker(2, journalTotals(draft.lines), t)).toBe(
      "Balance the lines on its page before submitting",
    );
    expect(submitBlocker(3, journalTotals(pending.lines), t)).toBeNull();
  });
});
