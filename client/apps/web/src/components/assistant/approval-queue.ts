import { AI_CONTROL_PATH, aiControlAgentPath } from "@/routes/agent-control/ai-control-tabs";
import type { DecisionFocus } from "@/stores/assistant-store";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  AssistantArtifact,
  AssistantPlan,
  AssistantProposal,
  ProposalHold,
} from "@/types/assistant";
import { classifyPlan } from "./plan-state";
import { classifyProposal, type ProposalPresentation } from "./proposal-state";

/** The most proposals one batch decision takes; the server refuses more. */
export const MAX_BATCH_PROPOSALS = 50;

/**
 * One thing the approval box asks the person to decide.
 *
 * - `proposal`: one write, approved as shown or with the person's changes.
 * - `batch`: several writes of one tool raised together, or asked for
 *   together, decided in one answer; each still carries the digest of its own
 *   preview, so every approval stays pinned to what was shown for it.
 * - `plan`: several writes run in order, decided whole.
 *
 * `members` are the keys deferral remembers: an entry is deferred only while
 * every one of them is, so a write that joins it later opens the box again.
 *
 * `hold` is the switch keeping the entry from being decided: the
 * organization's pause or the agent's own shadow switch. A held entry still
 * waits on the person, so it is asked like any other, with the switch named
 * in place of the approval; the server refuses a decision until it is off.
 */
export type ApprovalEntry =
  | {
      kind: "proposal";
      key: string;
      createdAt: number;
      members: string[];
      hold: ProposalHold | null;
      proposal: AssistantProposal;
    }
  | {
      kind: "batch";
      key: string;
      createdAt: number;
      members: string[];
      hold: ProposalHold | null;
      toolName: string;
      proposals: AssistantProposal[];
    }
  | {
      kind: "plan";
      key: string;
      createdAt: number;
      members: string[];
      hold: ProposalHold | null;
      plan: AssistantPlan;
      steps: AssistantProposal[];
    };

export function proposalKey(id: string): string {
  return `proposal:${id}`;
}

/**
 * The decision request the agent raised for an entry: the one naming any of
 * its proposals, or its plan. Members are keyed, so a proposal is looked up by
 * its key, not its bare id.
 */
export function decisionArtifactFor(
  entry: ApprovalEntry,
  artifacts: readonly Pick<AssistantArtifact, "id" | "kind" | "proposalId" | "planId">[],
): string | null {
  const artifact = artifacts.find(
    (candidate) =>
      candidate.kind === "decision_request" &&
      ((candidate.proposalId !== "" && entry.members.includes(proposalKey(candidate.proposalId))) ||
        (entry.kind === "plan" && candidate.planId !== "" && candidate.planId === entry.plan.id)),
  );

  return artifact?.id ?? null;
}

export function planKey(id: string): string {
  return `plan:${id}`;
}

/** The keys a request to decide names, for clearing them from the deferred list. */
export function focusKeys(focus: DecisionFocus): string[] {
  return focus.planId !== "" ? [planKey(focus.planId)] : focus.proposalIds.map(proposalKey);
}

function proposalEntry(proposal: AssistantProposal): ApprovalEntry {
  const key = proposalKey(proposal.id);

  return {
    kind: "proposal",
    key,
    createdAt: proposal.createdAt,
    members: [key],
    hold: proposal.hold ?? null,
    proposal,
  };
}

function batchEntry(toolName: string, proposals: AssistantProposal[]): ApprovalEntry {
  const members = proposals.map((proposal) => proposalKey(proposal.id));

  return {
    kind: "batch",
    key: `batch:${proposals.map((proposal) => proposal.id).join(",")}`,
    createdAt: Math.min(...proposals.map((proposal) => proposal.createdAt)),
    members,
    hold: proposals[0]?.hold ?? null,
    toolName,
    proposals,
  };
}

function chunk<T>(items: readonly T[], size: number): T[][] {
  const chunks: T[][] = [];
  for (let start = 0; start < items.length; start += size) {
    chunks.push(items.slice(start, start + size));
  }

  return chunks;
}

/**
 * The wording the person changed on a draft and sent for approval, which goes
 * with the approval as its modifications; null when they changed nothing.
 */
