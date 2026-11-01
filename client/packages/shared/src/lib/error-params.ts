import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { isRecord } from "@trenova/shared/lib/utils";

/**
 * The params a server error carries (a business rule's ids, an `overridable` flag),
 * whether it came back from a REST route or as a GraphQL error's extensions.
 */
export function errorParams(error: unknown): Record<string, string> {
  if (error instanceof ApiRequestError) {
    return error.getParams();
  }
  if (error instanceof GraphQLRequestError && isRecord(error.params)) {
    return Object.fromEntries(
      Object.entries(error.params).map(([key, value]) => [key, String(value)]),
    );
  }
  return {};
}
