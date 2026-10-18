import { AnalysisResult } from "@/components/assistant/analysis-result";
import { analysisRunStatusAttrs, analysisViewOf } from "@/lib/analysis-run";
import { formatWorkDuration } from "@/lib/ai-usage-format";
import { agentExtension } from "@/lib/queries/agent-extension";
import type { AnalysisRun } from "@/types/agent-extension";
import { useQuery } from "@tanstack/react-query";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQueryState } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { ANALYSIS_RUN_PARAM, analysisRunParser } from "../../ai-control-tabs";
import { AnalysisRunsSkeleton } from "./analysis-runs-skeleton";

/**
 * Every analysis agents ran, newest first: what it answered, the data it
 * read, the code, what it printed and drew, how long it ran and what it cost.
 */
export function AnalysisRuns() {
  const t = useT();
  const runs = useQuery(agentExtension.analysisRuns());
  const [linkedRun] = useQueryState(ANALYSIS_RUN_PARAM, analysisRunParser);

  if (runs.isPending) {
    return <AnalysisRunsSkeleton />;
  }

  const items = runs.data?.items ?? [];
  const linkedMissing =
    linkedRun !== null && runs.isSuccess && !items.some((run) => run.id === linkedRun);

  return (
    <section aria-label={t("Analyses")} className="flex flex-col gap-3">
      <h4 className="text-sm font-semibold">{t("Recent analyses")}</h4>
      {linkedMissing && (
        <Alert size="sm">
          <AlertDescription>
            {t("The analysis you opened is older than the ones listed here.")}
          </AlertDescription>
        </Alert>
      )}
      {runs.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("The analyses could not be loaded.")}</AlertDescription>
        </Alert>
      ) : items.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {t("No analyses yet. Each one an agent runs is listed here with its code and output.")}
        </p>
      ) : (
        <ul className="divide-border flex flex-col divide-y rounded-md border">
          {items.map((run) => (
            <AnalysisRunRow key={run.id} run={run} linked={run.id === linkedRun} />
          ))}
        </ul>
      )}
    </section>
  );
}

function AnalysisRunRow({ run, linked }: { run: AnalysisRun; linked: boolean }) {
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
      {open && (
        <div className="bg-muted/30 flex flex-col gap-3 px-3 pt-1 pb-3">
          <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-xs">
            <dt className="text-muted-foreground">{t("Ran for")}</dt>
            <dd>{formatWorkDuration(run.durationMs / 1000)}</dd>
            <dt className="text-muted-foreground">{t("Cost")}</dt>
            <dd>{formatCurrency(run.costUsd)}</dd>
          </dl>
          <AnalysisResult analysis={analysisViewOf(run)} />
        </div>
      )}
    </li>
  );
}
