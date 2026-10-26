import { AnalysisResult } from "@/components/assistant/analysis-result";
import { analysisRunStatusAttrs, analysisViewOf } from "@/lib/analysis-run";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import { ANALYSIS_RUNS_PAGE, agentAnalysis } from "@/lib/queries/agent-analysis";
import { apiService } from "@/services/api";
import type { AnalysisRunsCursor } from "@/services/agent-analysis";
import type { AnalysisRunSummary } from "@/types/analysis";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { useQueryState } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { ANALYSIS_RUN_PARAM, analysisRunParser } from "../../ai-control-tabs";
import { AnalysisRunsSkeleton } from "./analysis-runs-skeleton";

/**
 * Every script an agent ran that showed someone a figure, table or chart, newest first: what it
 * answered and how it ended. Opening one reads it whole: the data it read, what it showed and
 * printed, and the code.
 */
export function AnalysisRuns() {
  const t = useT();
  const runs = useInfiniteQuery({
    queryKey: agentAnalysis.runs().queryKey,
    queryFn: ({ pageParam, signal }) =>
      apiService.agentAnalysisService.listRuns(
        { cursor: pageParam, limit: ANALYSIS_RUNS_PAGE },
        { signal },
      ),
    initialPageParam: null as AnalysisRunsCursor | null,
    getNextPageParam: (page): AnalysisRunsCursor | null =>
      page.nextBefore > 0 && page.nextBeforeId !== ""
        ? { before: page.nextBefore, beforeId: page.nextBeforeId }
        : null,
  });
  const [linkedRun] = useQueryState(ANALYSIS_RUN_PARAM, analysisRunParser);

  if (runs.isPending) {
    return <AnalysisRunsSkeleton />;
  }

  const items = runs.data?.pages.flatMap((page) => page.items) ?? [];
  const linkedElsewhere =
    linkedRun !== null && runs.isSuccess && !items.some((run) => run.id === linkedRun);

  return (
    <section aria-label={t("Analyses")} className="flex flex-col gap-3">
      {linkedElsewhere && <LinkedAnalysis runId={linkedRun} />}
      {runs.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("The analyses could not be loaded.")}</AlertDescription>
        </Alert>
      ) : items.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {t(
            "No analyses yet. Each script an agent runs that shows someone a figure, table or chart is listed here.",
          )}
        </p>
      ) : (
        <>
          <ul className="divide-border flex flex-col divide-y rounded-md border">
            {items.map((run) => (
              <AnalysisRunRow key={run.id} run={run} linked={run.id === linkedRun} />
            ))}
          </ul>
          {runs.hasNextPage && (
            <Button
              variant="outline"
              size="sm"
              className="self-center"
              disabled={runs.isFetchingNextPage}
              onClick={() => void runs.fetchNextPage()}
            >
              {runs.isFetchingNextPage ? t("Loading…") : t("Show older analyses")}
            </Button>
          )}
        </>
      )}
    </section>
  );
}

/** An analysis a link opened that is not on the pages loaded: read on its own, open. */
function LinkedAnalysis({ runId }: { runId: string }) {
  const t = useT();
  const opened = useQuery({ ...agentAnalysis.run(runId), staleTime: Number.POSITIVE_INFINITY });

  if (opened.isError) {
    return (
      <Alert size="sm">
        <AlertDescription>{t("The analysis you opened was not found.")}</AlertDescription>
      </Alert>
    );
  }
  if (!opened.data) {
    return (
      <div
        aria-busy="true"
        aria-label={t("Loading analysis")}
        className="ui-shimmer h-11 w-full rounded-md"
      />
    );
  }

  return (
    <ul aria-label={t("The analysis you opened")} className="flex flex-col rounded-md border">
      <AnalysisRunRow run={opened.data} linked />
    </ul>
  );
}

function AnalysisRunRow({ run, linked }: { run: AnalysisRunSummary; linked: boolean }) {
  const t = useT();
  const [toggled, setToggled] = useState<boolean | null>(null);
  const open = toggled ?? linked;
  const rowRef = useRef<HTMLLIElement>(null);

  useEffect(() => {
    if (linked) {
      rowRef.current?.scrollIntoView({ block: "nearest" });
    }
  }, [linked]);

  const status = analysisRunStatusAttrs(t)[run.status];

  return (
    <li ref={rowRef} className="flex flex-col">
      <button
        type="button"
        aria-expanded={open}
        className="ui-focus-ring hover:bg-muted/40 flex w-full items-center gap-3 px-3 py-2.5 text-left"
        onClick={() => setToggled(!open)}
      >
        <Badge variant={phaseTone(status.phase)} title={status.description}>
          {status.text}
        </Badge>
        <span className="min-w-0 flex-1 truncate text-sm">
          {run.title || run.purpose || t("Analysis")}
        </span>
        <span className="text-muted-foreground shrink-0 text-xs">
          {formatUnixTime(run.startedAt)}
        </span>
        <ChevronDownIcon
          className={cn("text-muted-foreground size-3.5 shrink-0", open && "rotate-180")}
        />
      </button>
      {open && <AnalysisRunDetail run={run} />}
    </li>
  );
}

/** One analysis opened in the history, read whole when it opens. */
function AnalysisRunDetail({ run }: { run: AnalysisRunSummary }) {
  const t = useT();
  const whole = useQuery({ ...agentAnalysis.run(run.id), staleTime: Number.POSITIVE_INFINITY });

  return (
    <div className="bg-muted/30 flex flex-col gap-3 px-3 pt-1 pb-3">
      <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-xs">
        <dt className="text-muted-foreground">{t("Ran for")}</dt>
        <dd>{formatWorkDuration(run.durationMs / 1000)}</dd>
      </dl>
      {whole.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("This analysis could not be loaded.")}</AlertDescription>
        </Alert>
      ) : whole.data ? (
        <AnalysisResult analysis={analysisViewOf(whole.data)} />
      ) : (
        <div
          aria-busy="true"
          aria-label={t("Loading analysis")}
          className="ui-shimmer h-32 w-full rounded-md"
        />
      )}
    </div>
  );
}
