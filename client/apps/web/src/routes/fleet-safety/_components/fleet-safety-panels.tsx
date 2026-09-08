import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { terminalStandings } from "@/lib/fleet-safety-console";
import type { FleetSafetyRankRow, FleetSafetyTerminalRow } from "@/lib/graphql/fleet-safety";
import { Avatar, AvatarBadge, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { safetyRatingLabel, safetyRatingTone } from "@trenova/shared/lib/csa";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { AwardIcon, Building2Icon, ShieldAlertIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";

const STAGGER_LIMIT = 8;

type TerminalsPanelProps = {
  terminals: readonly FleetSafetyTerminalRow[];
  totalWorkers: number;
  selected: string;
  onSelect: (fleetCodeId: string) => void;
};

/**
 * Terminals with the most flagged drivers first. Clicking one narrows every
 * section to it, which is the question a safety director asks next: is it
 * the fleet, or is it one yard.
 */
export function TerminalsPanel({
  terminals,
  totalWorkers,
  selected,
  onSelect,
}: TerminalsPanelProps) {
  const standings = useMemo(
    () => terminalStandings(terminals, totalWorkers),
    [terminals, totalWorkers],
  );

  return (
    <SectionPanel
      title="By terminal"
      icon={<Building2Icon />}
      hint={`${totalWorkers} drivers`}
      help="Drivers by terminal, the yard with the most at-risk drivers first. Choose one to narrow every section on the page to it."
    >
      {standings.length === 0 ? (
        <SectionPanelQuiet>No active drivers.</SectionPanelQuiet>
      ) : (
        <ul className="divide-y">
          {standings.map(({ terminal, flagged, share }) => {
            const id = terminal.fleetCodeId ?? "";
            const active = Boolean(id) && id === selected;
            return (
              <li key={terminal.fleetCodeId ?? "unassigned"}>
                <button
                  type="button"
                  disabled={!id}
                  aria-pressed={active}
                  onClick={() => onSelect(active ? "" : id)}
                  className={cn(
                    "flex w-full flex-col gap-1.5 px-3 py-2 text-left transition-colors",
                    id && "hover:bg-accent",
                    active && "bg-accent",
                  )}
                >
                  <span className="flex items-center justify-between gap-2 text-xs">
                    <span className="flex min-w-0 items-center gap-2">
                      <span
                        aria-hidden
                        className={cn(
                          "size-2 shrink-0 rounded-full",
                          !terminal.color && "bg-muted-foreground/40",
                        )}
                        style={terminal.color ? { backgroundColor: terminal.color } : undefined}
                      />
                      <span className="truncate font-medium">{terminal.code || "No terminal"}</span>
                      {terminal.description ? (
                        <span className="text-muted-foreground truncate">
                          {terminal.description}
                        </span>
                      ) : null}
                    </span>
                    <span className="flex shrink-0 items-center gap-1.5">
                      {terminal.atRisk > 0 ? (
                        <Badge variant="inactive">{terminal.atRisk} at risk</Badge>
                      ) : null}
                      {terminal.watch > 0 ? (
                        <Badge variant="warning">{terminal.watch} watch</Badge>
                      ) : null}
                      <span className="text-muted-foreground tabular-nums">
                        {terminal.workers} · avg {terminal.averageScore}
                      </span>
                    </span>
                  </span>
                  <span
                    role="img"
                    aria-label={`${terminal.code || "No terminal"}: ${terminal.workers} of ${totalWorkers} drivers, ${flagged} flagged`}
                    className="bg-muted flex h-1 w-full overflow-hidden rounded-full"
                  >
                    <span
                      aria-hidden
                      className="bg-brand/50 h-full rounded-full transition-[width] duration-700 ease-out motion-reduce:transition-none"
                      style={{ width: `${share}%` }}
                    />
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </SectionPanel>
  );
}

type RankListProps = {
  title: string;
  kind: "worst" | "best";
  empty: string;
  rows: readonly FleetSafetyRankRow[];
};

/**
 * One end of the ranking. The two lists are the same query read from
 * opposite ends, so a driver never appears on both.
 */
export function RankList({ title, kind, empty, rows }: RankListProps) {
  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <SectionPanel
      title={title}
      icon={kind === "worst" ? <ShieldAlertIcon /> : <AwardIcon />}
      help={
        kind === "worst"
          ? "The drivers carrying the most active points, worst first. Points roll off two years after the event."
          : "The cleanest records on the fleet. The two lists read the same ranking from opposite ends, so nobody appears on both."
      }
      hint={rows.length > 0 ? `${rows.length} drivers` : undefined}
    >
      {rows.length === 0 ? (
        <SectionPanelQuiet>{empty}</SectionPanelQuiet>
      ) : (
        <ul className="divide-y">
          {rows.map((row, index) => (
            <m.li
              key={row.workerId}
              initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.25, delay: Math.min(index, STAGGER_LIMIT) * 0.03 }}
            >
              <Link
                to={`/hr/workers?entityId=${row.workerId}&modType=edit&tab=safety`}
                className="hover:bg-accent grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 transition-colors"
              >
                <span className="flex min-w-0 items-center gap-2.5">
                  <Avatar size="sm">
                    <AvatarFallback className="text-2xs font-medium">
                      {getNameInitials(row.name)}
                    </AvatarFallback>
                    {row.fleetColor ? (
                      <AvatarBadge aria-hidden style={{ backgroundColor: row.fleetColor }} />
                    ) : null}
                  </Avatar>
                  <span className="flex min-w-0 flex-col leading-tight">
                    <span className="flex min-w-0 items-center gap-1.5">
                      <span className="truncate text-sm font-medium">{row.name}</span>
                      {row.fleetCode ? (
                        <span className="text-muted-foreground text-2xs">{row.fleetCode}</span>
                      ) : null}
                    </span>
                    <span className="text-muted-foreground text-xs tabular-nums">
                      {row.events} event{row.events === 1 ? "" : "s"}
                      {row.lastEventAt ? ` · last ${formatUnixDate(row.lastEventAt)}` : ""}
                    </span>
                  </span>
                </span>
                <span className="flex items-center gap-2">
                  <span className="text-right text-xs tabular-nums">
                    <span className="font-mono font-medium">{row.score}</span>
                    {row.activePoints > 0 ? (
                      <span className="text-muted-foreground"> · {row.activePoints} pts</span>
                    ) : null}
                  </span>
                  <Badge variant={safetyRatingTone(row.rating)}>
                    {safetyRatingLabel(row.rating)}
                  </Badge>
                </span>
              </Link>
            </m.li>
          ))}
        </ul>
      )}
    </SectionPanel>
  );
}
