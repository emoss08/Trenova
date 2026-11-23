import { initialsOf } from "@/components/record-header";
import { SearchInput } from "@/components/search-input";
import { SectionPanel } from "@/components/section-panel";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import { useWorkPane } from "@/components/work-pane";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { dayKey } from "@/lib/day-grouping";
import {
  fetchWorkerActivity,
  WORKER_ACTIVITY_KEY,
  type WorkerActivityEntry,
} from "@/lib/graphql/worker-activity";
import {
  entrySentence,
  changeChips,
} from "@/routes/admin/audit-logs/_components/audit-entry-sentence";
import { useWorkerAreas } from "@/routes/worker/worker-areas";
import type { WorkerTabProps, WorkerTarget } from "@/routes/worker/use-worker-state";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ArrowRightIcon, BotIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { dateTimeFormatter } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useNow } from "@trenova/shared/hooks/use-now";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useState } from "react";
import {
  ACTIVITY_KIND_LABELS,
  ACTIVITY_KINDS,
  ACTIVITY_SOURCE_LABELS,
  activityDays,
  activityFieldFilters,
  activityPeople,
  activitySource,
  actorName,
  drawableTarget,
  type ActivityKind,
  type ActorFilter,
} from "./activity-model";
import { ActivitySkeleton } from "./activity-skeleton";
import { ListMoreLine } from "../list-more";
import { LineSketch } from "../list-sketch";
import { useWorkerList } from "../use-worker-list";
import { SECONDS_IN_DAY } from "@/lib/time-units";

const PAGE_SIZE = 30;
const CHIP_LIMIT = 3;
const KINDS = Object.keys(ACTIVITY_KINDS) as ActivityKind[];

/** "Today", "Yesterday", else the weekday and date, the year only when it is not this one. */
function dayLabel(
  timestamp: number,
  key: string,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  const today = dayKey(now, timezone);
  if (key === today) return t("Today");
  if (key === dayKey(now - SECONDS_IN_DAY, timezone)) return t("Yesterday");
  return dateTimeFormatter({
    weekday: "long",
    month: "short",
    day: "numeric",
    year: key.slice(0, 4) === today.slice(0, 4) ? undefined : "numeric",
    timeZone: timezone,
  }).format(new Date(timestamp * 1000));
}

/**
 * A day's heading, pinned while its changes scroll under it. The minute clock lives here, the one
 * place that reads it, so "Today" turns into "Yesterday" at midnight without redrawing the list.
 */
function DayHeading({
  timestamp,
  dayKeyValue,
  count,
  timezone,
}: {
  timestamp: number;
  dayKeyValue: string;
  count: number;
  timezone: string;
}) {
  const t = useT();
  const now = useNow("minute");
  return (
    <h4 className="-mx-1.5 flex items-baseline gap-2 border-b border-border-subtle bg-card px-3.5 pt-2.5 pb-1.5 text-xs font-medium">
      {dayLabel(timestamp, dayKeyValue, now, timezone, t)}
      {count > 1 ? (
        <span className="font-normal text-foreground-subtle tabular-nums">
          {t("{0, plural, one {# change} other {# changes}}", count)}
        </span>
      ) : null}
    </h4>
  );
}

/**
 * The worker's whole file as a trail: every change to the worker and their sub-records, newest
 * first, grouped by day, in a bounded region that mounts only the changes in view and reads more
 * as it scrolls. Search, who and what narrow it on the server. An entry about a record still on
 * this page opens it.
 */
