import type { PendingPlanNode, PendingProposalNode } from "../use-pending-decisions";

/* An AgentProposal as the decision queue's PendingProposalFields reads it. */
export function proposalNode(overrides: Partial<PendingProposalNode> = {}): PendingProposalNode {
  return {
    __typename: "AgentProposal",
    id: "aprop_1",
    reviewRequired: true,
    organizationId: "org_1",
    businessUnitId: "bu_1",
    runId: "arun_1",
    toolName: "post_customer_payment",
    toolParams: { amount: "1500.00" },
    confidence: 0.9,
    rationale: "The remittance names both invoices.",
    autonomyTier: "ActWithApproval",
    status: "Pending",
    planId: null,
    planStep: 0,
    simulatedAt: null,
    simulation: null,
    modifications: null,
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    parameterFields: [],
    evidence: [],
    run: null,
    ...overrides,
  };
}

/* An AgentPlan as the decision queue's PendingPlanFields reads it. */
export function planNode(overrides: Partial<PendingPlanNode> = {}): PendingPlanNode {
  return {
    __typename: "AgentPlan",
    id: "apl_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    runId: "arun_1",
    title: "Post two payments",
    summary: "",
    planStatus: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedByUserId: null,
    decidedAt: null,
    expiresAt: 0,
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    run: null,
    ...overrides,
  } as PendingPlanNode;
}
