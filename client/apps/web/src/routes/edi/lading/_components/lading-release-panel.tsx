import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { lading } from "@/lib/queries/lading";
import { apiService } from "@/services/api";
import {
  ChevronRightIcon,
  ClockRewindIcon,
  EyeIcon,
  InfoCircleIcon,
  MinusCircleIcon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { LadingCheck } from "@trenova/shared/edi/lading";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type {
  LadingProject,
  LadingReplay,
  LadingShadowReport,
  LadingVersion,
} from "@trenova/shared/types/lading";
import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { Fragment, useId, useState, type ReactNode } from "react";
import {
  DiffList,
  PanelEmpty,
  PanelToolbar,
  ProportionBar,
  StatusIcon,
  type Status,
} from "./lading-panel-kit";
import { releaseStages, type ReleaseStage, type StageState } from "./studio-model";

const DAY = 86_400;
const REPLAY_DAYS = [7, 30, 90] as const;
const RUNNING_REPLAY_REFRESH_MS = 5_000;
const FIELDS_SHOWN = 8;

type View = "replays" | "shadow";
type ShadowResult = LadingShadowReport["results"][number];

/**
 * What stands between a version and live traffic. The pipeline across the top says how far the
 * version has come; below it, the replays of past messages through it against the active
 * version, or the messages its shadow read beside the active one, as a table with the selected
 * row's differences beside it.
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
  const active = versions.find((item) => item.id === project.activeVersionId);
  const inbound = check.maps.filter((map) => map.direction === "inbound");
  const shadowed = version.status === "Shadow" || version.shadowUntil != null;
  const [view, setView] = useState<View>(() =>
    version.status === "Shadow" ? "shadow" : "replays",
  );
  const [onlyDiffering, setOnlyDiffering] = useState(true);

  const replays = useQuery({
    ...lading.replays(version.id),
    refetchInterval: (query) =>
      query.state.data?.some((replay: LadingReplay) => replay.status === "Running")
        ? RUNNING_REPLAY_REFRESH_MS
        : false,
  });
  const shadow = useQuery({
    ...lading.shadowResults(version.id, onlyDiffering),
    enabled: shadowed,
  });

  const stages = releaseStages({
    version,
    activeVersionId: project.activeVersionId,
    dirty,
    inboundMaps: inbound.map((map) => map.name),
    replays: replays.data ?? [],
  });

  return (
    <div className="flex flex-col">
      <Pipeline stages={stages} />
      {view === "replays" ? (
        <Replays
          version={version}
          active={active}
          check={check}
          dirty={dirty}
          replays={replays}
          switcher={
            <ViewSwitch
              view={view}
              onChange={setView}
              replays={replays.data?.length}
              shadow={shadow.data?.summary?.total}
            />
          }
        />
      ) : (
        <Shadow
          version={version}
          shadowed={shadowed}
          results={shadow}
          onlyDiffering={onlyDiffering}
          onOnlyDiffering={setOnlyDiffering}
          switcher={
            <ViewSwitch
              view={view}
              onChange={setView}
              replays={replays.data?.length}
              shadow={shadow.data?.summary?.total}
            />
          }
        />
      )}
    </div>
  );
}

const STAGE_STYLE: Record<StageState, string> = {
  done: "text-foreground",
  running: "text-foreground",
  attention: "text-warning-foreground",
  failed: "text-danger-foreground",
  pending: "text-muted-foreground",
  skipped: "text-muted-foreground",
};

function StageMark({ state }: { state: StageState }) {
  switch (state) {
    case "done":
      return <StatusIcon status="passed" />;
    case "running":
      return <StatusIcon status="running" />;
    case "attention":
      return <StatusIcon status="warning" />;
    case "failed":
      return <StatusIcon status="failed" />;
    case "pending":
      return <StatusIcon status="pending" />;
    case "skipped":
      return <MinusCircleIcon aria-hidden className="text-muted-foreground size-3.5 shrink-0" />;
  }
}

/** The steps to live traffic, left to right, each with its mark and, on hover, why. */
function Pipeline({ stages }: { stages: readonly ReleaseStage[] }) {
  const t = useT();
  const label = (stage: ReleaseStage): { name: string; detail: string; why: string } => {
    switch (stage.key) {
      case "saved":
        return {
          name: t("Saved"),
          detail: stage.state === "done" ? "" : t("Unsaved changes"),
          why: t("A version is replayed and certified as it was saved."),
        };
      case "replayed":
        return {
          name: t("Replayed"),
          detail:
            stage.reason === "noActive"
              ? t("Nothing to compare with")
              : stage.reason === "isActive"
                ? t("This is the active version")
                : stage.reason === "noInbound"
                  ? t("No inbound map")
                  : t("{0} of {1} maps", stage.clean, stage.total),
          why: t(
            "Each inbound map replays the partner's past messages against the active version without a new rejection.",
          ),
        };
      case "certified":
        return {
          name: t("Certified"),
          detail: "",
          why: t(
            "No errors, every test passing, every flow clause tested and a clean replay. Certify from the version actions.",
          ),
        };
      case "shadow":
        return {
          name: t("Shadow"),
          detail:
            stage.state === "running"
              ? t("Running")
              : stage.state === "skipped"
                ? t("Skipped")
                : t("Optional"),
          why: t(
            "The certified version reads live messages beside the active one and sends nothing.",
          ),
        };
      case "live":
        return {
          name: t("Live"),
          detail: stage.superseded ? t("Since superseded") : "",
          why: t("Activating makes this the version that reads and writes the partner's messages."),
        };
    }
  };

  return (
    <ol
      aria-label={t("Release steps")}
      className="flex flex-wrap items-center gap-x-1.5 gap-y-1 border-b px-3 py-2"
    >
      {stages.map((stage, index) => {
        const { name, detail, why } = label(stage);
        return (
          <Fragment key={stage.key}>
            {index > 0 ? (
              <li aria-hidden className="text-muted-foreground">
                <ChevronRightIcon className="size-3.5" />
              </li>
            ) : null}
            <li>
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span
                      tabIndex={0}
                      className={cn(
                        "ui-focus-ring flex items-center gap-1.5 rounded-md px-1.5 py-1 text-xs",
                        STAGE_STYLE[stage.state],
                        stage.state === "running" && "bg-info-subtle",
                      )}
                    />
                  }
                >
                  <StageMark state={stage.state} />
                  <span className="font-medium">{name}</span>
                  {detail ? <span className="text-muted-foreground">{detail}</span> : null}
                </TooltipTrigger>
                <TooltipContent className="max-w-64">{why}</TooltipContent>
              </Tooltip>
            </li>
          </Fragment>
        );
      })}
    </ol>
  );
}

