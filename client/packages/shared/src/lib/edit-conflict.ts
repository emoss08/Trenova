import type { EditConflict } from "@trenova/shared/types/errors";
import { ApiRequestError } from "./api";
import { GraphQLRequestError } from "./graphql";

/**
 * The conflict a save that lost a race came back with, whichever transport carried it.
 * Null for any other failure, and for a 409 from a server that did not say what changed.
 */
export function editConflictOf(error: unknown): EditConflict | null {
  if (error instanceof ApiRequestError || error instanceof GraphQLRequestError) {
    return error.getConflict();
  }
  return null;
}
