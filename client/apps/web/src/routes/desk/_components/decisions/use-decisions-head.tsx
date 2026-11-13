import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryStates } from "nuqs";
import { useMemo } from "react";
import type { NeedsHeadLens } from "../attention/needs-head";
import { DecisionFilterMenu } from "./decision-filter-menu";
import {
  DECISION_ADDRESS,
  NO_DECISION_FACETS,
  NO_DECISION_FILTERS,
  decisionAddressOf,
  decisionFilterTokens,
  decisionFiltersFromAddress,
  pendingFilter,
  type DecisionFacets,
  type DecisionFilters,
} from "./decision-filters";
import {
  usePendingDecisionSummary,
  usePendingDecisions,
  useRecentDecisions,
} from "./use-pending-decisions";

/**
 * What the Decisions lens says in the page's head, and what its queue is
 * narrowed to: the words searched for, one agent, one kind of change. They
 * live in the address; the page reads them here once and hands them to the
 * lens. Nothing is read while another lens is in front.
 */
export function useDecisionsHead({
  enabled,
  since,
  filterOpen,
  onFilterOpen,
}: {
  /** Set while the lens is in front; nothing is read for it otherwise. */
  enabled: boolean;
  /** The start of the reader's day, which "decided" counts from. */
  since: number;
  filterOpen: boolean;
  onFilterOpen: (open: boolean) => void;
}): {
  head: NeedsHeadLens;
  filters: DecisionFilters;
  setFilters: (filters: DecisionFilters) => void;
  /** How many changes wait in all, whatever the queue is narrowed to; null until it is known. */
  total: number | null;
} {
  const t = useT();
  const [address, setAddress] = useQueryStates(DECISION_ADDRESS, { history: "replace" });
  const filters = useMemo(() => decisionFiltersFromAddress(address), [address]);
  const setFilters = (next: DecisionFilters) => void setAddress(decisionAddressOf(next));

  const summary = usePendingDecisionSummary(enabled).data;
  const facets = useMemo<DecisionFacets>(
    () =>
      summary
        ? {
            agents: summary.byAgent.map((agent) => ({
              id: agent.agentDefinitionId,
              name: agent.agentName,
              count: agent.count,
            })),
            tools: summary.byTool.map((entry) => ({ tool: entry.toolName, count: entry.count })),
          }
        : NO_DECISION_FACETS,
    [summary],
  );

  // The lens reads the same queue and the same decisions; these are that read.
  const queue = usePendingDecisions(pendingFilter(filters), enabled).data;
  const decided = useRecentDecisions(since, enabled).data;
  const waiting = queue ? (queue.totalCount ?? queue.nodes.length) : null;
  const done = decided?.length ?? 0;

  const tokens = decisionFilterTokens(t, filters, facets).map((token) => ({
    key: token.key,
    label: token.label,
    onRemove: () =>
      setFilters(token.type === "agent" ? { ...filters, agentId: "" } : { ...filters, tool: "" }),
  }));

  return {
    filters,
    setFilters,
    total: summary?.total ?? null,
    head: {
      progress:
        waiting !== null && decided !== undefined
          ? { label: t("{0} of {1} decided", done, done + waiting), done, total: done + waiting }
          : null,
      whose: null,
      search: {
        value: filters.search,
        onCommit: (search) => setFilters({ ...filters, search }),
      },
      filter: (
        <DecisionFilterMenu
          filters={filters}
          facets={facets}
          open={filterOpen}
          onOpenChange={onFilterOpen}
          onChange={setFilters}
        />
      ),
      tokens,
      onClearFilters: () => setFilters({ ...NO_DECISION_FILTERS, search: filters.search }),
    },
  };
}
