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

export type InboundMessageRow = InboundMessageListFieldsFragment;
export type InboundMessageDetail = InboundMessageDetailFieldsFragment;
export type InboundMailbox = InboundMailboxFieldsFragment;
export type { InboundClassification, InboundMessageStatus };

type RequestOptions = { signal?: AbortSignal };

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
      getFragmentData(InboundMessageListFieldsFragmentDoc, edge.node),
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

  return getFragmentData(InboundMessageDetailFieldsFragmentDoc, data.inboundMessage);
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
    getFragmentData(InboundMailboxFieldsFragmentDoc, mailbox),
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

  return getFragmentData(InboundMessageDetailFieldsFragmentDoc, data.reviewInboundMessage);
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

  return getFragmentData(InboundMessageDetailFieldsFragmentDoc, data.linkInboundMessage);
}
