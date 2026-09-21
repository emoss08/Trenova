import type { AssistantMessage, AssistantPlan, AssistantProposal } from "@/types/assistant";
import { classifyProposal, type ProposalPresentation } from "./proposal-state";

/**
 * What a plan card should show. The states are the proposal card's, because
 * a plan is decided and runs the way one proposal is: the difference is that
 * "done" means every step ran and "failed" means one stopped the rest.
 */
export type PlanPresentation = ProposalPresentation;

/** What one step of a plan shows beside its number. */
export type PlanStepState =
  | "waiting"
  | "running"
  | "done"
  | "failed"
  | "skipped"
  | "declined"
  | "simulated";

export type PlanGroup = {
  plan: AssistantPlan;
  /** The plan's proposals in the order they run; empty when none is listed. */
  steps: AssistantProposal[];
};

export function classifyPlan(
  plan: AssistantPlan,
  now: number = Date.now() / 1000,
): PlanPresentation {
  if (plan.status === "Pending" && (plan.expiresAt ?? 0) > 0 && plan.expiresAt! <= now) {
    return "closed";
  }

  switch (plan.status) {
    case "Pending":
      return plan.hold ? "held" : "awaiting";
    case "Approved":
      return "running";
    case "Completed":
      return "done";
    case "Failed":
      return "failed";
    case "Rejected":
      return "declined";
    default:
      return "closed";
  }
}

export function isPlanDecidable(plan: AssistantPlan): boolean {
  return classifyPlan(plan) === "awaiting";
}

/**
 * A step's state is read from its own proposal rather than the plan's
 * counters, so the card can show which write ran, which failed and which
 * never got its turn.
 */
export function planStepState(step: AssistantProposal): PlanStepState {
  if (step.status === "Skipped") {
    return "skipped";
  }

  switch (classifyProposal(step)) {
    case "done":
      return "done";
    case "failed":
      return "failed";
    case "running":
      return "running";
    case "declined":
      return "declined";
    case "simulated":
      return "simulated";
    default:
      return "waiting";
  }
}

/**
 * Splits a thread's proposals into plans and the proposals that stand alone.
 *
 * A plan is shown once, under the message its first step came from, with its
 * steps inside it; the steps are taken out of the per-message proposals so a
 * write is never shown twice. A plan whose message is not in the visible
 * thread goes to `orphans`, like a loose proposal. A proposal naming a plan
 * the thread does not list stays a proposal on its own, because a pending
 * write hidden behind a plan nobody can see is worse than one shown loose.
 */
export function groupPlans(
  plans: readonly AssistantPlan[],
  proposals: readonly AssistantProposal[],
  messages: readonly AssistantMessage[],
): { byMessage: Map<string, PlanGroup[]>; orphans: PlanGroup[]; standalone: AssistantProposal[] } {
  const knownMessageIds = new Set(messages.map((message) => message.id));
  const stepsByPlan = new Map<string, AssistantProposal[]>();
  for (const plan of plans) {
    stepsByPlan.set(plan.id, []);
  }

  const standalone: AssistantProposal[] = [];
  for (const proposal of proposals) {
    const steps = proposal.planId === "" ? undefined : stepsByPlan.get(proposal.planId);
    if (steps) {
      steps.push(proposal);
    } else {
      standalone.push(proposal);
    }
  }

  const byMessage = new Map<string, PlanGroup[]>();
  const orphans: PlanGroup[] = [];
  for (const plan of plans) {
    const steps = (stepsByPlan.get(plan.id) ?? []).sort((a, b) => a.planStep - b.planStep);
    const group: PlanGroup = { plan, steps };
    const anchor = steps[0]?.sourceMessageId ?? "";
    if (anchor === "" || !knownMessageIds.has(anchor)) {
      orphans.push(group);
      continue;
    }

    const existing = byMessage.get(anchor);
    if (existing) {
      existing.push(group);
    } else {
      byMessage.set(anchor, [group]);
    }
  }

  return { byMessage, orphans, standalone };
}
