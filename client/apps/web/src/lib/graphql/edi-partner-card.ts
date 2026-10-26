import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DeskEdiPartnerCardDocument,
  DeskEdiPartnerMessageFieldsFragmentDoc,
  DeskEdiPartnerMessagesDocument,
  type DeskEdiPartnerCardQuery,
  type DeskEdiPartnerMessageFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RawEdiPartnerCard = DeskEdiPartnerCardQuery["ediPartnerCard"];

/** The partner card as the server sends it, with the messages it lists read out of their fragment. */
export type DeskEdiPartnerCard = Omit<RawEdiPartnerCard, "attention"> & {
  attention: Omit<RawEdiPartnerCard["attention"], "overdueAcks" | "deadLettered"> & {
    overdueAcks: DeskEdiPartnerMessageFieldsFragment[];
    deadLettered: DeskEdiPartnerMessageFieldsFragment[];
  };
};
export type DeskEdiPartner = DeskEdiPartnerCard["partner"];
export type EdiPartnerDocumentProfile = DeskEdiPartnerCard["documentProfiles"][number];
export type EdiPartnerTransport = DeskEdiPartnerCard["transports"][number];
export type EdiPartnerActivity = DeskEdiPartnerCard["activity"];
export type EdiPartnerAttention = DeskEdiPartnerCard["attention"];
export type EdiPartnerQuarantinedFile = EdiPartnerAttention["quarantinedFiles"][number];
export type EdiPartnerCarrier = NonNullable<DeskEdiPartnerCard["carriers"]>[number];
export type EdiPartnerMessage = DeskEdiPartnerMessageFieldsFragment;

/** One page of a partner's messages. `total` is read with the first page only. */
export type EdiPartnerMessagesPage = {
  rows: EdiPartnerMessage[];
  total: number | null;
  /** False when total is the server's estimate of a list too large to count. */
  totalExact?: boolean;
  endCursor: string | null;
  hasNextPage: boolean;
};

/**
 * Everything the Desk's EDI partner card draws, read in one request. The server
 * refuses a partner the person may not read, and the card says it could not be
 * read.
 */
export async function fetchDeskEdiPartner(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskEdiPartnerCard> {
  const data = await requestGraphQL({
    document: DeskEdiPartnerCardDocument,
    operationName: "DeskEdiPartnerCard",
    variables: { id },
    signal: options?.signal,
  });
  const card = data.ediPartnerCard;
  return {
    ...card,
    attention: {
      ...card.attention,
      overdueAcks: getFragmentData(
        DeskEdiPartnerMessageFieldsFragmentDoc,
        card.attention.overdueAcks,
      ),
      deadLettered: getFragmentData(
        DeskEdiPartnerMessageFieldsFragmentDoc,
        card.attention.deadLettered,
      ),
    },
  };
}

/** The partner's messages both ways, newest first, a page at a time. */
export async function fetchEdiPartnerMessages(
  partnerId: string,
  page: { first: number; after: string | null },
  options?: { signal?: AbortSignal },
): Promise<EdiPartnerMessagesPage> {
  const firstPage = page.after === null;
  const data = await requestGraphQL({
    document: DeskEdiPartnerMessagesDocument,
    operationName: "DeskEdiPartnerMessages",
    variables: {
      partnerId,
      input: {
        first: page.first,
        after: page.after,
        sort: [{ field: "generatedAt", direction: "desc" }],
      },
      includeTotalCount: firstPage,
    },
    signal: options?.signal,
  });
  const rows = getFragmentData(
    DeskEdiPartnerMessageFieldsFragmentDoc,
    data.ediMessages.edges.map((edge) => edge.node),
  );
  return {
    rows,
    total: firstPage ? (data.ediMessages.totalCount ?? rows.length) : null,
    totalExact: data.ediMessages.totalCount == null || data.ediMessages.totalCountExact !== false,
    endCursor: data.ediMessages.pageInfo.endCursor ?? null,
    hasNextPage: data.ediMessages.pageInfo.hasNextPage,
  };
}
