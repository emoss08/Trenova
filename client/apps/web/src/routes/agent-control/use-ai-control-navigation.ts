import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import { useQueryStates } from "nuqs";
import { useCallback } from "react";
import {
  ACTIVITY_VIEW_PARAM,
  AI_CONTROL_TAB_PARAM,
  CLEARED_TABLE_STATE,
  QUALITY_AGENT_PARAM,
  QUALITY_SUITE_RUN_PARAM,
  QUALITY_VIEW_PARAM,
  SAFETY_VIEW_PARAM,
  activityViewParser,
  activityViews,
  aiControlTabParser,
  qualityAgentParser,
  qualitySuiteRunParser,
  qualityViewParser,
  qualityViews,
  safetyViewParser,
  safetyViews,
  type ActivityView,
  type AIControlTab,
  type QualityView,
  type RailView,
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
};

export type AIControlDestination = {
  tab: AIControlTab;
  view?: RailView;
  /** Narrows the quality views that take an agent to one agent. */
  qualityAgent?: string | null;
  /** Opens one suite run's cases. */
  suiteRun?: string | null;
  /** Filters the destination's table starts with, in its own field names. */
  fieldFilters?: FieldFilter[];
};

const isActivityView = (view: RailView): view is ActivityView =>
  (activityViews as readonly string[]).includes(view);
const isSafetyView = (view: RailView): view is SafetyView =>
  (safetyViews as readonly string[]).includes(view);
const isQualityView = (view: RailView): view is QualityView =>
  (qualityViews as readonly string[]).includes(view);

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
        },
        { history: "push" },
      );
    },
    [setParams],
  );
}
