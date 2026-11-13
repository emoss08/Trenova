import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskActivityDocument,
  DeskActivityEntryDocument,
  DeskActivityEntryFieldsFragmentDoc,
  DeskAttentionDocument,
  DeskAttentionGroupFieldsFragmentDoc,
  DeskRecordFieldsFragmentDoc,
  WatchtowerAgentFieldsFragmentDoc,
  WatchtowerItemFieldsFragmentDoc,
  type DeskActivityEntryFieldsFragment,
  type DeskActivityOutcome,
  type DeskAttentionGroupFieldsFragment,
  type DeskLens,
  type DeskRecordFieldsFragment,
  type WatchtowerSeverity,
  type WatchtowerSourceKind,
  type WatchtowerItemFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { WatchtowerAgent } from "@/lib/graphql/watchtower";

export type { DeskActivityOutcome, DeskLens };
export type DeskRecord = DeskRecordFieldsFragment;

export type DeskAttentionItem = WatchtowerItemFieldsFragment & {
  deskLens: DeskLens;
  aboutId: string | null;
};

export type DeskAttentionGroup = Omit<DeskAttentionGroupFieldsFragment, "record" | "items"> & {
  record: DeskRecord | null;
  items: DeskAttentionItem[];
};

export type DeskLensCounts = {
  everything: number;
  watchtower: number;
  triage: number;
  decisions: number;
};

/** Open items of one kind the reader may see, which the list's filter offers the kind by. */
export type DeskAttentionKind = {
  kind: WatchtowerSourceKind;
  label: string;
  lens: DeskLens;
  count: number;
};

export type DeskAttention = {
  groups: DeskAttentionGroup[];
  counts: DeskLensCounts;
  /** Kinds with something open in the list's own scope, most first. */
  kinds: DeskAttentionKind[];
  hasMore: boolean;
  seenAt: number;
  /** The reader may see everyone's cases, so the list can offer only their own. */
  seesEveryone: boolean;
};

export type DeskActivityEntry = {
  id: string;
  outcome: DeskActivityOutcome;
  at: number;
  agent: WatchtowerAgent | null;
  runId: string;
  planId: string | null;
  title: string;
  detail: string;
  count: number;
  records: DeskRecord[];
  recordsMore: number;
  decidedBy: { id: string; name: string } | null;
  proposalIds: string[];
};

export type DeskActivity = { entries: DeskActivityEntry[]; hasMore: boolean };

type RequestOptions = { signal?: AbortSignal };

export type DeskAttentionFilter = {
  lens?: DeskLens;
  /** Rows with an item of one of these kinds, within the lens. */
  kinds?: readonly WatchtowerSourceKind[];
  /** Rows with an item of one of these severities. */
  severities?: readonly WatchtowerSeverity[];
  first?: number;
  recordId?: string | null;
  /** Only rows with an item an agent's run is working on right now. */
  working?: boolean;
  /** Only the reader's own cases and the ones nobody owns. */
  mine?: boolean;
  /** Rows with an item whose title, summary or record name holds these words. */
  search?: string;
  /** Only rows with an item the reader has not opened yet. */
  unseen?: boolean;
};

export async function fetchDeskAttention(
  filter: DeskAttentionFilter = {},
  options?: RequestOptions,
): Promise<DeskAttention> {
  const data = await requestGraphQL({
    document: DeskAttentionDocument,
    operationName: "DeskAttention",
    variables: {
      input: {
        lens: filter.lens ?? "Everything",
        kinds: filter.kinds && filter.kinds.length > 0 ? [...filter.kinds] : null,
        severities:
          filter.severities && filter.severities.length > 0 ? [...filter.severities] : null,
        first: filter.first ?? null,
        recordId: filter.recordId ?? null,
        working: filter.working ?? false,
        mine: filter.mine ?? false,
        search: filter.search?.trim() || null,
        unseen: filter.unseen ?? false,
      },
    },
    signal: options?.signal,
  });
  const result = data.deskAttention;

  return {
    groups: result.groups.map((ref) => {
      const group = getFragmentData(DeskAttentionGroupFieldsFragmentDoc, ref);
      return {
        ...group,
        record: group.record ? getFragmentData(DeskRecordFieldsFragmentDoc, group.record) : null,
        items: group.items.map((item) => ({
          ...getFragmentData(WatchtowerItemFieldsFragmentDoc, item),
          deskLens: item.deskLens,
          aboutId: item.aboutId ?? null,
        })),
      };
    }),
    counts: result.counts,
    kinds: result.kinds,
    hasMore: result.hasMore,
    seenAt: result.seenAt,
    seesEveryone: result.seesEveryone,
  };
}

function activityEntry(ref: DeskActivityEntryFieldsFragment): DeskActivityEntry {
  return {
    ...ref,
    agent: ref.agent ? getFragmentData(WatchtowerAgentFieldsFragmentDoc, ref.agent) : null,
    records: ref.records.map((record) => getFragmentData(DeskRecordFieldsFragmentDoc, record)),
    planId: ref.planId ?? null,
    decidedBy: ref.decidedBy ?? null,
  };
}

export async function fetchDeskActivity(
  since: number,
  first?: number,
  options?: RequestOptions,
): Promise<DeskActivity> {
  const data = await requestGraphQL({
    document: DeskActivityDocument,
    operationName: "DeskActivity",
    variables: { since, first: first ?? null },
    signal: options?.signal,
  });

  return {
    hasMore: data.deskActivity.hasMore,
    entries: data.deskActivity.entries.map((entry) =>
      activityEntry(getFragmentData(DeskActivityEntryFieldsFragmentDoc, entry)),
    ),
  };
}

export type DeskActivityChange = {
  proposalId: string;
  record: DeskRecord | null;
  what: string;
  why: string;
  proposedAt: number;
  at: number;
  outcome: DeskActivityOutcome;
  error: string;
};

export type DeskActivityDetail = {
  entry: DeskActivityEntry;
  tool: string;
  changes: DeskActivityChange[];
  ranBefore: number;
  ranSince: number;
};

/** One line of Nova handled, by the id of any write it folds. */
export async function fetchDeskActivityEntry(
  id: string,
  options?: RequestOptions,
): Promise<DeskActivityDetail> {
  const data = await requestGraphQL({
    document: DeskActivityEntryDocument,
    operationName: "DeskActivityEntry",
    variables: { id },
    signal: options?.signal,
  });
  const detail = data.deskActivityEntry;

  return {
    entry: activityEntry(getFragmentData(DeskActivityEntryFieldsFragmentDoc, detail.entry)),
    tool: detail.tool,
    changes: detail.changes.map((change) => ({
      ...change,
      record: change.record ? getFragmentData(DeskRecordFieldsFragmentDoc, change.record) : null,
    })),
    ranBefore: detail.ranBefore,
    ranSince: detail.ranSince,
  };
}
