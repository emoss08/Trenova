import { PortalRunResult } from "@/components/assistant/portal-run-result";
import { usePermission } from "@/hooks/use-permission";
import { portalRunResultView, portalRunStatusAttrs, portalRunTitle } from "@/lib/portal-run";
import { agentExtension } from "@/lib/queries/agent-extension";
import type { PortalRun } from "@/types/agent-extension";
import { useQuery } from "@tanstack/react-query";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryState } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { PORTAL_RUN_PARAM, portalRunParser } from "../../ai-control-tabs";
import { PortalRunsSkeleton } from "./portal-runs-skeleton";

/**
 * Every portal task agents ran, newest first: what the browser was told, every page it
 * reached, how it ended and its recording. This is the record a person reviews.
 */
export function PortalRuns() {
  const t = useT();
  const runs = useQuery(agentExtension.portalRuns(""));
  const [linkedRun] = useQueryState(PORTAL_RUN_PARAM, portalRunParser);

  if (runs.isPending) {
    return <PortalRunsSkeleton />;
  }

  const items = runs.data?.items ?? [];
  const linkedMissing =
    linkedRun !== null && runs.isSuccess && !items.some((run) => run.id === linkedRun);

  return (
    <section aria-label={t("Portal tasks")} className="flex flex-col gap-3">
      <h4 className="text-sm font-semibold">{t("Recent portal tasks")}</h4>
      {linkedMissing && (
        <Alert size="sm">
          <AlertDescription>
            {t("The portal task you opened is older than the ones listed here.")}
          </AlertDescription>
        </Alert>
      )}
      {runs.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("The portal tasks could not be loaded.")}</AlertDescription>
        </Alert>
      ) : items.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {t("No portal tasks yet. Each one an agent runs is listed here with what it did.")}
        </p>
      ) : (
        <ul className="divide-border flex flex-col divide-y rounded-md border">
          {items.map((run) => (
            <PortalRunRow key={run.id} run={run} linked={run.id === linkedRun} />
          ))}
        </ul>
      )}
    </section>
  );
}

function PortalRunRow({ run, linked }: { run: PortalRun; linked: boolean }) {
  const t = useT();
  const [toggled, setToggled] = useState<boolean | null>(null);
  const open = toggled ?? linked;
  const rowRef = useRef<HTMLLIElement>(null);

  useEffect(() => {
    if (linked) {
      rowRef.current?.scrollIntoView({ block: "nearest" });
    }
  }, [linked]);

  const status = portalRunStatusAttrs(t)[run.status];

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
        <span className="min-w-0 flex-1 truncate text-sm">{portalRunTitle(t, run)}</span>
        <span className="text-muted-foreground shrink-0 text-xs">
          {formatUnixTime(run.startedAt)}
        </span>
        <ChevronDownIcon
          className={cn("text-muted-foreground size-3.5 shrink-0", open && "rotate-180")}
        />
      </button>
      {open && <PortalRunDetail run={run} />}
    </li>
  );
}

function PortalRunDetail({ run }: { run: PortalRun }) {
  const t = useT();
  const { allowed: canEnterCode } = usePermission(Resource.AgentExtension, Operation.Update);

  return (
    <div className="bg-muted/30 flex flex-col gap-3 px-3 pt-1 pb-3 text-sm">
      <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-xs">
        <dt className="text-muted-foreground">{t("Steps")}</dt>
        <dd>{run.steps}</dd>
        <dt className="text-muted-foreground">{t("Cost")}</dt>
        <dd>{formatCurrency(run.costUsd)}</dd>
        <dt className="text-muted-foreground">{t("Signed in")}</dt>
        <dd>{run.usedSignIn ? t("Yes") : t("No")}</dd>
        {run.finishedAt ? (
          <>
            <dt className="text-muted-foreground">{t("Finished")}</dt>
            <dd>{formatUnixTime(run.finishedAt)}</dd>
          </>
        ) : null}
      </dl>
      <div className="flex flex-col gap-1">
        <span className="text-muted-foreground text-xs">{t("What the browser was told")}</span>
        <p className="bg-background rounded-md border p-2 font-mono text-xs whitespace-pre-wrap">
          {run.instruction}
        </p>
      </div>
      <PortalRunResult run={portalRunResultView(run)} canEnterCode={canEnterCode} />
    </div>
  );
}
