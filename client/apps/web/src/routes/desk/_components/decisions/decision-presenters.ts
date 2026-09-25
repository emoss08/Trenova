import { presentProposal } from "@/components/assistant/proposal-presenters";
import type { AssistantProposal } from "@/types/assistant";
import {
  isPendingPlan,
  isPendingProposal,
  type PendingDecisionNode,
  type PlanStepNode,
} from "./use-pending-decisions";

/** One row of the queue as the list draws it, whichever record is behind it. */
export type DecisionRowView = {
  id: string;
  kind: "proposal" | "plan";
  title: string;
  summary: string;
  toolName: string;
  agent: { id: string; name: string; icon: string; accent: string; template: string } | null;
  createdAt: number;
  /** Set for a plan: how many writes it decides at once. */
  stepCount: number;
  /** A proposal in a batch is decided as proposed; a plan is decided whole on its own. */
  batchable: boolean;
};

function agentOf(node: PendingDecisionNode): DecisionRowView["agent"] {
  const definition = node.run?.definition;
  if (!definition) {
    return null;
  }

  return {
    id: definition.id,
    name: definition.name,
    icon: definition.icon ?? "",
    accent: definition.accent ?? "",
    template: definition.template ?? "",
  };
}

/** The proposal as the chat presenters read one, so the sentence is the same everywhere. */
export function asAssistantProposal(node: PlanStepNode): AssistantProposal {
  return {
    id: node.id,
    runId: node.runId,
    toolName: node.toolName,
    arguments: (node.toolParams ?? {}) as Record<string, unknown>,
    rationale: node.rationale,
    autonomyTier: node.autonomyTier,
    status: node.status,
    sourceMessageId: "",
    confidence: node.confidence,
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    hold: null,
    planId: node.planId ?? "",
    planStep: node.planStep,
    simulatedAt: node.simulatedAt ?? null,
    simulation: (node.simulation as AssistantProposal["simulation"]) ?? null,
    fields: node.parameterFields.map((field) => ({
      name: field.name,
      label: field.label,
      description: field.description,
      kind: field.kind,
      required: field.required,
      options: field.options,
      minimum: field.minimum ?? null,
      maximum: field.maximum ?? null,
      maxLength: field.maxLength ?? null,
      readOnly: field.readOnly,
    })),
    modifications: (node.modifications as Record<string, unknown> | null) ?? null,
  };
}

export function presentDecision(node: PendingDecisionNode): DecisionRowView | null {
  if (isPendingProposal(node)) {
    const view = presentProposal(asAssistantProposal(node));
    return {
      id: node.id,
      kind: "proposal",
      title: view.title,
      summary: view.summary,
      toolName: node.toolName,
      agent: agentOf(node),
      createdAt: node.createdAt,
      stepCount: 0,
      batchable: true,
    };
  }
  if (isPendingPlan(node)) {
    return {
      id: node.id,
      kind: "plan",
      title: node.title,
      summary: node.summary,
      toolName: "plan",
      agent: agentOf(node),
      createdAt: node.createdAt,
      stepCount: node.stepCount,
      batchable: false,
    };
  }

  return null;
}
