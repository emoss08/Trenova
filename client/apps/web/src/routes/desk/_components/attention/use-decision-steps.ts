import type { AssistantProposal } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  PendingProposalFieldsFragmentDoc,
  PlanStepsDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { asAssistantProposal } from "../decisions/decision-presenters";

/** The decision whose changes are read: one proposal, or a plan and its steps. */
export type DecisionTarget = { proposalId: string } | { planId: string };

/**
 * The changes one decision covers, in the order they run: its proposal, or
 * every step of its plan. Read by id rather than by status, so a decision
 * already made still shows what it changed.
 */
export function useDecisionSteps(target: DecisionTarget | null) {
  const filter =
    target === null
      ? null
      : "planId" in target
        ? { field: "planId", value: target.planId }
        : { field: "id", value: target.proposalId };

  return useQuery({
    queryKey: ["agent-proposal-list", "needs-decision", filter?.field ?? "", filter?.value ?? ""],
    queryFn: ({ signal }) =>
      requestGraphQL({
        document: PlanStepsDocument,
        operationName: "PlanSteps",
        variables: {
          input: {
            first: 50,
            fieldFilters: [{ field: filter?.field ?? "id", operator: "eq", value: filter?.value }],
            sort: [{ field: "planStep", direction: "asc" }],
          },
        },
        signal,
      }),
    enabled: filter !== null,
    select: (data): AssistantProposal[] =>
      data.agentProposals.edges.map((edge) =>
        asAssistantProposal(getFragmentData(PendingProposalFieldsFragmentDoc, edge.node)),
      ),
  });
}
