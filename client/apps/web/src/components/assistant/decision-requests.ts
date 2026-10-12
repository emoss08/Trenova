import type { ToolStep } from "./activity";

/**
 * A waiting decision the assistant asked the person to make now: one
 * proposal, several of one tool decided together, or a plan.
 */
export type DecisionRequestRef = {
  callId: string;
  proposalIds: string[];
  planId: string;
};

export const REQUEST_DECISION_TOOL = "request_decision";

export const WITHDRAW_PROPOSAL_TOOL = "withdraw_proposal";

function textOf(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

/**
 * What a request_decision call, or the artifact it kept, asks the person to
 * decide: a plan by its id, several proposals, or one.
 */
export function decisionRequestOf(
  args: Record<string, unknown> | null | undefined,
): Omit<DecisionRequestRef, "callId"> | null {
  const planId = textOf(args?.planId);
  if (planId !== "") {
    return { proposalIds: [], planId };
  }

  const many = Array.isArray(args?.proposalIds)
    ? [...new Set(args.proposalIds.map(textOf).filter((id) => id !== ""))]
    : [];
  if (many.length > 0) {
    return { proposalIds: many, planId: "" };
  }

  const one = textOf(args?.proposalId);
  return one === "" ? null : { proposalIds: [one], planId: "" };
}

function keyOf(request: Omit<DecisionRequestRef, "callId">): string {
  return request.planId !== "" ? `plan:${request.planId}` : request.proposalIds.join(",");
}

function collect(
  calls: readonly { callId: string; args: Record<string, unknown> | null | undefined }[],
): DecisionRequestRef[] {
  const requests: DecisionRequestRef[] = [];
  const seen = new Set<string>();

  for (const { callId, args } of calls) {
    const request = decisionRequestOf(args);
    if (request === null) continue;
    const key = keyOf(request);
    if (seen.has(key)) continue;
    seen.add(key);
    requests.push({ callId, ...request });
  }

  return requests;
}

/**
 * The decisions a live turn asked the person to make now, read from its
 * finished request_decision calls. The runtime answers the call itself and
 * succeeds only for what is still waiting, so a call that failed asks for
 * nothing; each moves the approval box to its decision.
 */
export function decisionRequestsFromSteps(steps: readonly ToolStep[]): DecisionRequestRef[] {
  return collect(
    steps
      .filter((step) => step.name === REQUEST_DECISION_TOOL && step.status === "done")
      .map((step) => ({ callId: step.id, args: step.arguments })),
  );
}
