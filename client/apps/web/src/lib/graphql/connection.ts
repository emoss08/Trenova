import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import type { TypedDocumentString } from "@trenova/graphql/generated/graphql";

/**
 * A page of a cursor connection as a wrapper returns it: the server's `edges`, `pageInfo` and,
 * when the operation selects it, `totalCount`. `useWorkerList` reads it.
 */
export type Connection<Row> = {
  edges?: readonly { node: Row }[] | null;
  pageInfo: { hasNextPage: boolean; endCursor?: string | null };
  totalCount?: number | null;
};

/** A connection whose nodes are one fragment spread, as the generated query type masks it. */
type MaskedConnection<Row> = {
  edges?: readonly { node: FragmentType<TypedDocumentString<Row, unknown>> }[] | null;
  pageInfo: { hasNextPage: boolean; endCursor?: string | null };
  totalCount?: number | null;
};

/**
 * A connection whose nodes spread one fragment, with each node read through that fragment: the
 * one checked way from the generated (masked) query type to `Connection<Row>`. Every connection
 * wrapper whose operation spreads a fragment in its nodes returns through this.
 *
 *   return unmaskConnection(WorkerTrainingRecordFieldsFragmentDoc, data.workerTrainingRecords);
 */
export function unmaskConnection<Row>(
  fragment: TypedDocumentString<Row, unknown>,
  page: MaskedConnection<Row>,
): Connection<Row> {
  return {
    pageInfo: page.pageInfo,
    totalCount: page.totalCount,
    edges: page.edges?.map((edge) => ({ node: getFragmentData(fragment, edge.node) })),
  };
}
