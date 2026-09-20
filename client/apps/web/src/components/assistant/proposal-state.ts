import type { AssistantMessage, AssistantProposal } from "@/types/assistant";

/**
 * What a proposal card should show.
 *
 * - `awaiting` — nobody has decided; this is the only state with buttons
 * - `running` — approved, but the tool has not reported back yet
 * - `done` — the tool ran
 * - `failed` — approved, and the tool refused or errored
 * - `declined` — a person rejected it
 * - `closed` — it stopped being actionable without anyone deciding
 */
export type ProposalPresentation =
  | "awaiting"
  | "running"
  | "done"
  | "failed"
  | "declined"
  | "closed";

/**
 * Decides how to present one proposal.
 *
 * Execution is read before status, and that ordering carries the whole point of
 * the card. "Accepted" only ever means a person said yes; whether the write
 * happened is `executedAt` and `executionError`. Showing an approved-but-failed
 * proposal as approved would tell someone their instruction was carried out when
 * it was refused, which is the exact failure the execution columns exist to make
 * visible.
 *
 * An approval that has not produced either outcome yet is `running` rather than
 * `done`, because the honest answer at that moment is "not yet".
 */
export function classifyProposal(
  proposal: AssistantProposal,
  now: number = Date.now() / 1000,
): ProposalPresentation {
  if (proposal.executionError !== "") {
    return "failed";
  }

  // The server sweeps expired proposals every quarter hour; between sweeps
  // the clock decides. A card offering buttons for a proposal the server will
  // refuse is a promise the click breaks.
  if (
    proposal.status === "Pending" &&
    (proposal.expiresAt ?? 0) > 0 &&
    proposal.expiresAt! <= now
  ) {
    return "closed";
  }

  if (proposal.status === "ExecutionFailed") {
    return "failed";
  }

  if (proposal.status === "Executed" || (proposal.executedAt ?? 0) > 0) {
    return "done";
  }

  switch (proposal.status) {
    case "Pending":
      return "awaiting";
    case "Accepted":
    case "Modified":
      return "running";
    case "Rejected":
      return "declined";
    default:
      return "closed";
  }
}

/**
 * Whether a proposal can still be decided.
 *
 * The server is the authority — approving a proposal someone else already
 * resolved fails there — so this only decides whether to offer the buttons.
 */
export function isDecidable(proposal: AssistantProposal): boolean {
  return classifyProposal(proposal) === "awaiting";
}

/**
 * Groups proposals under the assistant message that asked for them.
 *
 * A proposal whose source message is missing from the thread is not dropped: it
 * comes back under `orphans` and is rendered at the end, because a pending write
 * nobody can see is worse than one shown out of position. That happens when the
 * message list is truncated, or when the tool call could not be matched to a
 * saved turn.
 */
export function groupProposalsByMessage(
  proposals: readonly AssistantProposal[],
  messages: readonly AssistantMessage[],
): { byMessage: Map<string, AssistantProposal[]>; orphans: AssistantProposal[] } {
  const knownMessageIds = new Set(messages.map((message) => message.id));
  const byMessage = new Map<string, AssistantProposal[]>();
  const orphans: AssistantProposal[] = [];

  for (const proposal of proposals) {
    if (proposal.sourceMessageId === "" || !knownMessageIds.has(proposal.sourceMessageId)) {
      orphans.push(proposal);
      continue;
    }

    const existing = byMessage.get(proposal.sourceMessageId);
    if (existing) {
      existing.push(proposal);
    } else {
      byMessage.set(proposal.sourceMessageId, [proposal]);
    }
  }

  return { byMessage, orphans };
}

/**
 * Turns a tool's snake_case name into something readable.
 *
 * Tool names are identifiers chosen for the model, and showing `reassign_move`
 * to a dispatcher deciding whether to allow it is needlessly unfriendly.
 */
export function humanizeToolName(toolName: string): string {
  const words = toolName.split(/[_\-.]+/u).filter((word) => word !== "");
  if (words.length === 0) {
    return toolName;
  }

  return words
    .map((word, index) =>
      index === 0 ? word.charAt(0).toUpperCase() + word.slice(1) : word.toLowerCase(),
    )
    .join(" ");
}

/**
 * Flattens a proposal's arguments into rows an approver can read.
 *
 * Values are stringified rather than rendered structurally: the approver needs to
 * see exactly what would be sent, and a nested object shown as `[object Object]`
 * hides precisely the part worth checking.
 */
export function argumentRows(
  argumentsValue: AssistantProposal["arguments"],
): { key: string; value: string }[] {
  if (!argumentsValue) {
    return [];
  }

  return Object.entries(argumentsValue)
    .map(([key, value]) => ({ key, value: stringifyArgument(value) }))
    .sort((left, right) => left.key.localeCompare(right.key));
}

function stringifyArgument(value: unknown): string {
  if (value === null || value === undefined) {
    return "—";
  }

  if (typeof value === "string") {
    return value === "" ? "—" : value;
  }

  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  return JSON.stringify(value);
}
