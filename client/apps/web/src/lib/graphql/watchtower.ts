import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  HandOffWatchtowerItemDocument,
  MarkWatchtowerItemSeenDocument,
  ResolveWatchtowerItemDocument,
  RewordWatchtowerPlayDocument,
  SendWatchtowerPlayDocument,
  SnoozeWatchtowerItemDocument,
  WatchtowerAgentFieldsFragmentDoc,
  WatchtowerClearedDocument,
  WatchtowerCountsDocument,
  WatchtowerCountsFieldsFragmentDoc,
  WatchtowerFeedDocument,
  WatchtowerHandOffAgentsDocument,
  WatchtowerItemDetailDocument,
  WatchtowerItemFieldsFragmentDoc,
  WatchtowerPlayDocument,
  type WatchtowerPlayQuery,
  type WatchtowerAgentFieldsFragment,
  type WatchtowerAgentStatus,
  type WatchtowerAgentStep,
  type WatchtowerCountsFieldsFragment,
  type WatchtowerItemFieldsFragment,
  type WatchtowerItemStatus,
  type WatchtowerItemsInput,
  type WatchtowerResolutionCode,
  type WatchtowerSeverity,
  type WatchtowerSourceKind,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type WatchtowerItem = WatchtowerItemFieldsFragment;
export type WatchtowerCounts = WatchtowerCountsFieldsFragment;
export type WatchtowerAgent = WatchtowerAgentFieldsFragment;
export type WatchtowerAgentWork = NonNullable<WatchtowerItem["agentWork"]>;
export type WatchtowerFact = WatchtowerItem["facts"][number];
export type WatchtowerTrailEvent = WatchtowerItem["trail"][number];
/** What is happening on an item, typed for the view made for it. */
export type WatchtowerPanel = NonNullable<WatchtowerItem["panel"]>;
export type WatchtowerStopPanel = NonNullable<WatchtowerPanel["stop"]>;
export type WatchtowerClockPanel = NonNullable<WatchtowerPanel["clock"]>;
export type WatchtowerPaperPanel = NonNullable<WatchtowerPanel["paper"]>;
export type WatchtowerMessagePanel = NonNullable<WatchtowerPanel["message"]>;
export type WatchtowerExpiryPanel = NonNullable<WatchtowerPanel["expiry"]>;
export type WatchtowerCoveragePanel = NonNullable<WatchtowerPanel["coverage"]>;
export type WatchtowerContactPanel = NonNullable<WatchtowerPanel["contact"]>;
export type WatchtowerRunPanel = NonNullable<WatchtowerPanel["run"]>;
export type {
  WatchtowerAgentStatus,
  WatchtowerAgentStep,
  WatchtowerItemStatus,
  WatchtowerResolutionCode,
  WatchtowerSeverity,
  WatchtowerSourceKind,
};

type RequestOptions = { signal?: AbortSignal };

/** A write sent once: a retried request with the same key is replayed, not applied again. */
type WriteOptions = { idempotencyKey?: string };

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
  statuses?: WatchtowerItemStatus[];
  assignedToMe?: boolean;
  unassigned?: boolean;
  overdueOnly?: boolean;
  subjectId?: string | null;
  orderByDue?: boolean;
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
    statuses: filter.statuses ?? [],
    assignedToMe: filter.assignedToMe ?? false,
    unassigned: filter.unassigned ?? false,
    overdueOnly: filter.overdueOnly ?? false,
    subjectId: filter.subjectId ?? null,
    orderByDue: filter.orderByDue ?? false,
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

export async function fetchWatchtowerItem(
  id: string,
  options?: RequestOptions,
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: WatchtowerItemDetailDocument,
    operationName: "WatchtowerItemDetail",
    variables: { id },
    signal: options?.signal,
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.watchtowerItem);
}

/** How many items this reader resolved or snoozed since an instant. */
export async function fetchWatchtowerCleared(
  since: number,
  options?: RequestOptions,
): Promise<number> {
  const data = await requestGraphQL({
    document: WatchtowerClearedDocument,
    operationName: "WatchtowerCleared",
    variables: { since },
    signal: options?.signal,
  });

  return data.watchtowerCleared;
}

