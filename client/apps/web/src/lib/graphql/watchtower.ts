import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DismissWatchtowerItemDocument,
  HandOffWatchtowerItemDocument,
  MarkWatchtowerSeenDocument,
  WatchtowerCountsDocument,
  WatchtowerCountsFieldsFragmentDoc,
  WatchtowerFeedDocument,
  WatchtowerItemFieldsFragmentDoc,
  type WatchtowerCountsFieldsFragment,
  type WatchtowerItemFieldsFragment,
  type WatchtowerItemsInput,
  type WatchtowerSeverity,
  type WatchtowerSourceKind,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type WatchtowerItem = WatchtowerItemFieldsFragment;
export type WatchtowerCounts = WatchtowerCountsFieldsFragment;
export type { WatchtowerSeverity, WatchtowerSourceKind };

type RequestOptions = { signal?: AbortSignal };

export type WatchtowerPage = {
  items: WatchtowerItem[];
  endCursor: string | null;
  hasNextPage: boolean;
  /** Where this reader's eye last was, so the feed can draw the unseen line. */
  seenAt: number;
};

export type WatchtowerFilter = {
  kinds?: WatchtowerSourceKind[];
  severities?: WatchtowerSeverity[];
  unresolvedOnly?: boolean;
  after?: string | null;
  first?: number;
};

/** How much of the feed one request asks for. */
export const WATCHTOWER_PAGE_SIZE = 25;

export async function fetchWatchtowerFeed(
  filter: WatchtowerFilter = {},
  options?: RequestOptions,
): Promise<WatchtowerPage> {
  const input: WatchtowerItemsInput = {
    first: filter.first ?? WATCHTOWER_PAGE_SIZE,
    after: filter.after ?? null,
    kinds: filter.kinds ?? [],
    severities: filter.severities ?? [],
    unresolvedOnly: filter.unresolvedOnly ?? true,
  };

  const data = await requestGraphQL({
    document: WatchtowerFeedDocument,
    operationName: "WatchtowerFeed",
    variables: { input },
    signal: options?.signal,
  });

  return {
    items: data.watchtowerItems.edges.map((edge) =>
      getFragmentData(WatchtowerItemFieldsFragmentDoc, edge.node),
    ),
    endCursor: data.watchtowerItems.pageInfo.endCursor ?? null,
    hasNextPage: data.watchtowerItems.pageInfo.hasNextPage,
    seenAt: data.watchtowerItems.seenAt,
  };
}

export async function fetchWatchtowerCounts(options?: RequestOptions): Promise<WatchtowerCounts> {
  const data = await requestGraphQL({
    document: WatchtowerCountsDocument,
    operationName: "WatchtowerCounts",
    variables: {},
    signal: options?.signal,
  });

  return getFragmentData(WatchtowerCountsFieldsFragmentDoc, data.watchtowerCounts);
}

/**
 * Moves the reader's cursor. Passing no instant means now, which is what
 * opening the feed does.
 */
export async function markWatchtowerSeen(seenAt?: number): Promise<WatchtowerCounts> {
  const data = await requestGraphQL({
    document: MarkWatchtowerSeenDocument,
    operationName: "MarkWatchtowerSeen",
    variables: { seenAt: seenAt ?? null },
  });

  return getFragmentData(WatchtowerCountsFieldsFragmentDoc, data.markWatchtowerSeen);
}

export async function dismissWatchtowerItem(id: string): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: DismissWatchtowerItemDocument,
    operationName: "DismissWatchtowerItem",
    variables: { id },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.dismissWatchtowerItem);
}

export type HandOffResult = {
  runId: string | null;
  subscribers: { id: string; name: string }[];
  candidates: {
    id: string;
    name: string;
    icon?: string | null;
    accent?: string | null;
    template?: string | null;
  }[];
  templates: string[];
};

/**
 * Hands an item to an agent, or to whoever subscribes to its event.
 *
 * With no agent named and nobody subscribing, the answer is the agents that
 * could take it — so the person picks rather than being told nothing
 * happened.
 */
export async function handOffWatchtowerItem(
  id: string,
  agentDefinitionId?: string,
): Promise<HandOffResult> {
  const data = await requestGraphQL({
    document: HandOffWatchtowerItemDocument,
    operationName: "HandOffWatchtowerItem",
    variables: { id, input: { agentDefinitionId: agentDefinitionId ?? null } },
  });

  const result = data.handOffWatchtowerItem;

  return {
    runId: result.run?.id ?? null,
    subscribers: result.subscribers.map((agent) => ({ id: agent.id, name: agent.name })),
    candidates: result.candidates.map((agent) => ({
      id: agent.id,
      name: agent.name,
      icon: agent.icon,
      accent: agent.accent,
      template: agent.template,
    })),
    templates: result.templates,
  };
}
