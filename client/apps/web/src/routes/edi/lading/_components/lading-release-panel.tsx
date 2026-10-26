import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { lading } from "@/lib/queries/lading";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { LadingCheck } from "@trenova/shared/edi/lading";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { LadingProject, LadingReplay, LadingVersion } from "@trenova/shared/types/lading";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

const DAY = 86_400;
const REPLAY_DAYS = [7, 30, 90] as const;
const RUNNING_REPLAY_REFRESH_MS = 5_000;

/**
 * What stands between a version and live traffic: replays of past messages through it, compared
 * with the active version, and the shadow run beside the active version on live messages.
 */
export function LadingReleasePanel({
  project,
  version,
  versions,
  check,
  dirty,
}: {
  project: LadingProject;
  version: LadingVersion;
  versions: readonly LadingVersion[];
  check: LadingCheck;
  dirty: boolean;
}) {
  const t = useT();
  const active = versions.find((item) => item.id === project.activeVersionId);
  return (
    <div className="flex flex-col gap-5 p-3">
      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{t("Replays")}</h3>
        <p className="text-muted-foreground text-xs">
          {t(
            "A replay reads the partner's past messages through this version and the active one, and lists every difference. A version is certified only after each of its inbound maps replays without new rejections.",
          )}
        </p>
        <Replays version={version} active={active} check={check} dirty={dirty} />
      </section>
      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{t("Shadow")}</h3>
        <Shadow version={version} />
      </section>
    </div>
  );
}

