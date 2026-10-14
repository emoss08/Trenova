import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryState } from "nuqs";
import {
  QUALITY_AGENT_PARAM,
  QUALITY_SUITE_RUN_PARAM,
  qualityAgentParser,
  qualitySuiteRunParser,
  type QualityView,
} from "../../ai-control-tabs";
import { FigsSkeleton, HeroSkeleton, type FigSkeleton } from "../kit/tab-skeleton";
import { ExtractionViewSkeleton } from "./extraction/extraction-skeleton";

/** What each quality table shows at first. */
const TABLE_COLUMNS = { agents: 4, runs: 5, cases: 4, ratings: 4, golden: 5 } as const;

/** Satisfaction, ratings, quality score, regressions, and this month's eval spend. */
const FIGURES: readonly FigSkeleton[] = [
  [72, 60, 72],
  [48, 68, 112],
  [80, 56, 92],
  [76, 20, 60],
  [128, 84, 56],
];

/** Whether the view is headed by Nova's sentence and the figures. */
export function qualityViewHeaded(view: QualityView): boolean {
  return view !== "extraction" && view !== "golden";
}

/** Nova's sentence on how the agents score, and the five figures under it. */
export function QualityHeadSkeleton() {
  return (
    <>
      <HeroSkeleton />
      <FigsSkeleton items={FIGURES} />
    </>
  );
}

/** The line that names the one agent a table is narrowed to, and the way back to all. */
function ScopeSkeleton() {
  return (
    <div className="flex items-center justify-between gap-2" aria-hidden>
      <Skeleton className="h-4 w-44" />
      <Skeleton className="h-6 w-32" />
    </div>
  );
}

/** A table under the line that narrows it, when the address narrows it. */
function ScopedTableSkeleton({ scoped, columns }: { scoped: boolean; columns: number }) {
  const table = <DataTableSkeleton columnCount={columns} aria-hidden />;
  if (!scoped) {
    return table;
  }

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <ScopeSkeleton />
      {table}
    </div>
  );
}

/** One suite run's cases: the way back and the run it is, over its table. */
function SuiteRunCasesSkeleton() {
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex min-w-0 items-center gap-2" aria-hidden>
        <Skeleton className="h-6 w-36" />
        <Skeleton className="h-4 w-48" />
      </div>
      <DataTableSkeleton columnCount={TABLE_COLUMNS.cases} aria-hidden />
    </div>
  );
}

/**
 * The view under the figures, as the address picks it: a table, a suite run's cases in
 * place of the runs, a table narrowed to one agent, or document extraction's own layout.
 */
export function QualityBodySkeleton({ view }: { view: QualityView }) {
  const [agentId] = useQueryState(QUALITY_AGENT_PARAM, qualityAgentParser);
  const [suiteRunId] = useQueryState(QUALITY_SUITE_RUN_PARAM, qualitySuiteRunParser);

  switch (view) {
    case "extraction":
      return <ExtractionViewSkeleton />;
    case "runs":
      return suiteRunId ? (
        <SuiteRunCasesSkeleton />
      ) : (
        <ScopedTableSkeleton scoped={agentId !== null} columns={TABLE_COLUMNS.runs} />
      );
    case "ratings":
      return <ScopedTableSkeleton scoped={agentId !== null} columns={TABLE_COLUMNS.ratings} />;
    default:
      return <DataTableSkeleton columnCount={TABLE_COLUMNS[view]} aria-hidden />;
  }
}

/**
 * The Quality tab while its code is read: Nova's sentence and the figures over the agent
 * views, and the view the tab opens on.
 */
export function QualityTabSkeleton({ view }: { view: QualityView }) {
  const t = useT();

  return (
    <div className="tabp flex flex-col gap-1" aria-busy="true" aria-label={t("Loading quality")}>
      {qualityViewHeaded(view) && <QualityHeadSkeleton />}
      <QualityBodySkeleton view={view} />
    </div>
  );
}