function ViewSwitch({
  view,
  onChange,
  replays,
  shadow,
}: {
  view: View;
  onChange: (view: View) => void;
  replays: number | undefined;
  shadow: number | undefined;
}) {
  const t = useT();
  const count = (n: number | undefined) =>
    n === undefined ? null : <span className="text-muted-foreground tabular-nums">{n}</span>;
  return (
    <SegmentedControl<View>
      value={view}
      aria-label={t("Show")}
      onValueChange={onChange}
      items={[
        {
          value: "replays",
          label: (
            <span className="flex items-center gap-1.5">
              {t("Replays")}
              {count(replays)}
            </span>
          ),
        },
        {
          value: "shadow",
          label: (
            <span className="flex items-center gap-1.5">
              {t("Shadow")}
              {count(shadow)}
            </span>
          ),
        },
      ]}
    />
  );
}

/** A table on the left and the selected row's detail beside it, stacked on a narrow panel. */
function TableWithDetail({ table, detail }: { table: ReactNode; detail: ReactNode }) {
  return (
    <div className="grid items-start md:grid-cols-[minmax(0,1fr)_minmax(18rem,26rem)]">
      <div className="min-w-0">{table}</div>
      <aside className="min-w-0 border-t md:border-t-0 md:border-l">{detail}</aside>
    </div>
  );
}

const REPLAY_COLUMNS = "grid-cols-[1.25rem_minmax(0,1.2fr)_minmax(0,1.4fr)_repeat(3,3.5rem)]";

