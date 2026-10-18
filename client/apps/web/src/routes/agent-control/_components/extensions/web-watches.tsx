import { useApiMutation } from "@/hooks/use-api-mutation";
import { agentExtension } from "@/lib/queries/agent-extension";
import { assistant } from "@/lib/queries/assistant";
import { webWatchCadenceLabel, webWatchStatusAttrs } from "@/lib/web-watch";
import { apiService } from "@/services/api";
import type { WebWatch, WebWatchFindingsResponse } from "@/types/agent-extension";
import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { useQueryState } from "nuqs";
import { useEffect, useRef, useState } from "react";
import { WEB_WATCH_PARAM, webWatchParser } from "../../ai-control-tabs";
import { WebWatchesSkeleton } from "./web-watches-skeleton";

/**
 * Every web watch agents set, newest first: what it searches for and why, how often the vendor
 * looks and until when, what each check found, and what it has cost. A watch still looking can
 * be stopped here, which ends the wait the work had on it.
 */
export function WebWatches({ canUpdate }: { canUpdate: boolean }) {
  const t = useT();
  const watches = useQuery(agentExtension.webWatches());
  const [linkedWatch] = useQueryState(WEB_WATCH_PARAM, webWatchParser);

  if (watches.isPending) {
    return <WebWatchesSkeleton />;
  }

  const items = watches.data?.items ?? [];
  const linkedMissing =
    linkedWatch !== null && watches.isSuccess && !items.some((watch) => watch.id === linkedWatch);

  return (
    <section aria-label={t("Web watches")} className="flex flex-col gap-3">
      <h4 className="text-sm font-semibold">{t("Web watches")}</h4>
      {linkedMissing && (
        <Alert size="sm">
          <AlertDescription>
            {t("The web watch you opened is older than the ones listed here.")}
          </AlertDescription>
        </Alert>
      )}
      {watches.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{t("The web watches could not be loaded.")}</AlertDescription>
        </Alert>
      ) : items.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {t(
            "No web watches yet. When a person asks an agent to keep an eye on something, the watch is listed here with what it found.",
          )}
        </p>
      ) : (
        <ul className="divide-border flex flex-col divide-y rounded-md border">
          {items.map((watch) => (
            <WebWatchRow
              key={watch.id}
              watch={watch}
              linked={watch.id === linkedWatch}
              canUpdate={canUpdate}
            />
          ))}
        </ul>
      )}
    </section>
  );
}

function WebWatchRow({
  watch,
  linked,
  canUpdate,
}: {
  watch: WebWatch;
  linked: boolean;
  canUpdate: boolean;
}) {
  const t = useT();
  const [toggled, setToggled] = useState<boolean | null>(null);
  const open = toggled ?? linked;
  const rowRef = useRef<HTMLLIElement>(null);

  useEffect(() => {
    if (linked) {
      rowRef.current?.scrollIntoView({ block: "nearest" });
    }
  }, [linked]);

  const status = webWatchStatusAttrs(t)[watch.status];

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
        <span className="min-w-0 flex-1 truncate text-sm">{watch.purpose || watch.query}</span>
        <span className="text-muted-foreground shrink-0 text-xs">
          {t("{0, plural, one {# finding} other {# findings}}", watch.findingCount)}
        </span>
        <ChevronDownIcon
          className={cn("text-muted-foreground size-3.5 shrink-0", open && "rotate-180")}
        />
      </button>
      {open && <WebWatchDetail watch={watch} canUpdate={canUpdate} />}
    </li>
  );
}

