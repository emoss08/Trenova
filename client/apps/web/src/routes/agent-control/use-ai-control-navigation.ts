import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import { useQueryStates } from "nuqs";
import { useCallback } from "react";
import { AUDIT_SCOPE_PARAM, auditScopeParser } from "./_components/audit/audit-model";
import {
  ACTIVITY_VIEW_PARAM,
  AI_CONTROL_TAB_PARAM,
  AUDIT_VIEW_PARAM,
  CLEARED_TABLE_STATE,
  QUALITY_AGENT_PARAM,
  QUALITY_SUITE_RUN_PARAM,
  QUALITY_VIEW_PARAM,
  RETRIEVAL_SOURCE_PARAM,
  SAFETY_VIEW_PARAM,
  activityViewParser,
  activityViews,
  aiControlTabParser,
  auditViewParser,
  auditViews,
  qualityAgentParser,
  qualitySuiteRunParser,
  qualityViewParser,
  qualityViews,
  retrievalSourceParser,
  safetyViewParser,
  safetyViews,
  type ActivityView,
  type AIControlTab,
  type AuditView,
  type QualityView,
  type RailView,
  type RetrievalSource,
  type SafetyView,
} from "./ai-control-tabs";

const navigationParsers = {
  ...searchParamsParser,
  [AI_CONTROL_TAB_PARAM]: aiControlTabParser,
  [ACTIVITY_VIEW_PARAM]: activityViewParser,
  [SAFETY_VIEW_PARAM]: safetyViewParser,
  [QUALITY_VIEW_PARAM]: qualityViewParser,
  [QUALITY_AGENT_PARAM]: qualityAgentParser,
  [QUALITY_SUITE_RUN_PARAM]: qualitySuiteRunParser,
  [RETRIEVAL_SOURCE_PARAM]: retrievalSourceParser,
  [AUDIT_VIEW_PARAM]: auditViewParser,
  [AUDIT_SCOPE_PARAM]: auditScopeParser,
};

export type AIControlDestination = {
  tab: AIControlTab;
  view?: RailView;
  /** Narrows the quality views that take an agent to one agent. */
  qualityAgent?: string | null;
  /** Opens one suite run's cases. */
  suiteRun?: string | null;
  /** Narrows Retrieval's failed items to one source. */
  retrievalSource?: RetrievalSource | null;
  /** Filters the destination's table starts with, in its own field names. */
  fieldFilters?: FieldFilter[];
};

const isActivityView = (view: RailView): view is ActivityView =>
  (activityViews as readonly string[]).includes(view);
const isSafetyView = (view: RailView): view is SafetyView =>
  (safetyViews as readonly string[]).includes(view);
const isQualityView = (view: RailView): view is QualityView =>
  (qualityViews as readonly string[]).includes(view);
const isAuditView = (view: RailView): view is AuditView =>
  (auditViews as readonly string[]).includes(view);

/**
 * Moves between the page's sections and the tables under them in one write
 * to the address. Every table here keeps its page, search, filters, sort and
 * open row in the same keys, so a move always clears them first; a
 * destination that wants a filter names it, in the fields of the table it
 * opens.
 */
export function useAIControlNavigation() {
  const [, setParams] = useQueryStates(navigationParsers);

  return useCallback(
    (destination: AIControlDestination) => {
      const view = destination.view;
      void setParams(
        {
          ...CLEARED_TABLE_STATE,
          fieldFilters: destination.fieldFilters ?? null,
          [AI_CONTROL_TAB_PARAM]: destination.tab,
          [ACTIVITY_VIEW_PARAM]:
            destination.tab === "activity" && view && isActivityView(view) ? view : null,
          [SAFETY_VIEW_PARAM]:
            destination.tab === "safety" && view && isSafetyView(view) ? view : null,
          [QUALITY_VIEW_PARAM]:
            destination.tab === "quality" && view && isQualityView(view) ? view : null,
          [QUALITY_AGENT_PARAM]: destination.qualityAgent ?? null,
          [QUALITY_SUITE_RUN_PARAM]: destination.suiteRun ?? null,
          [RETRIEVAL_SOURCE_PARAM]:
            destination.tab === "retrieval" ? (destination.retrievalSource ?? null) : null,
          [AUDIT_VIEW_PARAM]:
            destination.tab === "audit" && view && isAuditView(view) ? view : null,
          [AUDIT_SCOPE_PARAM]: null,
        },
        { history: "push" },
      );
    },
    [setParams],
  );
}
