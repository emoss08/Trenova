import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskActivityDocument,
  DeskAttentionDocument,
  DeskAttentionGroupFieldsFragmentDoc,
  DeskRecordFieldsFragmentDoc,
  WatchtowerAgentFieldsFragmentDoc,
  WatchtowerItemFieldsFragmentDoc,
  type DeskActivityOutcome,
  type DeskAttentionGroupFieldsFragment,
  type DeskLens,
  type DeskRecordFieldsFragment,
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

export type DeskAttention = {
  groups: DeskAttentionGroup[];
  counts: DeskLensCounts;
  hasMore: boolean;
  seenAt: number;
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
  first?: number;
  recordId?: string | null;
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
        first: filter.first ?? null,
        recordId: filter.recordId ?? null,
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
    hasMore: result.hasMore,
    seenAt: result.seenAt,
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
    entries: data.deskActivity.entries.map((entry) => ({
      ...entry,
      agent: entry.agent ? getFragmentData(WatchtowerAgentFieldsFragmentDoc, entry.agent) : null,
      records: entry.records.map((record) => getFragmentData(DeskRecordFieldsFragmentDoc, record)),
      planId: entry.planId ?? null,
      decidedBy: entry.decidedBy ?? null,
    })),
  };
}
