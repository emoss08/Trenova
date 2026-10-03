import {
  DecideAgentProposalsDocument,
  PendingDecisionsDocument,
  PendingDecisionSummaryDocument,
  PendingPlanFieldsFragmentDoc,
  PendingProposalFieldsFragmentDoc,
  PlanStepsDocument,
  RecentDecisionsDocument,
  type DecideAgentProposalsInput,
  type PendingPlanFieldsFragment,
  type PendingProposalFieldsFragment,
  type PendingDecisionsInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";

export const PENDING_DECISIONS_KEY = "pending-decisions";
export const PENDING_DECISION_SUMMARY_KEY = "pending-decision-summary";
const PAGE_SIZE = 25;
const REFETCH_INTERVAL = 60_000;

export type PendingProposalNode = PendingProposalFieldsFragment & { __typename: "AgentProposal" };
export type PendingPlanNode = PendingPlanFieldsFragment & { __typename: "AgentPlan" };
export type PendingDecisionNode = PendingProposalNode | PendingPlanNode;
export type PlanStepNode = PendingProposalFieldsFragment;

export type PendingDecisionFilter = {
  agentDefinitionId?: string;
  toolName?: string;
};

export function isPendingProposal(node: PendingDecisionNode): node is PendingProposalNode {
  return node.__typename === "AgentProposal";
}

export function isPendingPlan(node: PendingDecisionNode): node is PendingPlanNode {
  return node.__typename === "AgentPlan";
}

/** The queue, a page at a time, newest first. */
export function usePendingDecisions(filter: PendingDecisionFilter, enabled = true) {
  return useInfiniteQuery({
    queryKey: [PENDING_DECISIONS_KEY, filter.agentDefinitionId ?? "", filter.toolName ?? ""],
    initialPageParam: null as string | null,
    queryFn: ({ pageParam, signal }) => {
      const input: PendingDecisionsInput = {
        first: PAGE_SIZE,
        after: pageParam,
        agentDefinitionId: filter.agentDefinitionId || null,
        toolName: filter.toolName || null,
      };
      return requestGraphQL({
        document: PendingDecisionsDocument,
        operationName: "PendingDecisions",
        variables: { input, includeTotalCount: pageParam === null },
        signal,
      });
    },
    getNextPageParam: (lastPage) => {
      const { hasNextPage, endCursor } = lastPage.pendingDecisions.pageInfo;
      return hasNextPage && endCursor ? endCursor : undefined;
    },
    refetchInterval: REFETCH_INTERVAL,
    enabled,
    select: (data) => ({
      nodes: data.pages.flatMap((page) =>
        page.pendingDecisions.edges.flatMap((edge): PendingDecisionNode[] => {
          const node = edge.node;
          if (node.__typename === "AgentProposal") {
            return [
              {
                ...getFragmentData(PendingProposalFieldsFragmentDoc, node),
                __typename: "AgentProposal",
              },
            ];
          }
          if (node.__typename === "AgentPlan") {
            return [
              { ...getFragmentData(PendingPlanFieldsFragmentDoc, node), __typename: "AgentPlan" },
            ];
          }
          return [];
        }),
      ),
      totalCount: data.pages[0]?.pendingDecisions.totalCount ?? null,
    }),
  });
}

export function usePendingDecisionSummary(enabled = true) {
  return useQuery({
    queryKey: [PENDING_DECISION_SUMMARY_KEY],
    queryFn: ({ signal }) =>
      requestGraphQL({
        document: PendingDecisionSummaryDocument,
        operationName: "PendingDecisionSummary",
        signal,
      }),
    refetchInterval: REFETCH_INTERVAL,
    enabled,
    select: (data) => data.pendingDecisionSummary,
  });
}

/** One decision made lately, with the proposal it decided. */
export type RecentDecisionNode = {
  id: string;
  decision: string;
  reasonCode: string;
  note: string;
  decidedByUserId: string;
  decidedByName: string;
  decidedAt: number;
  proposal: PendingProposalNode & { executionError: string };
};

/** The start of today where the person is, in Unix seconds. */
export function startOfToday(now = new Date()): number {
  const midnight = new Date(now);
  midnight.setHours(0, 0, 0, 0);
  return Math.floor(midnight.getTime() / 1000);
}

/**
 * What was decided since a moment, newest first. Kept under the queue's key,
 * so every decision that refreshes the queue refreshes this with it.
 */
export function useRecentDecisions(since: number, enabled = true) {
  return useQuery({
    queryKey: [PENDING_DECISIONS_KEY, "recent", since],
    queryFn: ({ signal }) =>
      requestGraphQL({
        document: RecentDecisionsDocument,
        operationName: "RecentDecisions",
        variables: { since, first: 50 },
        signal,
      }),
    refetchInterval: REFETCH_INTERVAL,
    enabled,
    select: (data): RecentDecisionNode[] =>
      data.recentDecisions.map((row) => ({
        ...row,
        proposal: {
          ...getFragmentData(PendingProposalFieldsFragmentDoc, row.proposal),
          executionError: row.proposal.executionError,
          __typename: "AgentProposal",
        },
      })),
  });
}

/** A plan's steps, read only when the plan is open in the detail pane. */
export function usePlanSteps(planId: string | null) {
  return useQuery({
    queryKey: ["agent-proposal-list", "plan-steps", planId ?? ""],
    queryFn: ({ signal }) =>
      requestGraphQL({
        document: PlanStepsDocument,
        operationName: "PlanSteps",
        variables: {
          input: {
            first: 50,
            fieldFilters: [{ field: "planId", operator: "eq", value: planId }],
            sort: [{ field: "planStep", direction: "asc" }],
          },
        },
        signal,
      }),
    enabled: planId !== null,
    select: (data): PlanStepNode[] =>
      data.agentProposals.edges.map((edge) =>
        getFragmentData(PendingProposalFieldsFragmentDoc, edge.node),
      ),
  });
}

export async function decideAgentProposals(ids: string[], input: DecideAgentProposalsInput) {
  const data = await requestGraphQL({
    document: DecideAgentProposalsDocument,
    operationName: "DecideAgentProposals",
    variables: { ids, input },
  });

  return data.decideAgentProposals;
}
