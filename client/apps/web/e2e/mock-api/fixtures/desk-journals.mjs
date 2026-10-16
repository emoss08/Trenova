// The journal entries and manual journals the Desk cards thread names, as the
// DeskJournalEntryCard and DeskManualJournalCard queries answer for them: the
// same fixtures the cards' unit tests read, moved by whole days so their
// booking days stay the first second of a UTC day. JE-2610-00418 is Acme's
// invoice posted; JE-2610-00431 waits on approval; JE-2610-00402 was reversed
// by JE-2610-00427; JE-2610-00422 is what a written-off short pay became.
// MJ-2610-0007 waits on approval; MJ-2610-0009 is a draft $40.00 out of
// balance asked for in a closed September; MJ-2610-0004 is posted; and
// MJ-2610-0005 was rejected.
import jePending from "../../../src/routes/desk/_components/artifacts/cards/journal-entry/__tests__/je-pending.json" with { type: "json" };
import jePosted from "../../../src/routes/desk/_components/artifacts/cards/journal-entry/__tests__/je-posted.json" with { type: "json" };
import jeReversal from "../../../src/routes/desk/_components/artifacts/cards/journal-entry/__tests__/je-reversal.json" with { type: "json" };
import jeReversed from "../../../src/routes/desk/_components/artifacts/cards/journal-entry/__tests__/je-reversed.json" with { type: "json" };
import jeWriteoff from "../../../src/routes/desk/_components/artifacts/cards/journal-entry/__tests__/je-writeoff.json" with { type: "json" };
import mjDraft from "../../../src/routes/desk/_components/artifacts/cards/manual-journal/__tests__/mj-draft.json" with { type: "json" };
import mjPending from "../../../src/routes/desk/_components/artifacts/cards/manual-journal/__tests__/mj-pending.json" with { type: "json" };
import mjPosted from "../../../src/routes/desk/_components/artifacts/cards/manual-journal/__tests__/mj-posted.json" with { type: "json" };
import mjRejected from "../../../src/routes/desk/_components/artifacts/cards/manual-journal/__tests__/mj-rejected.json" with { type: "json" };

const DAY = 86_400;

/** Every Unix-second instant and booking day in a fixture moved by `delta` whole days; the rest copied as it is. */
function moved(value, delta) {
  if (Array.isArray(value)) return value.map((entry) => moved(entry, delta));
  if (value === null || typeof value !== "object") return value;
  return Object.fromEntries(
    Object.entries(value).map(([key, entry]) => [
      key,
      /(At|Date)$/.test(key) && typeof entry === "number" && entry > 0
        ? entry + delta
        : moved(entry, delta),
    ]),
  );
}

function at(fixture, anchor) {
  return moved(fixture.card, Math.round((anchor - fixture.now) / DAY) * DAY);
}

/** Each journal entry the cards thread names, by id. */
export function deskJournalEntries(anchor) {
  return Object.fromEntries(
    [jePosted, jePending, jeReversed, jeReversal, jeWriteoff].map((fixture) => [
      fixture.card.entry.id,
      at(fixture, anchor),
    ]),
  );
}

/** Each manual journal the cards thread names, by id. */
export function deskManualJournals(anchor) {
  return Object.fromEntries(
    [mjPending, mjDraft, mjPosted, mjRejected].map((fixture) => [
      fixture.card.journal.id,
      at(fixture, anchor),
    ]),
  );
}

/** The cards thread's journal entry conversations. */
export const DESK_JOURNAL_ENTRY_CONVERSATIONS = {
  "journal-posted": {
    recordId: "je_2610_00418",
    title: "Journal entry JE-2610-00418",
    question: "What did INV-2610-0042 book to the ledger?",
    reply:
      "One entry, posted Oct 14: $3,612.50 to receivables against freight, fuel surcharge and accessorial revenue. [JE-2610-00418](artifact:art_journal-posted) has the lines.",
  },
  "journal-pending": {
    recordId: "je_2610_00431",
    title: "Journal entry JE-2610-00431",
    question: "What's holding Marcus Hale's settlement journal?",
    reply:
      "It is booked and balanced but waits on approval before it reaches the general ledger. [JE-2610-00431](artifact:art_journal-pending) shows what it posts.",
  },
  "journal-reversed": {
    recordId: "je_2610_00402",
    title: "Journal entry JE-2610-00402",
    question: "Is the Riverside duplicate still on the books?",
    reply:
      "No. JE-2610-00427 reversed it on Oct 16, so the pair nets to zero. [JE-2610-00402](artifact:art_journal-reversed) shows both.",
  },
  "journal-gone": {
    recordId: "je_2610_00001",
    title: "Journal entry JE-2610-00001",
    question: "Pull up JE-2610-00001.",
    reply:
      "That entry can no longer be read. [JE-2610-00001](artifact:art_journal-gone) is what I found then.",
  },
};

/** The cards thread's manual journal conversations. */
export const DESK_MANUAL_JOURNAL_CONVERSATIONS = {
  "manual-pending": {
    recordId: "mjr_2610_0007",
    title: "Manual journal MJ-2610-0007",
    question: "Which manual journals are waiting on me?",
    reply:
      "One: Dana's October fuel accrual, $24,535.50, balanced and ready to approve. [MJ-2610-0007](artifact:art_manual-pending) has the lines and why.",
  },
  "manual-draft": {
    recordId: "mjr_2610_0009",
    title: "Manual journal MJ-2610-0009",
    question: "Can Dana's toll reclass go in?",
    reply:
      "Not yet. It is still a draft and $40.00 out of balance, and September is closed, so it would post on Oct 1. [MJ-2610-0009](artifact:art_manual-draft) shows the gap.",
  },
  "manual-posted": {
    recordId: "mjr_2610_0004",
    title: "Manual journal MJ-2610-0004",
    question: "Did the Harbor Freight write-off post?",
    reply:
      "Yes. Priya approved and posted it Oct 13 as JE-2610-00422. [MJ-2610-0004](artifact:art_manual-posted) has the entry.",
  },
  "manual-rejected": {
    recordId: "mjr_2610_0005",
    title: "Manual journal MJ-2610-0005",
    question: "What happened to the Acme detention write-off?",
    reply:
      "Priya rejected it Oct 11 and asked for it to be waived under the detention policy instead. [MJ-2610-0005](artifact:art_manual-rejected) has her note.",
  },
  "manual-gone": {
    recordId: "mjr_2610_0001",
    title: "Manual journal MJ-2610-0001",
    question: "Pull up MJ-2610-0001.",
    reply:
      "That journal can no longer be read. [MJ-2610-0001](artifact:art_manual-gone) is what I found then.",
  },
};