/** The agents an item can be handed to. */
export async function fetchWatchtowerHandOffAgents(
  options?: RequestOptions,
): Promise<WatchtowerAgent[]> {
  const data = await requestGraphQL({
    document: WatchtowerHandOffAgentsDocument,
    operationName: "WatchtowerHandOffAgents",
    variables: {},
    signal: options?.signal,
  });

  return data.watchtowerHandOffAgents.map((agent) =>
    getFragmentData(WatchtowerAgentFieldsFragmentDoc, agent),
  );
}

/** Records that this reader opened an item, so it stops showing as new to them. */
export async function markWatchtowerItemSeen(id: string): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: MarkWatchtowerItemSeenDocument,
    operationName: "MarkWatchtowerItemSeen",
    variables: { id },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.markWatchtowerItemSeen);
}

export type HandOffResult = {
  item: WatchtowerItem;
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
  options?: WriteOptions,
): Promise<HandOffResult> {
  const data = await requestGraphQL({
    document: HandOffWatchtowerItemDocument,
    operationName: "HandOffWatchtowerItem",
    variables: { id, input: { agentDefinitionId: agentDefinitionId ?? null } },
    idempotencyKey: options?.idempotencyKey,
  });

  const result = data.handOffWatchtowerItem;

  return {
    item: getFragmentData(WatchtowerItemFieldsFragmentDoc, result.item),
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

/**
 * The version a person was looking at travels with every change, so a
 * change made on a stale screen is refused rather than silently applied over
 * someone else's.
 */
type ExceptionChange = { id: string; version: number };

const noteOrNull = (note?: string): string | null => {
  const trimmed = note?.trim() ?? "";
  return trimmed === "" ? null : trimmed;
};

export async function snoozeWatchtowerItem(
  change: ExceptionChange & { until: number; note?: string },
  options?: WriteOptions,
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: SnoozeWatchtowerItemDocument,
    operationName: "SnoozeWatchtowerItem",
    variables: {
      id: change.id,
      input: {
        until: change.until,
        note: noteOrNull(change.note),
        expectedVersion: change.version,
      },
    },
    idempotencyKey: options?.idempotencyKey,
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.snoozeWatchtowerItem);
}

export async function resolveWatchtowerItem(
  change: ExceptionChange & { code: WatchtowerResolutionCode; note?: string },
  options?: WriteOptions,
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: ResolveWatchtowerItemDocument,
    operationName: "ResolveWatchtowerItem",
    variables: {
      id: change.id,
      input: { code: change.code, note: noteOrNull(change.note), expectedVersion: change.version },
    },
    idempotencyKey: options?.idempotencyKey,
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.resolveWatchtowerItem);
}

/**
 * An item's play as the reader would send it: who it goes to, how, and the
 * words. The server plans it from the record; only the words are the
 * reader's to change.
 */
export type WatchtowerPlay = WatchtowerPlayQuery["watchtowerPlay"];

/**
 * Plans the item's play. It rejects with the reason, in words for the
 * reader, when the play cannot be sent: nobody on the record can be reached,
 * it is already done, or it is not theirs to send.
 */
export async function fetchWatchtowerPlay(
  itemId: string,
  options?: RequestOptions,
): Promise<WatchtowerPlay> {
  const data = await requestGraphQL({
    document: WatchtowerPlayDocument,
    operationName: "WatchtowerPlay",
    variables: { itemId },
    signal: options?.signal,
  });

  return data.watchtowerPlay;
}

/** Has Nova rewrite the words in the box the way the reader asks. Nothing is sent. */
export async function rewordWatchtowerPlay(
  itemId: string,
  input: { message: string; guidance: string },
): Promise<string> {
  const data = await requestGraphQL({
    document: RewordWatchtowerPlayDocument,
    operationName: "RewordWatchtowerPlay",
    variables: { itemId, input },
  });

  return data.rewordWatchtowerPlay;
}

/** Sends the item's play under the reader's name, with the words they approved. */
export async function sendWatchtowerPlay(
  itemId: string,
  message: string,
  options?: WriteOptions,
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: SendWatchtowerPlayDocument,
    operationName: "SendWatchtowerPlay",
    variables: { itemId, input: { message: message === "" ? null : message } },
    idempotencyKey: options?.idempotencyKey,
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.sendWatchtowerPlay);
}