export function pendingEdits(proposal: AssistantProposal): Record<string, unknown> | null {
  const edits = proposal.pendingModifications;

  return edits && Object.keys(edits).length > 0 ? edits : null;
}

/**
 * Proposals of one tool from one run, as entries: alone when there is one,
 * otherwise as batches the server takes in one call.
 */
function groupEntries(toolName: string, group: AssistantProposal[]): ApprovalEntry[] {
  if (group.length === 1) {
    return [proposalEntry(group[0])];
  }

  return chunk(group, MAX_BATCH_PROPOSALS).map((part) =>
    part.length === 1 ? proposalEntry(part[0]) : batchEntry(toolName, part),
  );
}

/**
 * Proposals asked for together by name: they stand alone, wait on the
 * person, and share one tool, or the request decides nothing together.
 */
function requestedTogether(
  focus: DecisionFocus | null | undefined,
  standalone: readonly AssistantProposal[],
): AssistantProposal[] {
  if (!focus || focus.planId !== "" || focus.proposalIds.length < 2) {
    return [];
  }

  const byId = new Map(
    standalone
      .filter((proposal) => !proposal.hold && pendingEdits(proposal) === null)
      .map((proposal) => [proposal.id, proposal]),
  );
  const requested = focus.proposalIds
    .map((id) => byId.get(id))
    .filter((proposal) => proposal !== undefined);
  const tools = new Set(requested.map((proposal) => proposal.toolName));

  return requested.length >= 2 && tools.size === 1 ? requested.slice(0, MAX_BATCH_PROPOSALS) : [];
}

/**
 * Everything in a conversation waiting on the person, as the approval box
 * asks it: oldest first, one entry at a time.
 *
 * A plan is one entry with its steps. A proposal outside any plan is its own
 * entry, except that writes of one tool raised by the same turn are one
 * batch, and so are proposals the agent asked the person to decide together.
 * Proposals from different turns are never merged: each was proposed on its
 * own and is decided against the preview it was shown with. What waits on
 * the person is listed, held or not; an expired or decided write is history,
 * not a question.
 */
export function approvalQueue(
  proposals: readonly AssistantProposal[],
  plans: readonly AssistantPlan[],
  focus?: DecisionFocus | null,
  now: number = Date.now() / 1000,
): ApprovalEntry[] {
  const listedPlans = new Set(plans.map((plan) => plan.id));
  const stepsByPlan = new Map<string, AssistantProposal[]>();
  const standalone: AssistantProposal[] = [];
  for (const proposal of proposals) {
    if (proposal.planId !== "" && listedPlans.has(proposal.planId)) {
      const steps = stepsByPlan.get(proposal.planId) ?? [];
      steps.push(proposal);
      stepsByPlan.set(proposal.planId, steps);
      continue;
    }
    if (waitsOnPerson(classifyProposal(proposal, now))) {
      standalone.push(proposal);
    }
  }

  const entries: ApprovalEntry[] = [];
  for (const plan of plans) {
    if (!waitsOnPerson(classifyPlan(plan, now))) {
      continue;
    }
    const key = planKey(plan.id);
    entries.push({
      kind: "plan",
      key,
      createdAt: plan.createdAt,
      members: [key],
      hold: plan.hold ?? null,
      plan,
      steps: (stepsByPlan.get(plan.id) ?? []).sort((a, b) => a.planStep - b.planStep),
    });
  }

  const together = requestedTogether(focus, standalone);
  if (together.length > 0) {
    entries.push(batchEntry(together[0].toolName, together));
  }
  const taken = new Set(together.map((proposal) => proposal.id));

  const byRunAndTool = new Map<string, AssistantProposal[]>();
  for (const proposal of standalone) {
    if (taken.has(proposal.id)) {
      continue;
    }
    // A change the person edited is approved as edited, against the preview
    // of those edits, so it is never folded into a batch of others.
    if (pendingEdits(proposal) !== null) {
      entries.push(proposalEntry(proposal));
      continue;
    }
    // A hold is the run's, so a run's writes are held together; the reason
    // is in the key all the same, so a batch is never part held.
    const group = `${proposal.runId}\u0000${proposal.toolName}\u0000${proposal.hold?.reason ?? ""}`;
    const members = byRunAndTool.get(group);
    if (members) {
      members.push(proposal);
    } else {
      byRunAndTool.set(group, [proposal]);
    }
  }
  for (const group of byRunAndTool.values()) {
    entries.push(...groupEntries(group[0].toolName, group));
  }

  return entries.sort((a, b) => a.createdAt - b.createdAt || a.key.localeCompare(b.key));
}

