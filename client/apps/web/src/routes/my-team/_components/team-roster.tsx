import { useT } from "@trenova/shared/i18n/use-t";
import {
  matchesTeamSearch,
  needsAttention,
  TEAM_PATH_LABELS,
  type ClassifiedMember,
  type CoverSource,
  type TeamPath,
} from "@/lib/my-team";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { Switch } from "@trenova/shared/components/ui/switch";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { approvalScopeLabel } from "@trenova/shared/lib/org-structure";
import { formatTenure } from "@trenova/shared/lib/tenure";
import { cn } from "@trenova/shared/lib/utils";
import { AlertTriangleIcon, ChevronRightIcon, SearchIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useId, useMemo, useState } from "react";
import { Link } from "react-router";
import { HealthTrio, MemberIdentity, memberHref } from "./member-identity";

type RosterView = "all" | TeamPath;

type TeamRosterProps = {
  rows: readonly ClassifiedMember[];
  now: number;
  includeInactive: boolean;
  onIncludeInactiveChange: (value: boolean) => void;
};

const PATH_ORDER: TeamPath[] = ["direct", "terminal", "covering"];

const STAGGER_LIMIT = 12;

export function TeamRoster({
  rows,
  now,
  includeInactive,
  onIncludeInactiveChange,
}: TeamRosterProps) {
  const t = useT();

  const [query, setQuery] = useState("");
  const [view, setView] = useState<RosterView>("all");
  const [attentionOnly, setAttentionOnly] = useState(false);
  const inactiveId = useId();

  const counts = useMemo(() => {
    const result: Record<TeamPath, number> = { direct: 0, terminal: 0, covering: 0 };
    for (const row of rows) result[row.path] += 1;
    return result;
  }, [rows]);

  const viewItems = useMemo<SegmentedControlItem<RosterView>[]>(() => {
    const items: SegmentedControlItem<RosterView>[] = [
      { value: "all", label: "Everyone", caption: String(rows.length) },
      { value: "direct", label: "Direct", caption: String(counts.direct) },
      { value: "terminal", label: "Terminal", caption: String(counts.terminal) },
    ];
    if (counts.covering > 0) {
      items.push({ value: "covering", label: "Covering", caption: String(counts.covering) });
    }
    return items;
  }, [counts, rows.length]);

  const activeView: RosterView = view === "covering" && counts.covering === 0 ? "all" : view;

  const visible = useMemo(
    () =>
      rows.filter(
        (row) =>
          (activeView === "all" || row.path === activeView) &&
          (!attentionOnly || needsAttention(row.member)) &&
          matchesTeamSearch(row.member, query),
      ),
    [rows, activeView, attentionOnly, query],
  );

  const groups = useMemo(() => groupRows(visible), [visible]);
  const filtered = query.trim().length > 0 || attentionOnly || activeView !== "all";

  return (
    <section aria-labelledby="team-roster-heading" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h3 id="team-roster-heading" className="sr-only">
            {t("Roster")}
          </h3>
          <Input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("Find someone by name, title or terminal")}
            aria-label={t("Find someone")}
            leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
            inputContainerClassName="w-72 max-w-full"
          />
          <Button
            size="sm"
            variant={attentionOnly ? "default" : "outline"}
            aria-pressed={attentionOnly}
            onClick={() => setAttentionOnly((current) => !current)}
          >
            <AlertTriangleIcon className="size-3.5" />
            {t("Needs attention")}
          </Button>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <SegmentedControl<RosterView>
            items={viewItems}
            value={activeView}
            onValueChange={setView}
            aria-label={t("Who to show")}
          />
          <div className="flex items-center gap-2">
            <Switch
              id={inactiveId}
              size="sm"
              checked={includeInactive}
              onCheckedChange={onIncludeInactiveChange}
            />
            <Label htmlFor={inactiveId} className="text-muted-foreground text-xs font-normal">
              {t("Include people who have left")}
            </Label>
          </div>
        </div>
      </div>

      {visible.length === 0 ? (
        <div className="text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed px-4 py-8 text-center text-sm">
          <p>
            {rows.length === 0
              ? t("Nobody is on your team.")
              : attentionOnly && !query
                ? t("Nobody in this view needs attention.")
                : t("Nobody matches that.")}
          </p>
          {filtered ? (
            <Button
              size="xs"
              variant="ghost"
              onClick={() => {
                setQuery("");
                setAttentionOnly(false);
                setView("all");
              }}
            >
              {t("Clear filters")}
            </Button>
          ) : null}
        </div>
      ) : (
        groups.map((group) => (
          <RosterGroup key={group.key} group={group} now={now} showHeading={activeView === "all"} />
        ))
      )}
    </section>
  );
}

