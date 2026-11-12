import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import {
  RecordShareCandidatesDocument,
  RecordShareFieldsFragmentDoc,
  RecordShareUserFieldsFragmentDoc,
  RecordSharesDocument,
  ShareRecordDocument,
  UnshareRecordDocument,
  type RecordShareEmailStatus,
  type RecordShareFieldsFragment,
  type RecordShareUserFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type { RecordShareEmailStatus };
export type RecordShareUser = RecordShareUserFieldsFragment;

/** One record shared with one teammate, as the share dialog lists it. */
export type RecordShare = Omit<RecordShareFieldsFragment, "sharedWith" | "sharedBy"> & {
  sharedWith: RecordShareUser | null;
  sharedBy: RecordShareUser | null;
};

/** What a share did: who the record is now shared with, and whether the emails went. */
export type ShareRecordResult = {
  shares: RecordShare[];
  recipientCount: number;
  emailsQueued: number;
  emailStatus: RecordShareEmailStatus;
};

/** What is shared: a record by its kind and id, and where in the app the link opens. */
export type ShareRecordInput = {
  kind: string;
  id: string;
  userIds: string[];
  note: string;
  path: string;
};

type RequestOptions = { signal?: AbortSignal };

function shareOf(ref: FragmentType<typeof RecordShareFieldsFragmentDoc>): RecordShare {
  const share = getFragmentData(RecordShareFieldsFragmentDoc, ref);

  return {
    ...share,
    sharedWith: share.sharedWith
      ? getFragmentData(RecordShareUserFieldsFragmentDoc, share.sharedWith)
      : null,
    sharedBy: share.sharedBy
      ? getFragmentData(RecordShareUserFieldsFragmentDoc, share.sharedBy)
      : null,
  };
}

/** Who a record has been shared with, newest first. */
export async function fetchRecordShares(
  kind: string,
  id: string,
  options?: RequestOptions,
): Promise<RecordShare[]> {
  const data = await requestGraphQL({
    document: RecordSharesDocument,
    operationName: "RecordShares",
    variables: { kind, id },
    signal: options?.signal,
  });

  return data.recordShares.map(shareOf);
}

/** The teammates a record can be shared with: those whose role already lets them open it. */
export async function fetchRecordShareCandidates(
  kind: string,
  id: string,
  query: string,
  options?: RequestOptions,
): Promise<RecordShareUser[]> {
  const data = await requestGraphQL({
    document: RecordShareCandidatesDocument,
    operationName: "RecordShareCandidates",
    variables: { kind, id, query: query.trim() || null, first: 10 },
    signal: options?.signal,
  });

  return data.recordShareCandidates.map((user) =>
    getFragmentData(RecordShareUserFieldsFragmentDoc, user),
  );
}

/** Shares a record with teammates; each gets a notification and, where it is set up, an email. */
export async function shareRecord(input: ShareRecordInput): Promise<ShareRecordResult> {
  const data = await requestGraphQL({
    document: ShareRecordDocument,
    operationName: "ShareRecord",
    variables: { input: { ...input, note: input.note.trim() } },
  });
  const result = data.shareRecord;

  return {
    shares: result.shares.map(shareOf),
    recipientCount: result.recipientCount,
    emailsQueued: result.emailsQueued,
    emailStatus: result.emailStatus,
  };
}

/**
 * Removes one share of a record: its recipient no longer reads the record's
 * case through it. Returns who the record is still shared with.
 */
export async function unshareRecord(
  kind: string,
  id: string,
  shareId: string,
): Promise<RecordShare[]> {
  const data = await requestGraphQL({
    document: UnshareRecordDocument,
    operationName: "UnshareRecord",
    variables: { input: { kind, id, shareId } },
  });

  return data.unshareRecord.map(shareOf);
}
