import { usePermission } from "@/hooks/use-permission";
import {
  fetchFleetSafety,
  FLEET_SAFETY_KEY,
  type FleetSafetyRankRow,
} from "@/lib/graphql/fleet-safety";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  csaBarWidth,
  csaBasicHint,
  csaBasicLabel,
  csaBasicTone,
  safetyEventKindLabel,
  safetyRatingLabel,
  safetyRatingTone,
  trendDirection,
} from "@trenova/shared/lib/csa";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { TrendingDownIcon, TrendingUpIcon, MinusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

const WINDOW_OPTIONS = [
  { value: "3", label: "Last 3 months" },
  { value: "6", label: "Last 6 months" },
  { value: "12", label: "Last 12 months" },
  { value: "24", label: "Last 24 months" },
];

export default function FleetSafetyConsole() {
  const { allowed: canRead } = usePermission(Resource.WorkerSafetyEvent, Operation.Read);
  const [windowMonths, setWindowMonths] = useState("12");
  const [fleetCodeId, setFleetCodeId] = useState<string>("");

  const fleetQuery = useQuery({
    queryKey: [FLEET_SAFETY_KEY, windowMonths, fleetCodeId],
    queryFn: ({ signal }) =>
      fetchFleetSafety(
        {
          windowMonths: Number(windowMonths),
          fleetCodeId: fleetCodeId || null,
          rankLimit: 10,
        },
        { signal },
      ),
    enabled: canRead,
  });

  const summary = fleetQuery.data;

  const worstBasicScore = useMemo(
    () => Math.max(0, ...(summary?.basics ?? []).map((basic) => basic.weightedScore)),
    [summary],
  );
  const peakTrend = useMemo(
    () => Math.max(1, ...(summary?.trend ?? []).map((point) => point.events)),
    [summary],
  );
  const direction = useMemo(
    () => trendDirection((summary?.trend ?? []).map((point) => point.events)),
    [summary],
  );

  // Safety events carry accidents, citations and discipline. Somebody without
  // the grant sees nothing rather than an empty page implying a clean fleet.
  if (!canRead) return null;

  if (fleetQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }
  if (!summary) return null;

  const terminalOptions = summary.terminals.filter((terminal) => terminal.fleetCodeId);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <Select value={windowMonths} onValueChange={(value) => setWindowMonths(value ?? "12")}>
            <SelectTrigger className="w-44" size="sm">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {WINDOW_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {terminalOptions.length > 0 ? (
            <Select
              value={fleetCodeId || "all"}
              onValueChange={(value) => setFleetCodeId(!value || value === "all" ? "" : value)}
            >
              <SelectTrigger className="w-52" size="sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Every terminal</SelectItem>
                {terminalOptions.map((terminal) => (
                  <SelectItem key={terminal.fleetCodeId} value={terminal.fleetCodeId as string}>
                    {terminal.code || "Unassigned"}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : null}
        </div>
        <p className="text-muted-foreground text-xs">As of {formatUnixDate(summary.asOf)}</p>
      </div>

      <section className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <Figure label="Drivers" value={String(summary.workers)} />
        <Figure label="Average score" value={String(summary.averageScore)} />
        <Figure
          label="At risk"
          value={String(summary.atRisk)}
          tone={summary.atRisk > 0 ? "critical" : undefined}
        />
        <Figure
          label="On watch"
          value={String(summary.watch)}
          tone={summary.watch > 0 ? "warning" : undefined}
        />
        <Figure
          label="Events"
          value={String(summary.totalEvents)}
          detail={`${summary.openEvents} open`}
        />
        <Figure
          label="Out of service"
          value={String(summary.outOfServiceOrders)}
          tone={summary.outOfServiceOrders > 0 ? "critical" : undefined}
        />
      </section>

      <section className="rounded-lg border p-4">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div>
            <h3 className="text-sm font-medium">CSA BASICs</h3>
            <p className="text-muted-foreground text-xs">
              Severity, plus two for an out-of-service order, weighted three times inside six months
              and twice inside a year. Bars are relative to this fleet&apos;s own worst category,
              not to a national percentile.
            </p>
          </div>
        </div>

        {summary.basicsInferred ? (
          <Alert variant="warning" className="mt-3">
            <AlertDescription>
              Some categories were reached from the kind of event rather than from violations
              somebody keyed in. Record the violation codes off an inspection report and these
              become the real thing.
            </AlertDescription>
          </Alert>
        ) : null}

        <ul className="mt-3 flex flex-col gap-2">
          {summary.basics.map((basic) => {
            const tone = csaBasicTone(basic.weightedScore, worstBasicScore);
            return (
              <li key={basic.basic} className="text-xs">
                <div className="flex items-center justify-between gap-3">
                  <span className="flex items-center gap-2">
                    <span className="font-medium">{csaBasicLabel(basic.basic)}</span>
                    {basic.inferred ? <Badge variant="secondary">Inferred</Badge> : null}
                    {basic.outOfService > 0 ? (
                      <Badge variant="inactive">{basic.outOfService} OOS</Badge>
                    ) : null}
                  </span>
                  <span className="text-muted-foreground tabular-nums">
                    {basic.weightedScore}
                    {basic.violations > 0
                      ? ` · ${basic.violations} violation${basic.violations === 1 ? "" : "s"}`
                      : basic.events > 0
                        ? ` · ${basic.events} event${basic.events === 1 ? "" : "s"}`
                        : ""}
                  </span>
                </div>
                <div className="bg-muted mt-1 h-2 w-full overflow-hidden rounded-full">
                  <div
                    className={cn(
                      "h-full rounded-full transition-all",
                      tone === "critical" && "bg-red-500",
                      tone === "warning" && "bg-amber-500",
                      tone === "muted" && "bg-muted-foreground/40",
                    )}
                    style={{ width: `${csaBarWidth(basic.weightedScore, worstBasicScore)}%` }}
                  />
                </div>
                <p className="text-muted-foreground mt-0.5 text-[11px]">
                  {csaBasicHint(basic.basic)}
                </p>
              </li>
            );
          })}
        </ul>
      </section>

      <div className="grid gap-4 lg:grid-cols-2">
        <section className="rounded-lg border p-4">
          <div className="flex items-center justify-between gap-2">
            <h3 className="text-sm font-medium">Events by month</h3>
            <span className="text-muted-foreground flex items-center gap-1 text-xs">
              {direction === "up" ? (
                <TrendingUpIcon className="size-3.5 text-red-500" />
              ) : direction === "down" ? (
                <TrendingDownIcon className="size-3.5 text-green-600" />
              ) : (
                <MinusIcon className="size-3.5" />
              )}
              {direction === "up" ? "Rising" : direction === "down" ? "Falling" : "Holding steady"}
            </span>
          </div>
          {summary.trend.length === 0 ? (
            <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
              No safety events in this window.
            </p>
          ) : (
            <div className="mt-4 flex h-32 items-end gap-1">
              {summary.trend.map((point) => (
                <div
                  key={point.periodStart}
                  className="flex flex-1 flex-col items-center justify-end gap-1"
                  title={`${point.events} events · ${point.accidents} accidents · ${point.outOfService} out of service`}
                >
                  <span className="text-muted-foreground text-[10px] tabular-nums">
                    {point.events}
                  </span>
                  <div
                    className={cn(
                      "w-full rounded-t-sm",
                      point.preventable > 0 ? "bg-red-500/70" : "bg-primary/60",
                    )}
                    style={{ height: `${Math.max(4, (point.events / peakTrend) * 100)}%` }}
                  />
                  <span className="text-muted-foreground text-[10px]">
                    {formatUnixDate(point.periodStart).slice(0, 3)}
                  </span>
                </div>
              ))}
            </div>
          )}

          <ul className="mt-4 flex flex-col gap-1 border-t pt-3">
            {summary.kinds.map((kind) => (
              <li key={kind.kind} className="flex items-center justify-between text-xs">
                <span>{safetyEventKindLabel(kind.kind)}</span>
                <span className="text-muted-foreground tabular-nums">
                  {kind.events}
                  {kind.preventable > 0 ? ` · ${kind.preventable} preventable` : ""}
                  {kind.open > 0 ? ` · ${kind.open} open` : ""}
                </span>
              </li>
            ))}
          </ul>
        </section>

        <section className="rounded-lg border p-4">
          <h3 className="text-sm font-medium">By terminal</h3>
          {summary.terminals.length === 0 ? (
            <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
              No active drivers.
            </p>
          ) : (
            <ul className="mt-3 flex flex-col gap-2">
              {summary.terminals.map((terminal) => (
                <li
                  key={terminal.fleetCodeId ?? "unassigned"}
                  className="flex items-center justify-between gap-2 text-xs"
                >
                  <span className="flex min-w-0 items-center gap-2">
                    <span
                      className="size-2 shrink-0 rounded-full"
                      style={{ backgroundColor: terminal.color || "var(--muted-foreground)" }}
                    />
                    <span className="truncate font-medium">{terminal.code || "No terminal"}</span>
                    {terminal.description ? (
                      <span className="text-muted-foreground truncate">{terminal.description}</span>
                    ) : null}
                  </span>
                  <span className="flex shrink-0 items-center gap-2 tabular-nums">
                    {terminal.atRisk > 0 ? (
                      <Badge variant="inactive">{terminal.atRisk} at risk</Badge>
                    ) : null}
                    {terminal.watch > 0 ? (
                      <Badge variant="warning">{terminal.watch} watch</Badge>
                    ) : null}
                    <span className="text-muted-foreground">
                      {terminal.workers} · avg {terminal.averageScore}
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <RankList title="Needs attention" empty="Nobody is carrying points." rows={summary.worst} />
        <RankList title="Best records" empty="No drivers to rank." rows={summary.best} />
      </div>
    </div>
  );
}

function RankList({
  title,
  empty,
  rows,
}: {
  title: string;
  empty: string;
  rows: FleetSafetyRankRow[];
}) {
  return (
    <section className="rounded-lg border p-4">
      <h3 className="text-sm font-medium">{title}</h3>
      {rows.length === 0 ? (
        <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
          {empty}
        </p>
      ) : (
        <ul className="mt-3 flex flex-col gap-1.5">
          {rows.map((row) => (
            <li key={row.workerId} className="flex items-center justify-between gap-2 text-xs">
              <Link
                to={`/hr/workers?entityId=${row.workerId}&modType=edit&tab=safety`}
                className="flex min-w-0 items-center gap-2 hover:underline"
              >
                {row.fleetColor ? (
                  <span
                    className="size-2 shrink-0 rounded-full"
                    style={{ backgroundColor: row.fleetColor }}
                  />
                ) : null}
                <span className="truncate font-medium">{row.name}</span>
                {row.fleetCode ? (
                  <span className="text-muted-foreground shrink-0">{row.fleetCode}</span>
                ) : null}
              </Link>
              <span className="flex shrink-0 items-center gap-2">
                <Badge variant={safetyRatingTone(row.rating)}>
                  {safetyRatingLabel(row.rating)}
                </Badge>
                <span className="text-muted-foreground tabular-nums">
                  {row.score}
                  {row.activePoints > 0 ? ` · ${row.activePoints} pts` : ""}
                </span>
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Figure({
  label,
  value,
  detail,
  tone,
}: {
  label: string;
  value: string;
  detail?: string;
  tone?: "critical" | "warning";
}) {
  return (
    <div className="rounded-lg border px-3 py-2">
      <p className="text-muted-foreground text-[11px]">{label}</p>
      <p
        className={cn(
          "text-lg font-semibold tabular-nums",
          tone === "critical" && "text-red-600 dark:text-red-400",
          tone === "warning" && "text-amber-600 dark:text-amber-400",
        )}
      >
        {value}
      </p>
      {detail ? <p className="text-muted-foreground text-[11px]">{detail}</p> : null}
    </div>
  );
}
