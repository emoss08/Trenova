import type { JournalReversal } from "@/types/journal-reversal";

/** What a person fills in to ask for a journal entry to be reversed. */
export type ReversalRequestValues = {
  originalJournalEntryId: string;
  /** The day it should post, as a date field gives it: `YYYY-MM-DD`. */
  requestedAccountingDate: string;
  reasonCode: string;
  reasonText: string;
};

/** The request the reversal is created from, its date read as the start of that day in UTC. */
export function reversalRequestBody(values: ReversalRequestValues): Partial<JournalReversal> {
  return {
    originalJournalEntryId: values.originalJournalEntryId,
    requestedAccountingDate: new Date(values.requestedAccountingDate).getTime() / 1000,
    reasonCode: values.reasonCode.trim(),
    reasonText: values.reasonText.trim(),
  };
}

/** Today in the reader's own calendar, as a date field writes it. */
export function todayField(now: Date = new Date()): string {
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${now.getFullYear()}-${month}-${day}`;
}
