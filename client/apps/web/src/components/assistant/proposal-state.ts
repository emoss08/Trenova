import type { AssistantMessage, AssistantPlan, AssistantProposal } from "@/types/assistant";

/**
 * What a proposal card should show.
 *
 * - `awaiting` — nobody has decided; this is the only state with buttons
 * - `held` — nobody has decided, and a shadow switch keeps anyone from doing so
 * - `running` — approved, but the tool has not reported back yet
 * - `done` — the tool ran
 * - `failed` — approved, and the tool refused or errored
 * - `declined` — a person rejected it
 * - `simulated` — cleared while the agent was in simulation: previewed, never made
 * - `closed` — it stopped being actionable without anyone deciding
 */
export type ProposalPresentation =
  | "awaiting"
  | "held"
  | "running"
  | "done"
  | "failed"
  | "declined"
  | "simulated"
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

  // A simulation is neither done nor running: the approval happened and the
  // change did not, on purpose. It is read before the timestamp checks
  // because a simulated proposal never carries an execution.
  if (proposal.status === "Simulated" || (proposal.simulatedAt ?? 0) > 0) {
    return "simulated";
  }

  if (proposal.status === "Executed" || (proposal.executedAt ?? 0) > 0) {
    return "done";
  }

  switch (proposal.status) {
    case "Pending":
      // The server refuses every decision while a shadow switch is on. A
      // card with buttons would only teach that by failing the click.
      return proposal.hold ? "held" : "awaiting";
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
 * Whether the write ran without waiting on anyone: a call the agent may make
 * on its own, such as a report saved to the person's own list. It is recorded
 * like any other so the thread can say what it did, but it was never a
 * question, and must not read as one that was approved.
 */
export function ranWithoutApproval(proposal: AssistantProposal): boolean {
  return proposal.autonomyTier === "AutoExecute";
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
 * Whether a proposal or plan is another agent's: one the conversation's agent
 * handed a task to. Only then does the card say who proposed it, because the
 * conversation's own agent is already named at the head of the reply. A
 * proposal that does not say who made it, or one shown where the
 * conversation's agent is not known, is not attributed.
 */
export function proposedByOther(
  conversationAgentId: string | null | undefined,
  proposerId: string | null | undefined,
): boolean {
  const own = conversationAgentId ?? "";
  const proposer = proposerId ?? "";

  return own !== "" && proposer !== "" && proposer !== own;
}

/**
 * The last assistant message of the turn each message belongs to.
 *
 * A turn that proposes a change ends in words about it: "I've proposed a new
 * report... approve it and I'll run it". The call that raised the proposal
 * came earlier in the turn, so a card placed under that call sat above the
 * sentence asking the person to decide it. Cards anchor to the end of their
 * turn instead, under the words that introduce them.
 */
export function turnEndByMessage(messages: readonly AssistantMessage[]): Map<string, string> {
  const ends = new Map<string, string>();
  let turn: string[] = [];
  let last = "";
  const close = () => {
    // A turn with none of its own words in view has no end to anchor to;
    // what it proposed is shown out of place rather than nowhere.
    for (const id of last === "" ? [] : turn) {
      ends.set(id, last);
    }
    turn = [];
    last = "";
  };

  for (const message of messages) {
    // Another agent's steps are part of the turn that handed it the task:
    // its task does not start a new turn, and its words do not end this one.
    // What it proposed is placed where the turn's own proposals are.
    if (message.kind === "Delegated") {
      turn.push(message.id);
      continue;
    }
    if (message.role === "User") {
      close();
      continue;
    }
    if (message.role === "Assistant") {
      turn.push(message.id);
      last = message.id;
    }
  }
  close();

  return ends;
}

/**
 * Groups proposals under the end of the turn that asked for them.
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
  const ends = turnEndByMessage(messages);
  const byMessage = new Map<string, AssistantProposal[]>();
  const orphans: AssistantProposal[] = [];

  for (const proposal of proposals) {
    const anchor = ends.get(proposal.sourceMessageId);
    if (proposal.sourceMessageId === "" || anchor === undefined) {
      orphans.push(proposal);
      continue;
    }

    const existing = byMessage.get(anchor);
    if (existing) {
      existing.push(proposal);
    } else {
      byMessage.set(anchor, [proposal]);
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

/** How often the lists are re-read while an approval is being carried out. */
export const RUNNING_POLL_INTERVAL_MS = 2000;

/**
 * Whether anything approved has not reported back yet.
 *
 * Execution happens after the resolve call returns, so a card that was
 * refetched once read "waiting for it to run" until the thread was
 * remounted. While a proposal or a plan is between approval and its
 * outcome the lists are polled; the moment nothing is, they are not.
 */
export function pollIntervalFor(
  proposals: readonly AssistantProposal[],
  plans: readonly AssistantPlan[],
): number | false {
  const running =
    proposals.some(
      (proposal) =>
        (proposal.status === "Accepted" || proposal.status === "Modified") &&
        !proposal.executedAt &&
        !proposal.simulatedAt &&
        proposal.executionError === "",
    ) ||
    plans.some(
      (plan) =>
        plan.status === "Approved" &&
        plan.completedSteps < plan.stepCount &&
        (plan.failedStep ?? 0) === 0,
    );

  return running ? RUNNING_POLL_INTERVAL_MS : false;
}

/**
 * Which of a thread's proposals and plans have been decided, and to what, as
 * one comparable value.
 *
 * The thread watches it to learn that something was decided somewhere else —
 * the Desk's decisions, AI Control, another tab — which is when the server
 * starts the turn reporting it and this view should pick that turn up. It
 * changes only when a decision lands or an outcome follows one; a list that
 * refetches unchanged leaves it alone.
 */
export function decidedSignature(
  proposals: readonly AssistantProposal[],
  plans: readonly AssistantPlan[],
): string {
  const decided = [
    ...proposals
      .filter((proposal) => proposal.status !== "Pending")
      .map((proposal) => `p:${proposal.id}:${proposal.status}`),
    ...plans
      .filter((plan) => plan.status !== "Pending")
      .map((plan) => `l:${plan.id}:${plan.status}`),
  ];

  return decided.sort().join("|");
}