function Replays({
  version,
  active,
  check,
  dirty,
}: {
  version: LadingVersion;
  active: LadingVersion | undefined;
  check: LadingCheck;
  dirty: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.EDI, Operation.Update);
  const inbound = check.maps.filter((map) => map.direction === "inbound");
  const [mapName, setMapName] = useState(() => inbound[0]?.name ?? "");
  const [days, setDays] = useState<(typeof REPLAY_DAYS)[number]>(30);

  const replays = useQuery({
    ...lading.replays(version.id),
    refetchInterval: (query) =>
      query.state.data?.some((replay: LadingReplay) => replay.status === "Running")
        ? RUNNING_REPLAY_REFRESH_MS
        : false,
  });

  const start = useApiMutation({
    mutationFn: () => {
      const to = Math.floor(Date.now() / 1000);
      return apiService.ladingService.startReplay(version.id, {
        map: mapName,
        from: to - days * DAY,
        to,
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: lading.replays(version.id).queryKey });
    },
  });

  const blocker = !active
    ? t("The project has no active version to replay against")
    : active.id === version.id
      ? t("The active version is not replayed against itself")
      : dirty
        ? t("Save the draft before replaying it")
        : check.errors > 0
          ? t("Fix the version's errors before replaying it")
          : inbound.length === 0
            ? t("The version has no inbound map to replay")
            : null;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <Select
          value={mapName}
          items={inbound.map((map) => ({ value: map.name, label: map.name }))}
          onValueChange={(next) => setMapName(next ?? "")}
        >
          <SelectTrigger className="w-48" aria-label={t("Map to replay")}>
            <SelectValue placeholder={t("Pick a map")} />
          </SelectTrigger>
          <SelectContent>
            {inbound.map((map) => (
              <SelectItem key={map.name} value={map.name}>
                {map.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={String(days)}
          items={REPLAY_DAYS.map((value) => ({
            value: String(value),
            label: t("{0, plural, one {Last # day} other {Last # days}}", value),
          }))}
          onValueChange={(next) => setDays(Number(next) as (typeof REPLAY_DAYS)[number])}
        >
          <SelectTrigger className="w-36" aria-label={t("Period")}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {REPLAY_DAYS.map((value) => (
              <SelectItem key={value} value={String(value)}>
                {t("{0, plural, one {Last # day} other {Last # days}}", value)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          type="button"
          size="sm"
          disabled={!canUpdate || blocker !== null || mapName === ""}
          isLoading={start.isPending}
          onClick={() => start.mutate(undefined)}
        >
          {t("Start replay")}
        </Button>
      </div>
      {blocker ? <p className="text-muted-foreground text-xs">{blocker}</p> : null}
      {replays.isPending ? (
        <div className="flex flex-col gap-2" aria-busy aria-label={t("Loading replays")}>
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      ) : replays.isError ? (
        <p className="text-danger text-sm">{t("The replays could not be loaded")}</p>
      ) : replays.data.length === 0 ? (
        <p className="text-muted-foreground text-sm">{t("This version has not been replayed")}</p>
      ) : (
        <ul className="flex flex-col divide-y rounded-md border">
          {replays.data.map((replay) => (
            <ReplayRow key={replay.id} replay={replay} />
          ))}
        </ul>
      )}
    </div>
  );
}

function ReplayRow({ replay }: { replay: LadingReplay }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const summary = replay.summary;
  const differing = replay.items.filter((item) => !item.same);

  return (
    <li className="flex flex-col gap-1 px-3 py-2 text-sm">
      <button
        type="button"
        className="ui-focus-ring flex flex-wrap items-baseline gap-x-3 text-left"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
      >
        <span className="font-medium">{replay.map}</span>
        <span className="text-muted-foreground text-xs">
          {t(
            "{0} to {1}",
            formatToUserTimezone(replay.periodStart),
            formatToUserTimezone(replay.periodEnd),
          )}
        </span>
        <span
          className={cn(
            "text-xs",
            replay.status === "Failed"
              ? "text-danger"
              : replay.status === "Running"
                ? "text-muted-foreground"
                : summary && summary.newlyRejected > 0
                  ? "text-danger"
                  : "text-success",
          )}
        >
          {replay.status === "Running"
            ? t("Running")
            : replay.status === "Failed"
              ? t("Failed")
              : summary
                ? t(
                    "{0} of {1} the same, {2} differ, {3} newly rejected",
                    summary.same,
                    summary.total,
                    summary.differ,
                    summary.newlyRejected,
                  )
                : t("Completed")}
        </span>
      </button>
      {replay.error ? <p className="text-danger text-xs">{replay.error}</p> : null}
      {open && differing.length > 0 ? (
        <ul className="flex flex-col gap-1 pl-3 text-xs">
          {differing.map((item) => (
            <li key={item.messageId} className="flex flex-col gap-0.5">
              <span className="font-mono">{item.messageId}</span>
              {item.error ? <span className="text-danger">{item.error}</span> : null}
              {(item.differences ?? []).map((difference) => (
                <span key={difference.path} className="text-muted-foreground font-mono">
                  {t(
                    "{0}: {1} → {2}",
                    difference.path,
                    JSON.stringify(difference.before),
                    JSON.stringify(difference.after),
                  )}
                </span>
              ))}
              {(item.newOutcomes ?? []).map((outcome, index) => (
                // oxlint-disable-next-line react/no-array-index-key -- outcomes can repeat
                <span key={index} className="text-danger">
                  {t("New: {0}", outcome.message)}
                </span>
              ))}
            </li>
          ))}
        </ul>
      ) : null}
    </li>
  );
}

function Shadow({ version }: { version: LadingVersion }) {
  const t = useT();
  const [onlyDiffering, setOnlyDiffering] = useState(true);
  const ran = version.status === "Shadow" || version.shadowUntil != null;
  const results = useQuery({
    ...lading.shadowResults(version.id, onlyDiffering),
    enabled: ran,
  });

  if (!ran) {
    return (
      <p className="text-muted-foreground text-xs">
        {version.status === "Certified"
          ? t(
              "Shadow this version from the version actions to run it beside the active one on live messages before activating it.",
            )
          : t("Only a certified version runs in shadow.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-3 text-xs">
        {version.shadowUntil ? (
          <span className="text-muted-foreground">
            {version.status === "Shadow"
              ? t("Shadowing until {0}", formatToUserTimezone(version.shadowUntil))
              : t("Shadowed until {0}", formatToUserTimezone(version.shadowUntil))}
          </span>
        ) : null}
        <label className="flex items-center gap-2">
          <Checkbox
            checked={onlyDiffering}
            onCheckedChange={(checked) => setOnlyDiffering(checked === true)}
          />
          {t("Only messages that differ")}
        </label>
      </div>
      {results.isPending ? (
        <div className="flex flex-col gap-2" aria-busy aria-label={t("Loading shadow results")}>
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
        </div>
      ) : results.isError ? (
        <p className="text-danger text-sm">{t("The shadow results could not be loaded")}</p>
      ) : (
        <>
          {results.data.summary ? (
            <p className="text-sm">
              {t(
                "{0} of {1} messages the same, {2} failed",
                results.data.summary.same,
                results.data.summary.total,
                results.data.summary.failed,
              )}
            </p>
          ) : null}
          {results.data.results.length === 0 ? (
            <p className="text-muted-foreground text-sm">
              {onlyDiffering
                ? t("No message has differed so far")
                : t("No message has arrived during the shadow so far")}
            </p>
          ) : (
            <ul className="flex flex-col divide-y rounded-md border text-xs">
              {results.data.results.map((result) => (
                <li key={result.id} className="flex flex-col gap-0.5 px-3 py-2">
                  <span className="flex gap-3">
                    <span className="font-mono">{result.transactionSet}</span>
                    <span className="text-muted-foreground">
                      {formatToUserTimezone(result.createdAt)}
                    </span>
                    <span className={result.same ? "text-success" : "text-warning"}>
                      {result.same ? t("Same") : t("Differs")}
                    </span>
                  </span>
                  {result.error ? <span className="text-danger">{result.error}</span> : null}
                  {(result.differences ?? []).map((difference) => (
                    <span key={difference.path} className="text-muted-foreground font-mono">
                      {t(
                        "{0}: {1} → {2}",
                        difference.path,
                        JSON.stringify(difference.before),
                        JSON.stringify(difference.after),
                      )}
                    </span>
                  ))}
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </div>
  );
}
