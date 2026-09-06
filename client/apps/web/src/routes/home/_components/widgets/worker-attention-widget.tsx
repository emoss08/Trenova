import {
  fetchWorkerRosterAttention,
  WORKER_ROSTER_ATTENTION_KEY,
} from "@/lib/graphql/worker-overview";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { rosterViews, type RosterViewId } from "@trenova/shared/lib/worker-roster-views";
import { getTodayDate } from "@trenova/shared/lib/date";
import { formatPtoDayTotal } from "@trenova/shared/lib/pto";
import { Link } from "react-router";
import type { WidgetProps } from "../widget-registry";
import { WidgetShell } from "../widget-shell";

type Tone = "critical" | "warning" | "muted";

type Line = {
  view: RosterViewId;
  label: string;
  count: number;
  tone: Tone;
};

/**
 * Where HR needs to look today. Every line links into the workers roster with
 * the matching saved view already applied, so the number and the list behind it
 * can never disagree.
 */
export function WorkerAttentionWidget({ widget }: WidgetProps) {
  const { data, isLoading } = useQuery({
    queryKey: [WORKER_ROSTER_ATTENTION_KEY],
    queryFn: ({ signal }) => fetchWorkerRosterAttention({ signal }),
  });

  if (isLoading) {
    return (
      <WidgetShell title={widget.title || "Workforce Attention"} href="/hr/workers">
        <div className="flex flex-col gap-1.5">
          {[0, 1, 2, 3].map((row) => (
            <Skeleton key={row} className="h-7 w-full" />
          ))}
        </div>
      </WidgetShell>
    );
  }

  const lines: Line[] = [
    {
      view: "non-compliant",
      label: "Non-compliant",
      count: data?.nonCompliant ?? 0,
      tone: "critical",
    },
    {
      view: "training-overdue",
      label: "Training overdue",
      count: data?.trainingOverdue ?? 0,
      tone: "critical",
    },
    { view: "at-risk", label: "Safety at risk", count: data?.atRisk ?? 0, tone: "critical" },
    {
      view: "expiring-soon",
      label: "Expiring in 30 days",
      count: data?.expiringSoon ?? 0,
      tone: "warning",
    },
  ];

  const allClear = lines.every((line) => line.count === 0);

  return (
    <WidgetShell title={widget.title || "Workforce Attention"} href="/hr/workers">
      {allClear ? (
        <p className="text-muted-foreground px-1.5 py-2 text-xs">
          Every active worker is compliant, in date and off the watch list.
        </p>
      ) : (
        <div className="flex flex-col gap-0.5">
          {lines.map((line) => (
            <Link
              key={line.view}
              to={rosterViewHref(line.view)}
              className="hover:bg-muted/60 flex items-center justify-between rounded px-1.5 py-1 text-xs transition-colors"
            >
              <span className={cn(line.count === 0 && "text-muted-foreground")}>{line.label}</span>
              <span
                className={cn(
                  "font-semibold tabular-nums",
                  line.count === 0 && "text-muted-foreground",
                  line.count > 0 && line.tone === "critical" && "text-red-600 dark:text-red-400",
                  line.count > 0 && line.tone === "warning" && "text-amber-600 dark:text-amber-400",
                )}
              >
                {line.count}
              </span>
            </Link>
          ))}
        </div>
      )}
      {data ? (
        <>
          <dl className="border-border/60 mt-2 grid grid-cols-3 gap-x-2 border-t pt-2">
            <Standing
              label="Reviews to sign"
              value={String(data.reviewsAwaitingSignOff)}
              muted={data.reviewsAwaitingSignOff === 0}
            />
            <Standing
              label="PTO liability"
              value={`${formatPtoDayTotal(data.ptoLiabilityDays)} days`}
              muted={Number(data.ptoLiabilityDays) === 0}
            />
            <Standing
              label="Certifications owed"
              value={String(data.leaveCertificationsOutstanding)}
              muted={data.leaveCertificationsOutstanding === 0}
            />
          </dl>
          <p className="text-muted-foreground mt-1.5 px-1.5 text-[11px]">
            {data.activeWorkers} active {data.activeWorkers === 1 ? "worker" : "workers"}
          </p>
        </>
      ) : null}
    </WidgetShell>
  );
}

/**
 * A figure with no list behind it: none of reviews, the PTO liability or the
 * leave certifications owed has a roster view that would show the same number,
 * so these read as standing totals rather than as links that promise a
 * filtered list.
 */
function Standing({ label, value, muted }: { label: string; value: string; muted: boolean }) {
  return (
    <div className="px-1.5">
      <dt className="text-muted-foreground text-[11px]">{label}</dt>
      <dd
        className={cn(
          "text-sm font-semibold tabular-nums",
          muted && "text-muted-foreground font-normal",
        )}
      >
        {value}
      </dd>
    </div>
  );
}

/**
 * Builds the roster link for a view by serialising the same filters the chips
 * apply, so arriving from here looks exactly like clicking the chip.
 */
export function rosterViewHref(id: RosterViewId): string {
  const view = rosterViews(getTodayDate()).find((candidate) => candidate.id === id);
  if (!view || view.filters.length === 0) return "/hr/workers";
  const params = new URLSearchParams({ fieldFilters: JSON.stringify(view.filters) });
  return `/hr/workers?${params.toString()}`;
}
