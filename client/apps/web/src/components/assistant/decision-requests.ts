import type { ToolStep } from "./activity";
import type { ToolExchange } from "./thread-view";

/**
 * A waiting decision the assistant put back in front of the person: one
 * proposal, several of one tool shown as one card, or a plan.
 */
export type DecisionRequestRef = {
  callId: string;
  proposalIds: string[];
  planId: string;
};

export const REQUEST_DECISION_TOOL = "request_decision";

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
 * The cards a saved turn asked the person to decide. The runtime answers
 * request_decision itself and only succeeds for what is still waiting in this
 * conversation, so a call that failed shows nothing.
 */
export function decisionRequestsFrom(tools: readonly ToolExchange[]): DecisionRequestRef[] {
  return collect(
    tools
      .filter(
        (exchange) =>
          exchange.call.name === REQUEST_DECISION_TOOL &&
          exchange.result !== null &&
          !exchange.result.toolFailed,
      )
      .map((exchange) => ({ callId: exchange.call.id, args: exchange.call.arguments })),
  );
}

/** The same cards, read out of a turn that is still streaming. */
export function decisionRequestsFromSteps(steps: readonly ToolStep[]): DecisionRequestRef[] {
  return collect(
    steps
      .filter((step) => step.name === REQUEST_DECISION_TOOL && step.status === "done")
      .map((step) => ({ callId: step.id, args: step.arguments })),
  );
}
