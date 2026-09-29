import type { ToolStep } from "./activity";
import type { ToolExchange } from "./thread-view";

/** A waiting proposal the assistant put back in front of the person. */
export type DecisionRequestRef = {
  callId: string;
  proposalId: string;
};

export const REQUEST_DECISION_TOOL = "request_decision";

function proposalIdOf(args: Record<string, unknown> | null | undefined): string {
  const value = args?.proposalId;
  return typeof value === "string" ? value.trim() : "";
}

/**
 * The cards a saved turn asked the person to decide. The runtime answers
 * request_decision itself and only succeeds for a proposal of this
 * conversation that is still waiting, so a call that failed shows nothing.
 */
export function decisionRequestsFrom(tools: readonly ToolExchange[]): DecisionRequestRef[] {
  const requests: DecisionRequestRef[] = [];
  const seen = new Set<string>();

  for (const exchange of tools) {
    if (exchange.call.name !== REQUEST_DECISION_TOOL) continue;
    const result = exchange.result;
    if (result === null || result.toolFailed) continue;
    const proposalId = proposalIdOf(exchange.call.arguments);
    if (proposalId === "" || seen.has(proposalId)) continue;
    seen.add(proposalId);
    requests.push({ callId: exchange.call.id, proposalId });
  }

  return requests;
}

/** The same cards, read out of a turn that is still streaming. */
export function decisionRequestsFromSteps(steps: readonly ToolStep[]): DecisionRequestRef[] {
  const requests: DecisionRequestRef[] = [];
  const seen = new Set<string>();

  for (const step of steps) {
    if (step.name !== REQUEST_DECISION_TOOL || step.status !== "done") continue;
    const proposalId = proposalIdOf(step.arguments);
    if (proposalId === "" || seen.has(proposalId)) continue;
    seen.add(proposalId);
    requests.push({ callId: step.id, proposalId });
  }

  return requests;
}
