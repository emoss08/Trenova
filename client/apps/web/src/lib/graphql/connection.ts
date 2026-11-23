/**
 * A page of a cursor connection as a wrapper returns it: the server's `edges`, `pageInfo` and,
 * when the operation selects it, `totalCount`, unmapped. `useWorkerList` reads it.
 */
export type Connection<Row> = {
  edges?: readonly { node: Row }[] | null;
  pageInfo: { hasNextPage: boolean; endCursor?: string | null };
  totalCount?: number | null;
};