/** Whether a card in this state still waits on a person: undecided, held or not. */
function waitsOnPerson(state: ProposalPresentation): boolean {
  return state === "awaiting" || state === "held";
}

/** Whether the person put this entry off: every write in it is deferred. */
export function isDeferred(entry: ApprovalEntry, deferred: ReadonlySet<string>): boolean {
  return entry.members.every((member) => deferred.has(member));
}

/** Whether an entry is the decision a request named: the plan, or any of its writes. */
export function holdsFocus(entry: ApprovalEntry, focus: DecisionFocus): boolean {
  if (focus.planId !== "") {
    return entry.kind === "plan" && entry.plan.id === focus.planId;
  }
  if (entry.kind === "plan") {
    return entry.steps.some((step) => focus.proposalIds.includes(step.id));
  }

  return focus.proposalIds.some((id) => entry.members.includes(proposalKey(id)));
}

/**
 * The entry the approval box shows, and its place in the queue: the one the
 * agent asked about when there is one still waiting, otherwise the oldest
 * the person has not put off and can decide now, otherwise the oldest held
 * one, so a pause never hides that something waits. Null when everything
 * waiting was deferred.
 */
export function currentEntry(
  queue: readonly ApprovalEntry[],
  deferred: ReadonlySet<string>,
  focus?: DecisionFocus | null,
): { entry: ApprovalEntry; index: number } | null {
  if (focus) {
    const index = queue.findIndex((entry) => holdsFocus(entry, focus));
    if (index >= 0) {
      return { entry: queue[index], index };
    }
  }

  const decidable = queue.findIndex((entry) => entry.hold === null && !isDeferred(entry, deferred));
  const index =
    decidable >= 0 ? decidable : queue.findIndex((entry) => !isDeferred(entry, deferred));

  return index >= 0 ? { entry: queue[index], index } : null;
}

/**
 * The switch holding an entry and where it is turned off: the organization's
 * pause on the AI Control overview, or the proposing agent's own shadow
 * switch on its sheet. Null when nothing holds it.
 */
export type HoldSwitch =
  | { scope: "organization"; to: string }
  | { scope: "agent"; to: string; agentName: string };

export function holdSwitch(entry: ApprovalEntry): HoldSwitch | null {
  return holdSwitchOf(entry.hold, proposerOf(entry));
}

/** The switch a hold names, for a proposal or plan raised by the given agent. */
export function holdSwitchOf(
  hold: ProposalHold | null | undefined,
  proposer: { agentId?: string | null; agentName?: string | null } | undefined,
): HoldSwitch | null {
  if (!hold) {
    return null;
  }
  if (hold.reason === "OrganizationPaused") {
    return { scope: "organization", to: AI_CONTROL_PATH };
  }

  return {
    scope: "agent",
    to: aiControlAgentPath(proposer?.agentId),
    agentName: hold.agentName || proposer?.agentName || "",
  };
}

/** Why a held change can't be decided, in a few words: the switch that is on. */
export function holdReasonText(held: HoldSwitch, t: TranslateFn): string {
  if (held.scope === "organization") {
    return t("Agents are paused for the organization");
  }

  return held.agentName !== ""
    ? t("{0} is in shadow", held.agentName)
    : t("This agent is in shadow");
}

/** What turns the switch off, as the label of the link to it. */
export function holdActionText(held: HoldSwitch, t: TranslateFn): string {
  return held.scope === "organization" ? t("Resume agents") : t("Turn off shadow");
}

function proposerOf(
  entry: ApprovalEntry,
): Pick<AssistantProposal, "agentId" | "agentName"> | undefined {
  switch (entry.kind) {
    case "proposal":
      return entry.proposal;
    case "batch":
      return entry.proposals[0];
    default:
      return entry.plan;
  }
}
