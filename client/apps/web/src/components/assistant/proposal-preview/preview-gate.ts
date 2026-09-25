import { GraphQLRequestError } from "@trenova/shared/lib/graphql";

/**
 * Whether a person may approve what a preview shows, and with which digest.
 *
 * - `loading`: nothing is on screen yet, so there is nothing to approve.
 * - `stale`: the record changed or went away since the proposal; the server
 *   refuses the approval, so it is not offered. Rejecting still is.
 * - `unavailable`: the preview could not be read. Approving is still allowed
 *   and goes without a digest, which the server records as not reviewed; the
 *   surface says so beside the button.
 * - `ready`: the digest names exactly what the person was shown.
 */
export type ApprovalGate =
  | { state: "loading" }
  | { state: "stale"; digest: string }
  | { state: "unavailable" }
  | { state: "ready"; digest: string };

type PreviewQueryState<T> = {
  data: T | undefined;
  isPending: boolean;
  isError: boolean;
};

export function approvalGate<T extends { digest: string; stale: boolean }>(
  query: PreviewQueryState<T>,
): ApprovalGate {
  // A preview already on screen stays the one approved against while a
  // refetch runs behind it; only a first read blocks the button.
  if (query.data !== undefined) {
    return query.data.stale
      ? { state: "stale", digest: query.data.digest }
      : { state: "ready", digest: query.data.digest };
  }
  if (query.isError) {
    return { state: "unavailable" };
  }

  return { state: "loading" };
}

export function canApprove(gate: ApprovalGate): boolean {
  return gate.state === "ready" || gate.state === "unavailable";
}

/** The digest a decision carries: what the person was shown, or nothing when nothing was. */
export function gateDigest(gate: ApprovalGate): string | undefined {
  return gate.state === "ready" || gate.state === "stale" ? gate.digest : undefined;
}

/**
 * The server refuses an approval whose digest no longer matches with a
 * conflict, and records nothing. That is the one refusal a surface answers by
 * showing the preview again rather than by reporting an error.
 */
export function isPreviewConflict(error: unknown): boolean {
  return error instanceof GraphQLRequestError && error.isConflictError();
}

/**
 * A preview of changed values that the server refused as invalid: the values
 * would not pass the tool's own checks, so approving them would be refused
 * too. Any other failure to read a preview leaves approval possible.
 */
export function isPreviewRefusal(error: unknown): boolean {
  return error instanceof GraphQLRequestError && error.isValidationError();
}

/**
 * The digests a batch approval sends: one for each proposal whose preview the
 * person was actually shown, in the order the batch runs. A proposal they
 * never opened goes without one and is recorded as approved unreviewed, which
 * is the truth; sending a digest fetched behind their back would claim a
 * review that did not happen.
 */
export function batchPreviewDigests(
  ids: readonly string[],
  shown: ReadonlyMap<string, string>,
): { proposalId: string; digest: string }[] {
  const digests: { proposalId: string; digest: string }[] = [];
  const seen = new Set<string>();
  for (const id of ids) {
    const digest = shown.get(id);
    if (digest === undefined || digest === "" || seen.has(id)) {
      continue;
    }
    seen.add(id);
    digests.push({ proposalId: id, digest });
  }

  return digests;
}