type RosterGroupModel = {
  key: string;
  path: TeamPath;
  cover: CoverSource | null;
  rows: ClassifiedMember[];
};

/**
 * One section per door, and one per manager being covered: two delegations
 * are two different people's teams, and a reader has to know whose is whose.
 */
function groupRows(rows: readonly ClassifiedMember[]): RosterGroupModel[] {
  const groups = new Map<string, RosterGroupModel>();
  for (const row of rows) {
    const key = row.path === "covering" ? `covering:${row.coveringFor?.id ?? ""}` : row.path;
    let group = groups.get(key);
    if (!group) {
      group = { key, path: row.path, cover: row.coveringFor, rows: [] };
      groups.set(key, group);
    }
    group.rows.push(row);
  }
  return Array.from(groups.values()).sort(
    (a, b) =>
      PATH_ORDER.indexOf(a.path) - PATH_ORDER.indexOf(b.path) ||
      (a.cover?.name ?? "").localeCompare(b.cover?.name ?? ""),
  );
}

function RosterGroup({
  group,
  now,
  showHeading,
}: {
  group: RosterGroupModel;
  now: number;
  showHeading: boolean;
}) {
  const t = useT();

  const heading =
    group.path === "covering"
      ? `${TEAM_PATH_LABELS.covering} ${group.cover?.name ?? "a manager"}`
      : TEAM_PATH_LABELS[group.path];

  return (
    <section aria-label={heading} className="flex flex-col gap-1.5">
      {showHeading ? (
        <header className="flex flex-wrap items-baseline justify-between gap-2 px-1">
          <h4 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
            {heading}
            <span className="ml-1.5 font-normal normal-case tabular-nums">{group.rows.length}</span>
          </h4>
          {group.cover ? (
            <span className="text-muted-foreground text-xs">
              {approvalScopeLabel(group.cover.scope)} ·{" "}
              {group.cover.endsAt
                ? t("until {0}", formatUnixDate(group.cover.endsAt))
                : t("until called back")}
            </span>
          ) : null}
        </header>
      ) : null}
      <MemberList rows={group.rows} now={now} />
    </section>
  );
}

function MemberList({ rows, now }: { rows: ClassifiedMember[]; now: number }) {
  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);

  // The stagger belongs to the first paint only. Filtering re-keys nothing, so
  // a row that survives a filter change must not replay its entrance.
  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <ul className="bg-card divide-y overflow-hidden rounded-lg border">
      {rows.map((row, index) => (
        <m.li
          key={row.member.workerId}
          initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.25, delay: Math.min(index, STAGGER_LIMIT) * 0.03 }}
        >
          <MemberRow row={row} now={now} />
        </m.li>
      ))}
    </ul>
  );
}

function MemberRow({ row, now }: { row: ClassifiedMember; now: number }) {
  const { member } = row;
  const left = member.status !== "Active";
  const tenure = formatTenure(member.hireDate, member.terminationDate, now);

  return (
    <Link
      to={memberHref(member.workerId)}
      className={cn(
        "group/row hover:bg-accent grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 transition-colors",
        left && "opacity-70 hover:opacity-100",
      )}
    >
      <MemberIdentity member={member} />
      <div className="flex items-center gap-4">
        <HealthTrio member={member} className="hidden md:flex" />
        <span
          className="text-muted-foreground hidden w-14 text-right text-xs tabular-nums sm:inline"
          title={
            left && member.terminationDate
              ? `${formatUnixDate(member.hireDate)} – ${formatUnixDate(member.terminationDate)}`
              : `Since ${formatUnixDate(member.hireDate)}`
          }
        >
          {tenure}
        </span>
        <ChevronRightIcon
          aria-hidden
          className="text-muted-foreground size-4 transition-transform group-hover/row:translate-x-0.5"
        />
      </div>
    </Link>
  );
}
