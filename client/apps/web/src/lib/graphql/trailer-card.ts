import {
  DeskTrailerCardDocument,
  type DeskTrailerCardQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskTrailer = NonNullable<DeskTrailerCardQuery["trailer"]>;

/**
 * One trailer as the Desk's card reads it. A trailer the server does not
 * return is an error rather than an empty card, so the card says it could not
 * be read.
 */
export async function fetchDeskTrailer(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskTrailer> {
  const data = await requestGraphQL({
    document: DeskTrailerCardDocument,
    operationName: "DeskTrailerCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.trailer) {
    throw new Error("Trailer not found");
  }
  return data.trailer;
}