export default function WorkerActivity({
  workerId,
  onOpen,
}: Pick<WorkerTabProps, "workerId" | "onOpen">) {
  const t = useT();
  const timezone = useUserTimezone();
  const currentUserId = useAuthStore((state) => state.user?.id ?? null);
  const { isDrawable } = useWorkerAreas();
  const { closeThen } = useWorkPane();
  const [search, setSearch] = useState("");
  const [actor, setActor] = useState<ActorFilter>("all");
  const [kind, setKind] = useState<ActivityKind | "all">("all");
  const query = useDebounce(search, 250);
  const fieldFilters = activityFieldFilters(actor, kind, currentUserId);
  const filtered = query.trim() !== "" || actor !== "all" || kind !== "all";

  const activity = useWorkerList({
    root: WORKER_ACTIVITY_KEY,
    workerId,
    variables: [query.trim(), fieldFilters],
    pageSize: PAGE_SIZE,
    keepPreviousRows: true,
    fetchPage: ({ first, after, signal }) =>
      fetchWorkerActivity(workerId, { first, after }, { query, fieldFilters }, { signal }),
  });
  const entries = activity.rows;

  // Everyone the trail has shown, kept as the filters narrow it, so the person picked in "Who"
  // stays in the list it was picked from. Remembered from earlier renders, set during render.
  const [people, setPeople] = useState<{ id: string; name: string }[]>([]);
  const seen = activityPeople(entries);
  if (seen.some((person) => !people.some((known) => known.id === person.id))) {
    const merged = new Map(people.map((person) => [person.id, person.name]));
    for (const person of seen) merged.set(person.id, person.name);
    setPeople(
      [...merged.entries()]
        .map(([id, name]) => ({ id, name }))
        .sort((a, b) => a.name.localeCompare(b.name)),
    );
  }

  if (activity.isPending) return <ActivitySkeleton />;

  const clearFilters = () => {
    setSearch("");
    setActor("all");
    setKind("all");
  };
  const open = (target: WorkerTarget) => closeThen(() => onOpen(target));
  const others = people.filter((person) => person.id !== currentUserId);

  const actorItems: { value: ActorFilter; label: string }[] = [
    { value: "all", label: t("Everyone") },
    ...(currentUserId ? [{ value: "me" as ActorFilter, label: t("You") }] : []),
    { value: "dash", label: t("From Dash") },
    { value: "agents", label: t("Any agent") },
    ...others.map((person) => ({ value: `user:${person.id}` as ActorFilter, label: person.name })),
  ];
  const kindItems = [
    { value: "all", label: t("All changes") },
    ...KINDS.map((value) => ({ value, label: t(ACTIVITY_KIND_LABELS[value]) })),
  ];

  const rows: VirtualRow[] = activityDays(entries, timezone).flatMap((day) => [
    {
      key: `day:${day.key}`,
      sticky: true,
      render: () => (
        <DayHeading
          timestamp={day.entries[0].timestamp}
          dayKeyValue={day.key}
          count={day.entries.length}
          timezone={timezone}
        />
      ),
    },
    ...day.entries.map((entry, index) => ({
      key: entry.id,
      render: () => (
        <div className={cn("pb-px", index === 0 && "pt-1")}>
          <ActivityRow
            entry={entry}
            timezone={timezone}
            target={drawableTarget(entry, isDrawable)}
            onOpen={open}
          />
        </div>
      ),
    })),
  ]);

  return (
    <SectionPanel
      className="text-sm"
      title={t("Activity")}
      hint={
        activity.total !== null
          ? t("{0, plural, one {# change} other {# changes}}", activity.total)
          : undefined
      }
      action={
        <div className="flex flex-wrap items-center justify-end gap-2">
          <SearchInput
            value={search}
            onValueChange={setSearch}
            placeholder={t("Search changes")}
            className="w-56"
          />
          <Select
            value={actor}
            onValueChange={(value) => setActor((value ?? "all") as ActorFilter)}
            items={actorItems}
          >
            <SelectTrigger className="min-w-32" aria-label={t("Who")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("Everyone")}</SelectItem>
              {currentUserId ? <SelectItem value="me">{t("You")}</SelectItem> : null}
              <SelectItem value="dash">{t("From Dash")}</SelectItem>
              <SelectItem value="agents">{t("Any agent")}</SelectItem>
              {others.length > 0 ? (
                <>
                  <SelectSeparator />
                  <SelectGroup>
                    <SelectLabel>{t("People in this file")}</SelectLabel>
                    {others.map((person) => (
                      <SelectItem key={person.id} value={`user:${person.id}`}>
                        {person.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </>
              ) : null}
            </SelectContent>
          </Select>
          <Select
            value={kind}
            onValueChange={(value) => setKind((value ?? "all") as ActivityKind | "all")}
            items={kindItems}
          >
            <SelectTrigger className="min-w-32" aria-label={t("Kind of change")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {kindItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      }
    >
      {activity.isError && entries.length === 0 ? (
        <ErrorState
          error={activity.error}
          layout="sheet"
          title={t("The activity could not be loaded")}
          onRetry={activity.refetch}
        />
      ) : entries.length === 0 ? (
        filtered ? (
          <div className="flex flex-col items-center gap-2 px-4 py-7 text-xs text-muted-foreground">
            <p>{t("No changes match.")}</p>
            <Button size="sm" variant="secondary" onClick={clearFilters}>
              {t("Clear the filters")}
            </Button>
          </div>
        ) : (
          <EmptySheet
            title={t("No activity yet")}
            description={t(
              "Every change to this worker's file is written here as it happens: who made it, when, and what it changed.",
            )}
            sketch={<LineSketch widths={["w-2/3", "w-1/2", "w-3/4"]} />}
          />
        )
      ) : (
        <VirtualRows
          rows={rows}
          aria-label={t("Activity")}
          mask="card"
          className={cn("max-h-160 min-h-48", activity.isStale && "opacity-60 transition-opacity")}
          contentClassName="@container px-1.5 pb-1.5"
          hoverGlide
          rowNavigation
          estimateSize={56}
          onEndReached={
            activity.hasMore && !activity.loadMoreFailed ? activity.loadMore : undefined
          }
          footer={<ListMoreLine more={activity} />}
        />
      )}
      <p className="border-t border-border-subtle px-3 py-2.5 text-xs text-muted-foreground">
        {t(
          "Kept for as long as the record is. Changes from Dash and from agents are marked so you can tell who made them.",
        )}
      </p>
    </SectionPanel>
  );
}

function ActivityRow({
  entry,
  timezone,
  target,
  onOpen,
}: {
  entry: WorkerActivityEntry;
  timezone: string;
  target: WorkerTarget | null;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const source = activitySource(entry.channel);
  const name = actorName(entry) ?? t("Someone");
  const { chips, more } = changeChips(entry.changes, CHIP_LIMIT);
  const time = dateTimeFormatter({ hour: "numeric", minute: "2-digit", timeZone: timezone }).format(
    new Date(entry.timestamp * 1000),
  );
  const body = (
    <>
      <span className="pt-0.5 font-mono text-xs text-muted-foreground">{time}</span>
      <span className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden
          className="flex size-6 shrink-0 items-center justify-center rounded-md border border-border-subtle bg-sunken text-2xs font-medium text-muted-foreground"
        >
          {source === "agent" ? <BotIcon className="size-3" /> : initialsOf(name)}
        </span>
        <span className="truncate">{name}</span>
      </span>
      <span className="mt-0.5 justify-self-end rounded-sm border border-border-subtle px-1.5 text-2xs text-muted-foreground @3xl:order-last">
        {t(ACTIVITY_SOURCE_LABELS[source])}
      </span>
      <span className="col-span-full min-w-0 pt-0.5 text-pretty @3xl:col-span-1">
        {entrySentence({
          operation: entry.operation,
          resource: entry.resource,
          entityRef: entry.entityRef,
        })}
        {chips.map((chip) => (
          <span
            key={chip.path}
            className="mx-0.5 inline-flex max-w-full flex-wrap items-center gap-1 rounded-md border border-border-subtle bg-sunken px-1.5 py-px align-[1px] text-xs"
          >
            <span className="text-muted-foreground">{chip.field}</span>
            {chip.from !== null ? (
              <>
                <del className="text-muted-foreground decoration-foreground-subtle">
                  {chip.from}
                </del>
                {chip.to !== null ? (
                  <ArrowRightIcon className="size-3 text-muted-foreground" />
                ) : null}
              </>
            ) : null}
            {chip.to !== null ? <ins className="font-medium no-underline">{chip.to}</ins> : null}
          </span>
        ))}
        {more > 0 ? (
          <span className="text-xs text-muted-foreground">
            {t("{0, plural, one {and # more field} other {and # more fields}}", more)}
          </span>
        ) : null}
        {entry.comment ? (
          <span className="block text-xs text-muted-foreground">{entry.comment}</span>
        ) : null}
      </span>
    </>
  );
  const layout =
    "grid w-full grid-cols-[4rem_minmax(0,1fr)_auto] items-start gap-x-3 gap-y-1 rounded-md p-2 text-left @3xl:grid-cols-[4.5rem_10.5rem_minmax(0,1fr)_auto]";

  return target ? (
    <button
      type="button"
      data-glide-target
      data-testid={`activity-${entry.id}`}
      onClick={() => onOpen(target)}
      className={cn(layout, "cursor-pointer ui-focus-ring")}
    >
      {body}
    </button>
  ) : (
    <div data-glide-target data-testid={`activity-${entry.id}`} className={layout}>
      {body}
    </div>
  );
}