function Replays({
  version,
  active,
  check,
  dirty,
  replays,
  switcher,
}: {
  version: LadingVersion;
  active: LadingVersion | undefined;
  check: LadingCheck;
  dirty: boolean;
  replays: UseQueryResult<LadingReplay[]>;
  switcher: ReactNode;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.EDI, Operation.Update);
  const inbound = check.maps.filter((map) => map.direction === "inbound");
  const [mapName, setMapName] = useState(() => inbound[0]?.name ?? "");
  const [days, setDays] = useState<(typeof REPLAY_DAYS)[number]>(30);
  const [picked, setPicked] = useState<string | null>(null);

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

  const list = replays.data ?? [];
  const selected = list.find((replay) => replay.id === picked) ?? list[0];

  return (
    <>
      <PanelToolbar
        end={
          <>
            <Select
              value={mapName}
              items={inbound.map((map) => ({ value: map.name, label: map.name }))}
              onValueChange={(next) => setMapName(next ?? "")}
            >
              <SelectTrigger className="w-44" aria-label={t("Map to replay")}>
                <SelectValue placeholder={t("Pick a map")} />
              </SelectTrigger>
              <SelectContent>
                {inbound.map((map) => (
                  <SelectItem key={map.name} value={map.name}>
                    <span className="font-mono text-xs">{map.name}</span>
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
              <ClockRewindIcon className="size-3.5" />
              {t("Replay")}
            </Button>
          </>
        }
      >
        {switcher}
      </PanelToolbar>
      {blocker ? (
        <Alert variant="info" size="sm" className="rounded-none border-x-0 border-t-0">
          <InfoCircleIcon />
          <AlertDescription>{blocker}</AlertDescription>
        </Alert>
      ) : null}
      {replays.isPending ? (
        <div className="flex flex-col gap-2 p-3" aria-busy aria-label={t("Loading replays")}>
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-7 w-full" />
        </div>
      ) : replays.isError ? (
        <PanelEmpty icon={XCircleIcon} tone="danger" title={t("The replays could not be loaded")} />
      ) : list.length === 0 ? (
        <PanelEmpty
          icon={ClockRewindIcon}
          title={t("This version has not been replayed")}
          description={t(
            "A replay reads the partner's past messages through this version and the active one, and lists every difference.",
          )}
        />
      ) : (
        <TableWithDetail
          table={
            <div className="text-sm">
              <div
                aria-hidden
                className={cn(
                  "text-muted-foreground grid items-center gap-3 border-b px-3 py-1.5 text-xs font-medium",
                  REPLAY_COLUMNS,
                )}
              >
                <span />
                <span>{t("Map")}</span>
                <span>{t("Period")}</span>
                <span className="text-right">
                  {t("Same")}
                </span>
                <span className="text-right">
                  {t("Differ")}
                </span>
                <span className="text-right">
                  {t("Rejected")}
                </span>
              </div>
              <ul aria-label={t("Replays")}>
                {list.map((replay) => (
                  <li key={replay.id}>
                    <button
                      type="button"
                      aria-current={replay.id === selected?.id ? "true" : undefined}
                      onClick={() => setPicked(replay.id)}
                      className={cn(
                        "ui-focus-ring grid w-full items-center gap-3 border-b px-3 py-1.5 text-left",
                        REPLAY_COLUMNS,
                        replay.id === selected?.id ? "bg-surface-active" : "hover:bg-surface-hover",
                      )}
                    >
                      <span>
                        <StatusIcon status={replayStatus(replay)} />
                      </span>
                      <span className="truncate font-mono text-xs">
                        {replay.map}
                      </span>
                      <span className="text-muted-foreground truncate text-xs">
                        {t(
                          "{0} to {1}",
                          formatToUserTimezone(replay.periodStart),
                          formatToUserTimezone(replay.periodEnd),
                        )}
                      </span>
                      {replay.summary ? (
                        <>
                          <Count value={replay.summary.same} />
                          <Count value={replay.summary.differ} tone="warning" />
                          <Count value={replay.summary.newlyRejected} tone="danger" />
                        </>
                      ) : (
                        <span
                          className={cn(
                            "col-span-3 text-right text-xs",
                            replay.status === "Failed" ? "text-danger" : "text-muted-foreground",
                          )}
                        >
                          {replay.status === "Running"
                            ? t("Running")
                            : replay.status === "Failed"
                              ? t("Failed")
                              : t("Completed")}
                        </span>
                      )}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          }
          detail={selected ? <ReplayDetail replay={selected} /> : null}
        />
      )}
    </>
  );
}

/** A count in a table cell: toned when there is any, a dash when there is none. */
function Count({ value, tone }: { value: number; tone?: "warning" | "danger" }) {
  return (
    <span
      className={cn(
        "text-right text-xs tabular-nums",
        value === 0
          ? "text-muted-foreground"
          : tone === "danger"
            ? "text-danger"
            : tone === "warning"
              ? "text-warning-foreground"
              : "",
      )}
    >
      {value === 0 && tone ? "—" : value}
    </span>
  );
}

function replayStatus(replay: LadingReplay): Status {
  if (replay.status === "Running") return "running";
  if (replay.status === "Failed") return "failed";
  const summary = replay.summary;
  if (!summary) return "passed";
  if (summary.newlyRejected > 0 || summary.failed > 0) return "failed";
  if (summary.differ > 0) return "warning";
  return "passed";
}

function percent(part: number, whole: number): number {
  return whole === 0 ? 100 : Math.round((part / whole) * 100);
}

function ReplayDetail({ replay }: { replay: LadingReplay }) {
  const t = useT();
  const summary = replay.summary;
  const differing = replay.items.filter((item) => !item.same);
  const fields = Object.entries(summary?.fields ?? {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, FIELDS_SHOWN);

  return (
    <div className="flex flex-col gap-3 p-3">
      <div className="flex flex-col gap-0.5">
        <span className="font-mono text-sm">{replay.map}</span>
        <span className="text-muted-foreground text-xs">
          {t(
            "{0} to {1}",
            formatToUserTimezone(replay.periodStart),
            formatToUserTimezone(replay.periodEnd),
          )}
        </span>
      </div>
      {replay.error ? <p className="text-danger text-xs">{replay.error}</p> : null}
      {summary ? (
        <>
          <div className="flex flex-col gap-1.5">
            <div className="flex items-baseline gap-2">
              <span className="text-xl font-semibold tabular-nums">
                {t("{0}%", percent(summary.same, summary.total))}
              </span>
              <span className="text-muted-foreground text-xs">
                {t("{0} of {1} messages the same", summary.same, summary.total)}
              </span>
            </div>
            <ProportionBar
              label={t("{0} of {1} messages the same", summary.same, summary.total)}
              parts={[
                { value: summary.same, tone: "success" },
                { value: summary.differ - summary.newlyRejected, tone: "warning" },
                { value: summary.newlyRejected, tone: "danger" },
                { value: summary.failed, tone: "neutral" },
              ]}
            />
          </div>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
            <Fact label={t("Differ")} value={summary.differ} tone="warning" />
            <Fact label={t("Newly rejected")} value={summary.newlyRejected} tone="danger" />
            <Fact label={t("Newly held")} value={summary.newlyHeld} tone="warning" />
            <Fact label={t("Newly declined")} value={summary.newlyDeclined} tone="warning" />
            <Fact label={t("Could not be read")} value={summary.failed} tone="danger" />
          </dl>
        </>
      ) : replay.status === "Running" ? (
        <p className="text-muted-foreground flex items-center gap-2 text-xs">
          <StatusIcon status="running" />
          {t("Reading the period's messages")}
        </p>
      ) : null}
      {fields.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          <span className="text-muted-foreground text-xs">{t("Fields that changed most")}</span>
          <ul className="flex flex-col gap-0.5 text-xs">
            {fields.map(([path, count]) => (
              <li key={path} className="flex items-center gap-2">
                <span className="min-w-0 flex-1 truncate font-mono">{path}</span>
                <span className="text-muted-foreground tabular-nums">{count}</span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {differing.length > 0 ? (
        <div className="flex flex-col gap-1.5">
          <span className="text-muted-foreground text-xs">
            {t("{0, plural, one {# message differs} other {# messages differ}}", differing.length)}
          </span>
          <ul className="flex flex-col gap-2">
            {differing.map((item) => (
              <li key={item.messageId}>
                <MessageChanges
                  title={<span className="font-mono">{item.messageId}</span>}
                  change={item}
                />
              </li>
            ))}
          </ul>
        </div>
      ) : replay.status === "Completed" ? (
        <p className="text-muted-foreground text-xs">{t("Every message read the same")}</p>
      ) : null}
    </div>
  );
}

function Fact({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "warning" | "danger";
}) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <dt className="text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          "tabular-nums",
          value === 0
            ? "text-muted-foreground"
            : tone === "danger"
              ? "text-danger"
              : "text-warning-foreground",
        )}
      >
        {value}
      </dd>
    </div>
  );
}

type MessageChange = {
  differences?: { path: string; before?: unknown; after?: unknown }[] | null;
  newOutcomes?: { kind: string; message: string; rule?: string | null }[] | null;
  goneOutcomes?: { kind: string; message: string; rule?: string | null }[] | null;
  error?: string | null;
};

/** How one message read differently: its error, the fields that changed, and rule outcomes. */
function MessageChanges({ title, change }: { title: ReactNode; change: MessageChange }) {
  const t = useT();
  return (
    <div className="flex flex-col gap-2 rounded-md border p-2.5 text-xs">
      <div className="flex items-center gap-2">{title}</div>
      {change.error ? <p className="text-danger">{change.error}</p> : null}
      {change.differences && change.differences.length > 0 ? (
        <DiffList differences={change.differences} />
      ) : null}
      {(change.newOutcomes ?? []).map((outcome, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- outcomes can repeat
        <div key={`new:${index}`} className="flex items-start gap-2">
          <Badge variant="danger">{t("New")}</Badge>
          <span className="min-w-0 flex-1">{outcome.message}</span>
          {outcome.rule ? <span className="text-muted-foreground">{outcome.rule}</span> : null}
        </div>
      ))}
      {(change.goneOutcomes ?? []).map((outcome, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- outcomes can repeat
        <div key={`gone:${index}`} className="flex items-start gap-2">
          <Badge variant="neutral" appearance="outline">
            {t("Gone")}
          </Badge>
          <span className="text-muted-foreground min-w-0 flex-1 line-through">
            {outcome.message}
          </span>
          {outcome.rule ? <span className="text-muted-foreground">{outcome.rule}</span> : null}
        </div>
      ))}
    </div>
  );
}

const SHADOW_COLUMNS = "grid-cols-[1.25rem_3.5rem_minmax(0,1fr)_minmax(0,10rem)_4.5rem]";

function Shadow({
  version,
  shadowed,
  results,
  onlyDiffering,
  onOnlyDiffering,
  switcher,
}: {
  version: LadingVersion;
  shadowed: boolean;
  results: UseQueryResult<LadingShadowReport>;
  onlyDiffering: boolean;
  onOnlyDiffering: (only: boolean) => void;
  switcher: ReactNode;
}) {
  const t = useT();
  const switchId = useId();
  const [picked, setPicked] = useState<string | null>(null);

  const toolbar = (
    <PanelToolbar
      end={
        shadowed ? (
          <>
            {version.shadowUntil ? (
              <span className="text-muted-foreground text-xs">
                {version.status === "Shadow"
                  ? t("Shadowing until {0}", formatToUserTimezone(version.shadowUntil))
                  : t("Shadowed until {0}", formatToUserTimezone(version.shadowUntil))}
              </span>
            ) : null}
            <label htmlFor={switchId} className="flex items-center gap-2 text-xs">
              <Switch
                id={switchId}
                size="sm"
                checked={onlyDiffering}
                onCheckedChange={(checked) => onOnlyDiffering(checked === true)}
              />
              {t("Only messages that differ")}
            </label>
          </>
        ) : null
      }
    >
      {switcher}
    </PanelToolbar>
  );

  if (!shadowed) {
    return (
      <>
        {toolbar}
        <PanelEmpty
          icon={EyeIcon}
          title={t("Not shadowed")}
          description={
            version.status === "Certified"
              ? t(
                  "Shadow this version from the version actions to run it beside the active one on live messages before activating it.",
                )
              : t("Only a certified version runs in shadow.")
          }
        />
      </>
    );
  }

  const report = results.data;
  const list = report?.results ?? [];
  const selected = list.find((result) => result.id === picked) ?? list[0];

  return (
    <>
      {toolbar}
      {results.isPending ? (
        <div className="flex flex-col gap-2 p-3" aria-busy aria-label={t("Loading shadow results")}>
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-7 w-full" />
        </div>
      ) : results.isError || !report ? (
        <PanelEmpty
          icon={XCircleIcon}
          tone="danger"
          title={t("The shadow results could not be loaded")}
        />
      ) : (
        <TableWithDetail
          table={
            list.length === 0 ? (
              <PanelEmpty
                compact
                icon={EyeIcon}
                title={
                  onlyDiffering
                    ? t("No message has differed so far")
                    : t("No message has arrived during the shadow so far")
                }
              />
            ) : (
              <div className="text-sm">
                <div
                  aria-hidden
                  className={cn(
                    "text-muted-foreground grid items-center gap-3 border-b px-3 py-1.5 text-xs font-medium",
                    SHADOW_COLUMNS,
                  )}
                >
                  <span />
                  <span>{t("Set")}</span>
                  <span>{t("Message")}</span>
                  <span>{t("Received")}</span>
                  <span className="text-right">
                    {t("Result")}
                  </span>
                </div>
                <ul aria-label={t("Shadow messages")}>
                  {list.map((result) => (
                    <li key={result.id}>
                      <button
                        type="button"
                        aria-current={result.id === selected?.id ? "true" : undefined}
                        onClick={() => setPicked(result.id)}
                        className={cn(
                          "ui-focus-ring grid w-full items-center gap-3 border-b px-3 py-1.5 text-left text-xs",
                          SHADOW_COLUMNS,
                          result.id === selected?.id ? "bg-surface-active" : "hover:bg-surface-hover",
                        )}
                      >
                        <span>
                          <StatusIcon status={shadowStatus(result)} />
                        </span>
                        <span className="font-mono">
                          {result.transactionSet}
                        </span>
                        <span className="truncate font-mono">
                          {result.messageId}
                        </span>
                        <span className="text-muted-foreground truncate">
                          {formatToUserTimezone(result.createdAt)}
                        </span>
                        <span
                          className={cn(
                            "text-right",
                            result.error
                              ? "text-danger"
                              : result.same
                                ? "text-muted-foreground"
                                : "text-warning-foreground",
                          )}
                        >
                          {result.error ? t("Failed") : result.same ? t("Same") : t("Differs")}
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              </div>
            )
          }
          detail={<ShadowDetail summary={report.summary} selected={selected} />}
        />
      )}
    </>
  );
}

function shadowStatus(result: ShadowResult): Status {
  if (result.error) return "failed";
  return result.same ? "passed" : "warning";
}

function ShadowDetail({
  summary,
  selected,
}: {
  summary: LadingShadowReport["summary"];
  selected: ShadowResult | undefined;
}) {
  const t = useT();
  return (
    <div className="flex flex-col gap-3 p-3">
      {summary && summary.total > 0 ? (
        <div className="flex flex-col gap-1.5">
          <div className="flex items-baseline gap-2">
            <span className="text-xl font-semibold tabular-nums">
              {t("{0}%", percent(summary.same, summary.total))}
            </span>
            <span className="text-muted-foreground text-xs">
              {t(
                "{0} of {1} messages the same, {2} failed",
                summary.same,
                summary.total,
                summary.failed,
              )}
            </span>
          </div>
          <ProportionBar
            label={t("{0} of {1} messages the same", summary.same, summary.total)}
            parts={[
              { value: summary.same, tone: "success" },
              { value: summary.total - summary.same - summary.failed, tone: "warning" },
              { value: summary.failed, tone: "danger" },
            ]}
          />
        </div>
      ) : (
        <p className="text-muted-foreground text-xs">
          {t("No message has arrived during the shadow so far")}
        </p>
      )}
      {selected ? (
        <MessageChanges
          title={
            <>
              <Badge variant="neutral" appearance="outline" className="font-mono">
                {selected.transactionSet}
              </Badge>
              <span className="truncate font-mono">{selected.messageId}</span>
            </>
          }
          change={selected}
        />
      ) : null}
      {selected && selected.same && !selected.error ? (
        <p className="text-muted-foreground text-xs">{t("Both versions read it the same")}</p>
      ) : null}
    </div>
  );
}
