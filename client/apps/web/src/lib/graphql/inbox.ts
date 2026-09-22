import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  InboundMailboxesDocument,
  InboundMailboxFieldsFragmentDoc,
  InboundMessageCountsDocument,
  InboundMessageDetailFieldsFragmentDoc,
  InboundMessageDocument,
  InboundMessageListFieldsFragmentDoc,
  InboundMessagesDocument,
  LinkInboundMessageDocument,
  ReviewInboundMessageDocument,
  type InboundClassification,
  type InboundMailboxFieldsFragment,
  type InboundMessageDetailFieldsFragment,
  type InboundMessageListFieldsFragment,
  type InboundMessageStatus,
  type InboundMessagesInput,
  type LinkInboundMessageInput,
  type ReviewInboundMessageInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { UnmaskFragments } from "@trenova/shared/types/graphql-connection";

/*
 * Unmasked, because the detail fragment spreads the list fragment and nests the
 * mailbox and attachment ones. getFragmentData unwraps the outermost mask at
 * runtime; the type has to be unwrapped all the way down or every field the
 * inner fragments contribute reads as missing.
 */
export type InboundMessageRow = UnmaskFragments<InboundMessageListFieldsFragment>;
export type InboundMessageDetail = UnmaskFragments<InboundMessageDetailFieldsFragment>;
export type InboundMailbox = UnmaskFragments<InboundMailboxFieldsFragment>;
export type { InboundClassification, InboundMessageStatus };

type RequestOptions = { signal?: AbortSignal };

/*
 * getFragmentData does the runtime unmasking — it hands back the same object
 * with the mask marker gone — but its return type is still the masked
 * fragment, and the masked type says nothing about the fields the nested
 * fragments contribute. The cast is the type catching up with what the call
 * already did; it is written once, here, rather than at every call site.
 */
function unmasked<T>(value: unknown): T {
  return value as T;
}


export type InboundMessagePage = {
  messages: InboundMessageRow[];
  endCursor: string | null;
  hasNextPage: boolean;
};

export type InboundMessageFilter = {
  statuses?: InboundMessageStatus[];
  classification?: InboundClassification | null;
  mailboxId?: string | null;
  shipmentId?: string | null;
  after?: string | null;
  first?: number;
};

export type InboundMessageCounts = {
  waiting: number;
  handled: number;
  ignored: number;
  quarantined: number;
  total: number;
};

/** How much of the inbox one request asks for. */
export const INBOX_PAGE_SIZE = 25;

export async function fetchInboundMessages(
  filter: InboundMessageFilter = {},
  options?: RequestOptions,
): Promise<InboundMessagePage> {
  const input: InboundMessagesInput = {
    first: filter.first ?? INBOX_PAGE_SIZE,
    after: filter.after ?? null,
    statuses: filter.statuses ?? [],
    classification: filter.classification ?? null,
    mailboxId: filter.mailboxId ?? null,
    shipmentId: filter.shipmentId ?? null,
  };

  const data = await requestGraphQL({
    document: InboundMessagesDocument,
    operationName: "InboundMessages",
    variables: { input },
    signal: options?.signal,
  });

  return {
    messages: data.inboundMessages.edges.map((edge) =>
      unmasked<InboundMessageRow>(
        getFragmentData(InboundMessageListFieldsFragmentDoc, edge.node),
      ),
    ),
    endCursor: data.inboundMessages.pageInfo.endCursor ?? null,
    hasNextPage: data.inboundMessages.pageInfo.hasNextPage,
  };
}

export async function fetchInboundMessage(
  id: string,
  options?: RequestOptions,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: InboundMessageDocument,
    operationName: "InboundMessage",
    variables: { id },
    signal: options?.signal,
  });

  return unmasked<InboundMessageDetail>(
    getFragmentData(InboundMessageDetailFieldsFragmentDoc, data.inboundMessage),
  );
}

export async function fetchInboundMessageCounts(
  options?: RequestOptions,
): Promise<InboundMessageCounts> {
  const data = await requestGraphQL({
    document: InboundMessageCountsDocument,
    operationName: "InboundMessageCounts",
    variables: {},
    signal: options?.signal,
  });

  return data.inboundMessageCounts;
}

export async function fetchInboundMailboxes(
  options?: RequestOptions,
): Promise<InboundMailbox[]> {
  const data = await requestGraphQL({
    document: InboundMailboxesDocument,
    operationName: "InboundMailboxes",
    variables: {},
    signal: options?.signal,
  });

  return data.inboundMailboxes.map((mailbox) =>
    unmasked<InboundMailbox>(getFragmentData(InboundMailboxFieldsFragmentDoc, mailbox)),
  );
}

export async function reviewInboundMessage(
  id: string,
  input: ReviewInboundMessageInput,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: ReviewInboundMessageDocument,
    operationName: "ReviewInboundMessage",
    variables: { id, input },
  });

  return unmasked<InboundMessageDetail>(
    getFragmentData(InboundMessageDetailFieldsFragmentDoc, data.reviewInboundMessage),
  );
}

export async function linkInboundMessage(
  id: string,
  input: LinkInboundMessageInput,
): Promise<InboundMessageDetail> {
  const data = await requestGraphQL({
    document: LinkInboundMessageDocument,
    operationName: "LinkInboundMessage",
    variables: { id, input },
  });

  return unmasked<InboundMessageDetail>(
    getFragmentData(InboundMessageDetailFieldsFragmentDoc, data.linkInboundMessage),
  );
}