function WebWatchDetail({ watch, canUpdate }: { watch: WebWatch; canUpdate: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const findings = useQuery(agentExtension.webWatchFindings(watch.id));
  const stop = useApiMutation<WebWatch, undefined>({
    resourceName: t("Web watch"),
    mutationFn: () => apiService.agentExtensionService.stopWebWatch(watch.id),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: agentExtension.webWatches._def }),
        queryClient.invalidateQueries({ queryKey: assistant.waits._def }),
      ]),
  });

  return (
    <div className="bg-muted/30 flex flex-col gap-3 px-3 pt-1 pb-3 text-sm">
      <dl className="grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1 text-xs">
        <dt className="text-muted-foreground">{t("Searches for")}</dt>
        <dd className="whitespace-pre-wrap">{watch.query}</dd>
        {watch.sites.length > 0 && (
          <>
            <dt className="text-muted-foreground">{t("Only on")}</dt>
            <dd>{watch.sites.join(", ")}</dd>
          </>
        )}
        <dt className="text-muted-foreground">{t("Looks")}</dt>
        <dd>{webWatchCadenceLabel(t, watch.cadence)}</dd>
        <dt className="text-muted-foreground">
          {watch.status === "Active" ? t("Ends") : t("Ended")}
        </dt>
        <dd>{formatUnixTime(watch.finishedAt ?? watch.endsAt)}</dd>
        {watch.lastRunAt ? (
          <>
            <dt className="text-muted-foreground">{t("Last looked")}</dt>
            <dd>{formatUnixTime(watch.lastRunAt)}</dd>
          </>
        ) : null}
        <dt className="text-muted-foreground">{t("Checks")}</dt>
        <dd>{watch.runCount}</dd>
        <dt className="text-muted-foreground">{t("Cost")}</dt>
        <dd>{formatCurrency(watch.costUsd)}</dd>
        <dt className="text-muted-foreground">{t("Set")}</dt>
        <dd>{formatUnixTime(watch.createdAt)}</dd>
      </dl>
      {watch.failure !== "" && (
        <Alert size="sm" variant={watch.status === "Failed" ? "destructive" : "default"}>
          <AlertDescription>{watch.failure}</AlertDescription>
        </Alert>
      )}
      <WebWatchFindings findings={findings} />
      {watch.status === "Active" && canUpdate && (
        <div className="flex justify-end">
          <Button
            type="button"
            size="sm"
            variant="outline"
            isLoading={stop.isPending}
            loadingText={t("Stopping")}
            onClick={() => stop.mutate(undefined)}
          >
            {t("Stop watching")}
          </Button>
        </div>
      )}
    </div>
  );
}

function WebWatchFindings({ findings }: { findings: UseQueryResult<WebWatchFindingsResponse> }) {
  const t = useT();

  if (findings.isPending) {
    return (
      <p className="text-muted-foreground text-xs" aria-busy>
        {t("Loading what it found")}
      </p>
    );
  }
  if (findings.isError) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription>{t("What the watch found could not be loaded.")}</AlertDescription>
      </Alert>
    );
  }
  const items = findings.data.items;
  if (items.length === 0) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("Nothing new yet. Each check that finds something is kept here.")}
      </p>
    );
  }

  return (
    <section aria-label={t("What it found")} className="flex flex-col gap-2">
      <span className="text-muted-foreground text-xs">
        {t("What it found, written outside your organization")}
      </span>
      <ol className="flex flex-col gap-2">
        {items.map((finding) => (
          <li
            key={finding.id}
            className="bg-background flex flex-col gap-1.5 rounded-md border p-2"
          >
            <div className="text-muted-foreground flex flex-wrap items-center gap-2 text-xs">
              <span>{formatUnixTime(finding.foundAt)}</span>
              {finding.deliveredAt ? null : <Badge variant="info">{t("Not yet passed on")}</Badge>}
            </div>
            {finding.summary !== "" && (
              <p className="text-xs whitespace-pre-wrap">{finding.summary}</p>
            )}
            {finding.results.length > 0 && (
              <ul className="flex flex-col gap-0.5 text-xs">
                {finding.results.map((result) => (
                  <li key={result.url} className="truncate">
                    <a
                      href={result.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="ui-focus-ring text-brand rounded-sm hover:underline"
                      title={result.url}
                    >
                      {result.title || result.url}
                    </a>
                    {result.publishedDate !== "" && (
                      <span className="text-muted-foreground"> · {result.publishedDate}</span>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </li>
        ))}
      </ol>
    </section>
  );
}
