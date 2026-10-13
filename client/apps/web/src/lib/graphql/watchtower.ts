import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AcknowledgeWatchtowerItemDocument,
  AssignWatchtowerItemDocument,
  CommentOnWatchtowerItemDocument,
  DismissWatchtowerItemDocument,
  HandOffWatchtowerItemDocument,
  MarkWatchtowerSeenDocument,
  ReopenWatchtowerItemDocument,
  ResolveWatchtowerItemDocument,
  SetWatchtowerItemDueDocument,
  SnoozeWatchtowerItemDocument,
  StartWatchtowerItemDocument,
  WatchtowerCountsDocument,
  WatchtowerCountsFieldsFragmentDoc,
  WatchtowerFeedDocument,
  WatchtowerItemDetailDocument,
  WatchtowerItemEventFieldsFragmentDoc,
  WatchtowerItemFieldsFragmentDoc,
  WatchtowerItemHistoryDocument,
  WatchtowerResolutionCodesDocument,
  type WatchtowerCountsFieldsFragment,
  type WatchtowerItemEventFieldsFragment,
  type WatchtowerItemEventKind,
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
export type WatchtowerItemEvent = WatchtowerItemEventFieldsFragment;
export type {
  WatchtowerItemEventKind,
  WatchtowerItemStatus,
  WatchtowerResolutionCode,
  WatchtowerSeverity,
  WatchtowerSourceKind,
};

export type WatchtowerResolutionOption = {
  code: WatchtowerResolutionCode;
  label: string;
  requiresNote: boolean;
};

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

/** An item's history, oldest first, as the server keeps it: never edited. */
export async function fetchWatchtowerItemHistory(
  id: string,
  options?: RequestOptions,
): Promise<WatchtowerItemEvent[]> {
  const data = await requestGraphQL({
    document: WatchtowerItemHistoryDocument,
    operationName: "WatchtowerItemHistory",
    variables: { id, limit: null },
    signal: options?.signal,
  });

  return data.watchtowerItemHistory.map((event) =>
    getFragmentData(WatchtowerItemEventFieldsFragmentDoc, event),
  );
}

export async function fetchWatchtowerResolutionCodes(
  options?: RequestOptions,
): Promise<WatchtowerResolutionOption[]> {
  const data = await requestGraphQL({
    document: WatchtowerResolutionCodesDocument,
    operationName: "WatchtowerResolutionCodes",
    variables: {},
    signal: options?.signal,
  });

  return data.watchtowerResolutionCodes.map((option) => ({
    code: option.code,
    label: option.label,
    requiresNote: option.requiresNote,
  }));
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

export async function assignWatchtowerItem(
  change: ExceptionChange & { assigneeId: string | null; note?: string },
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: AssignWatchtowerItemDocument,
    operationName: "AssignWatchtowerItem",
    variables: {
      id: change.id,
      input: {
        assigneeId: change.assigneeId,
        note: noteOrNull(change.note),
        expectedVersion: change.version,
      },
    },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.assignWatchtowerItem);
}

export async function acknowledgeWatchtowerItem(change: ExceptionChange): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: AcknowledgeWatchtowerItemDocument,
    operationName: "AcknowledgeWatchtowerItem",
    variables: { id: change.id, expectedVersion: change.version },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.acknowledgeWatchtowerItem);
}

export async function startWatchtowerItem(change: ExceptionChange): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: StartWatchtowerItemDocument,
    operationName: "StartWatchtowerItem",
    variables: { id: change.id, expectedVersion: change.version },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.startWatchtowerItem);
}

export async function snoozeWatchtowerItem(
  change: ExceptionChange & { until: number; note?: string },
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
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.snoozeWatchtowerItem);
}

export async function setWatchtowerItemDue(
  change: ExceptionChange & { dueAt: number | null; note?: string },
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: SetWatchtowerItemDueDocument,
    operationName: "SetWatchtowerItemDue",
    variables: {
      id: change.id,
      input: {
        dueAt: change.dueAt,
        note: noteOrNull(change.note),
        expectedVersion: change.version,
      },
    },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.setWatchtowerItemDue);
}

export async function commentOnWatchtowerItem(
  id: string,
  body: string,
): Promise<WatchtowerItemEvent> {
  const data = await requestGraphQL({
    document: CommentOnWatchtowerItemDocument,
    operationName: "CommentOnWatchtowerItem",
    variables: { id, body },
  });

  return getFragmentData(WatchtowerItemEventFieldsFragmentDoc, data.commentOnWatchtowerItem);
}

export async function resolveWatchtowerItem(
  change: ExceptionChange & { code: WatchtowerResolutionCode; note?: string },
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: ResolveWatchtowerItemDocument,
    operationName: "ResolveWatchtowerItem",
    variables: {
      id: change.id,
      input: { code: change.code, note: noteOrNull(change.note), expectedVersion: change.version },
    },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.resolveWatchtowerItem);
}

export async function reopenWatchtowerItem(
  change: ExceptionChange & { note?: string },
): Promise<WatchtowerItem> {
  const data = await requestGraphQL({
    document: ReopenWatchtowerItemDocument,
    operationName: "ReopenWatchtowerItem",
    variables: {
      id: change.id,
      input: { note: noteOrNull(change.note), expectedVersion: change.version },
    },
  });

  return getFragmentData(WatchtowerItemFieldsFragmentDoc, data.reopenWatchtowerItem);
}
