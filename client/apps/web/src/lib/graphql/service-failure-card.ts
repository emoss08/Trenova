import {
  DeskServiceFailureCardDocument,
  type DeskServiceFailureCardQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskServiceFailure = NonNullable<DeskServiceFailureCardQuery["serviceFailure"]>;

/**
 * One service failure as the Desk's card reads it, with the stop it happened
 * at, its load, its reason and the people who have handled it. A failure the
 * server does not return is an error rather than an empty card, so the card
 * says it could not be read.
 */
export async function fetchDeskServiceFailure(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskServiceFailure> {
  const data = await requestGraphQL({
    document: DeskServiceFailureCardDocument,
    operationName: "DeskServiceFailureCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.serviceFailure) {
    throw new Error("Service failure not found");
  }
  return data.serviceFailure;
}
