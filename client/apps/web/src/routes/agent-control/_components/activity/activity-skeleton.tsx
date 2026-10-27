import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ActivityView } from "../../ai-control-tabs";
import { FigsSkeleton, HeroSkeleton, type FigSkeleton } from "../kit/tab-skeleton";
import { AnalysisRunsSkeleton } from "./analysis-runs-skeleton";

/** The views that are a data table; analyses are a history of their own. */
export type ActivityTableView = Exclude<ActivityView, "analyses">;

/** What each activity table shows at first, so its skeleton has the loaded table's columns. */
export const ACTIVITY_TABLE_COLUMNS: Record<ActivityTableView, number> = {
  runs: 6,
  proposals: 6,
  plans: 5,
  evaluations: 5,
  exceptions: 7,
};

/** Runs today, failed, awaiting a decision, and approved as proposed. */
const FIGURES: readonly FigSkeleton[] = [
  [68, 52, 96],
  [40, 28, 88],
  [124, 28, 60],
  [140, 56, 72],
];

/** Nova's sentence on today's runs and what waits on a person, and the figures under it. */
export function ActivityHeadSkeleton() {
  return (
    <>
      <HeroSkeleton />
      <FigsSkeleton items={FIGURES} />
    </>
  );
}

/**
 * The Activity tab while its code is read: the head, and the table the view switch above
 * it picks, at that table's own columns.
 */
export function ActivityTabSkeleton({ view }: { view: ActivityView }) {
  const t = useT();

  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading activity")}>
      <ActivityHeadSkeleton />
      {view === "analyses" ? (
        <AnalysisRunsSkeleton />
      ) : (
        <DataTableSkeleton columnCount={ACTIVITY_TABLE_COLUMNS[view]} aria-hidden />
      )}
    </div>
  );
}
