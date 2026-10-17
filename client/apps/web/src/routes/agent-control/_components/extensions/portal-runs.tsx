import { ExternalLink } from "@/components/link";
import { portalRunStatusAttrs, portalRunTitle } from "@/lib/portal-run";
import { agentExtension } from "@/lib/queries/agent-extension";
import { apiService } from "@/services/api";
import type { PortalRecording, PortalRun } from "@/types/agent-extension";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ChevronDownIcon, PlayCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
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
        <span className="min-w-0 flex-1 truncate text-sm">
          {portalRunTitle(t, run)}
        </span>
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
  const [recording, setRecording] = useState<PortalRecording | null>(null);
  const fetchRecording = useMutation({
    mutationFn: () => apiService.agentExtensionService.portalRunRecording(run.id),
    onSuccess: setRecording,
  });
  const recordingLink = recording?.shareUrl || recording?.recordingUrl || "";

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
      {run.answer && run.status !== "Strayed" && (
        <div className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs">{t("What the portal said")}</span>
          <p className="text-xs whitespace-pre-wrap">{run.answer}</p>
        </div>
      )}
      {run.failure && (
        <Alert size="sm" variant={run.status === "Strayed" ? "warning" : "destructive"}>
          <AlertDescription>{run.failure}</AlertDescription>
        </Alert>
      )}
      {run.visitedUrls.length > 0 && (
        <div className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs">
            {t("{0, plural, one {# page visited} other {# pages visited}}", run.visitedUrls.length)}
          </span>
          <ul className="flex flex-col gap-0.5 font-mono text-xs">
            {run.visitedUrls.map((url, index) => (
              <li
                key={`${index}-${url}`}
                className={cn("truncate", run.offSiteUrls.includes(url) && "text-danger")}
                title={run.offSiteUrls.includes(url) ? t("Outside the site's addresses") : url}
              >
                {url}
              </li>
            ))}
          </ul>
        </div>
      )}
      {run.hasRecording && (
        <div className="flex flex-wrap items-center gap-2">
          {recordingLink ? (
            <ExternalLink href={recordingLink}>{t("Open the recording")}</ExternalLink>
          ) : (
            <Button
              type="button"
              size="sm"
              variant="outline"
              isLoading={fetchRecording.isPending}
              loadingText={t("Recording")}
              onClick={() => fetchRecording.mutate()}
            >
              <PlayCircleIcon className="size-3.5" />
              {t("Recording")}
            </Button>
          )}
          {fetchRecording.isError && (
            <Alert size="sm" variant="destructive">
              <AlertDescription>
                {fetchRecording.error instanceof Error
                  ? fetchRecording.error.message
                  : t("The recording could not be found.")}
              </AlertDescription>
            </Alert>
          )}
        </div>
      )}
    </div>
  );
}
