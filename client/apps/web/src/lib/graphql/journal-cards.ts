import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskJournalCardEntryFieldsFragmentDoc,
  DeskJournalCardLineFieldsFragmentDoc,
  DeskJournalCardPeriodFieldsFragmentDoc,
  DeskJournalCardPersonFieldsFragmentDoc,
  DeskJournalCardPostingFieldsFragmentDoc,
  DeskJournalEntryCardDocument,
  DeskManualJournalCardDocument,
  type DeskJournalCardEntryFieldsFragment,
  type DeskJournalCardLineFieldsFragment,
  type DeskJournalCardPeriodFieldsFragment,
  type DeskJournalCardPersonFieldsFragment,
  type DeskJournalEntryCardQuery,
  type DeskManualJournalCardQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type JournalCardLine = DeskJournalCardLineFieldsFragment;
export type JournalCardPeriod = DeskJournalCardPeriodFieldsFragment;
export type JournalCardPerson = DeskJournalCardPersonFieldsFragment;
export type JournalCardEntry = DeskJournalCardEntryFieldsFragment;

/** Where a journal not yet posted would land if it posted now; `refusal` says why it cannot. */
export type JournalCardPosting = {
  period: JournalCardPeriod | null;
  accountingDate: number | null;
  refusal: string;
};

type RawEntryCard = DeskJournalEntryCardQuery["journalEntryCard"];
type RawJournalCard = DeskManualJournalCardQuery["manualJournalCard"];
type RawPosting = RawEntryCard["posting"];
type RawPerson = RawEntryCard["createdBy"];
type RawPeriod = RawEntryCard["period"];
type RawEntry = RawEntryCard["reversalOf"];

export type DeskJournalEntry = JournalCardEntry &
  Pick<
    RawEntryCard["entry"],
    | "referenceType"
    | "referenceId"
    | "referenceNumber"
    | "fiscalPeriodId"
    | "reversalOfId"
    | "reversedById"
  >;

export type DeskJournalEntryCard = Omit<
  RawEntryCard,
  | "entry"
  | "lines"
  | "period"
  | "posting"
  | "createdBy"
  | "approvedBy"
  | "postedBy"
  | "rejectedBy"
  | "reversalOf"
  | "reversedBy"
  | "reversalRequestedBy"
> & {
  entry: DeskJournalEntry;
  lines: JournalCardLine[];
  period: JournalCardPeriod | null;
  posting: JournalCardPosting | null;
  createdBy: JournalCardPerson | null;
  approvedBy: JournalCardPerson | null;
  postedBy: JournalCardPerson | null;
  rejectedBy: JournalCardPerson | null;
  reversalOf: JournalCardEntry | null;
  reversedBy: JournalCardEntry | null;
  reversalRequestedBy: JournalCardPerson | null;
};
export type JournalCardSource = NonNullable<DeskJournalEntryCard["source"]>;
export type JournalCardReversal = NonNullable<DeskJournalEntryCard["reversal"]>;

export type DeskManualJournalCard = Omit<
  RawJournalCard,
  | "lines"
  | "requestedPeriod"
  | "posting"
  | "createdBy"
  | "approvedBy"
  | "rejectedBy"
  | "cancelledBy"
  | "postedBy"
  | "entries"
> & {
  lines: JournalCardLine[];
  requestedPeriod: JournalCardPeriod | null;
  posting: JournalCardPosting | null;
  createdBy: JournalCardPerson | null;
  approvedBy: JournalCardPerson | null;
  rejectedBy: JournalCardPerson | null;
  cancelledBy: JournalCardPerson | null;
  postedBy: JournalCardPerson | null;
  entries: JournalCardEntry[];
};
export type DeskManualJournal = DeskManualJournalCard["journal"];

function person(raw: RawPerson): JournalCardPerson | null {
  return raw ? getFragmentData(DeskJournalCardPersonFieldsFragmentDoc, raw) : null;
}

function period(raw: RawPeriod): JournalCardPeriod | null {
  return raw ? getFragmentData(DeskJournalCardPeriodFieldsFragmentDoc, raw) : null;
}

function entry(raw: RawEntry): JournalCardEntry | null {
  return raw ? getFragmentData(DeskJournalCardEntryFieldsFragmentDoc, raw) : null;
}

function posting(raw: RawPosting): JournalCardPosting | null {
  if (!raw) return null;
  const read = getFragmentData(DeskJournalCardPostingFieldsFragmentDoc, raw);
  return {
    period: read.period
      ? getFragmentData(DeskJournalCardPeriodFieldsFragmentDoc, read.period)
      : null,
    accountingDate: read.accountingDate ?? null,
    refusal: read.refusal,
  };
}

/**
 * Everything the Desk's journal entry card draws, read in one request: the
 * lines with each account's side of the books, who moved it, its period and,
 * while it waits, where it will post, its source and its reversal. The server
 * refuses an entry the person may not read, and the card says so.
 */
export async function fetchDeskJournalEntry(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskJournalEntryCard> {
  const data = await requestGraphQL({
    document: DeskJournalEntryCardDocument,
    operationName: "DeskJournalEntryCard",
    variables: { id },
    signal: options?.signal,
  });
  const card = data.journalEntryCard;
  return {
    ...card,
    entry: {
      ...getFragmentData(DeskJournalCardEntryFieldsFragmentDoc, card.entry),
      referenceType: card.entry.referenceType,
      referenceId: card.entry.referenceId,
      referenceNumber: card.entry.referenceNumber,
      fiscalPeriodId: card.entry.fiscalPeriodId,
      reversalOfId: card.entry.reversalOfId,
      reversedById: card.entry.reversedById,
    },
    lines: [...getFragmentData(DeskJournalCardLineFieldsFragmentDoc, card.lines)],
    period: period(card.period),
    posting: posting(card.posting),
    createdBy: person(card.createdBy),
    approvedBy: person(card.approvedBy),
    postedBy: person(card.postedBy),
    rejectedBy: person(card.rejectedBy),
    reversalOf: entry(card.reversalOf),
    reversedBy: entry(card.reversedBy),
    reversalRequestedBy: person(card.reversalRequestedBy),
  };
}

/**
 * Everything the Desk's manual journal card draws, read in one request: the
 * lines, the period asked for and where it will post, who asked, approved,
 * refused or posted it, and the entries it became.
 */
export async function fetchDeskManualJournal(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskManualJournalCard> {
  const data = await requestGraphQL({
    document: DeskManualJournalCardDocument,
    operationName: "DeskManualJournalCard",
    variables: { id },
    signal: options?.signal,
  });
  const card = data.manualJournalCard;
  return {
    ...card,
    lines: [...getFragmentData(DeskJournalCardLineFieldsFragmentDoc, card.lines)],
    requestedPeriod: period(card.requestedPeriod),
    posting: posting(card.posting),
    createdBy: person(card.createdBy),
    approvedBy: person(card.approvedBy),
    rejectedBy: person(card.rejectedBy),
    cancelledBy: person(card.cancelledBy),
    postedBy: person(card.postedBy),
    entries: [...getFragmentData(DeskJournalCardEntryFieldsFragmentDoc, card.entries)],
  };
}
