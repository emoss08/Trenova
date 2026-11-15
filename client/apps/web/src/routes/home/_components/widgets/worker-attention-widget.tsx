import { useT } from "@trenova/shared/i18n/use-t";
import {
  fetchWorkerRosterAttention,
  WORKER_ROSTER_ATTENTION_KEY,
} from "@/lib/graphql/worker-overview";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { PlanFeature } from "@/lib/plan-feature";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
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
  count: number | null;
  tone: Tone;
};

/**
 * Where HR needs to look today. Every line links into the workers roster with
 * the matching saved view already applied, so the number and the list behind it
 * can never disagree.
 */
export function WorkerAttentionWidget({ widget }: WidgetProps) {
  const t = useT();

  // Each line belongs to an add-on: compliance and expiries to Safety & Compliance's driver
  // compliance, the safety watch list to its safety events, overdue training to the HR Suite.
  // A line whose add-on the plan leaves out is not drawn at all.
  const compliance = usePlanFeature(PlanFeature.WorkforceCompliance).available;
  const safety = usePlanFeature(PlanFeature.WorkforceSafety).available;
  const talent = usePlanFeature(PlanFeature.WorkforceTalent).available;

  const { data, isLoading } = useQuery({
    queryKey: [WORKER_ROSTER_ATTENTION_KEY],
    queryFn: ({ signal }) => fetchWorkerRosterAttention({ signal }),
  });

  if (isLoading) {
    return (
      <WidgetShell title={widget.title || t("Workforce attention")} href="/hr/workers">
        <div className="flex flex-col gap-1.5">
          {[0, 1, 2, 3].map((row) => (
            <Skeleton key={row} className="h-7 w-full" />
          ))}
        </div>
      </WidgetShell>
    );
  }

  const lines: Line[] = [
    ...(compliance
      ? [
          {
            view: "non-compliant",
            label: t("Non-compliant"),
            count: data?.nonCompliant ?? 0,
            tone: "critical",
          } satisfies Line,
        ]
      : []),
    ...(talent
      ? [
          {
            view: "training-overdue",
            label: t("Training overdue"),
            count: data?.trainingOverdue ?? null,
            tone: "critical",
          } satisfies Line,
        ]
      : []),
    ...(safety
      ? [
          {
            view: "at-risk",
            label: t("Safety at risk"),
            count: data?.atRisk ?? 0,
            tone: "critical",
          } satisfies Line,
        ]
      : []),
    ...(compliance
      ? [
          {
            view: "expiring-soon",
            label: t("Expiring in 30 days"),
            count: data?.expiringSoon ?? 0,
            tone: "warning",
          } satisfies Line,
        ]
      : []),
  ];

  const shown = lines.filter((line) => line.count !== null);
  const allClear = shown.every((line) => line.count === 0);
  const standings = data
    ? [
        data.reviewsAwaitingSignOff == null ? null : (
          <Standing
            key="reviews"
            label={t("Reviews to sign")}
            value={String(data.reviewsAwaitingSignOff)}
            muted={data.reviewsAwaitingSignOff === 0}
          />
        ),
        data.ptoLiabilityDays == null ? null : (
          <Standing
            key="pto"
            label={t("PTO liability")}
            value={t("{0} days", formatPtoDayTotal(data.ptoLiabilityDays))}
            muted={Number(data.ptoLiabilityDays) === 0}
          />
        ),
        data.leaveCertificationsOutstanding == null ? null : (
          <Standing
            key="leave"
            label={t("Certifications owed")}
            value={String(data.leaveCertificationsOutstanding)}
            muted={data.leaveCertificationsOutstanding === 0}
          />
        ),
      ].filter(Boolean)
    : [];

  return (
    <WidgetShell title={widget.title || t("Workforce attention")} href="/hr/workers">
      {shown.length === 0 ? null : allClear ? (
        <p className="text-muted-foreground px-1.5 py-2 text-xs">
          {t("Every active worker is compliant, in date and off the watch list.")}
        </p>
      ) : (
        <div className="flex flex-col gap-0.5">
          {shown.map((line) => (
            <Link
              key={line.view}
              to={rosterViewHref(line.view)}
              className="hover:bg-surface-hover flex items-center justify-between rounded-md px-1.5 py-1 text-xs transition-colors"
            >
              <span className={cn(line.count === 0 && "text-muted-foreground")}>
                {t(line.label)}
              </span>
              <span
                className={cn(
                  "font-semibold tabular-nums",
                  line.count === 0 && "text-muted-foreground",
                  line.count !== null &&
                    line.count > 0 &&
                    line.tone === "critical" &&
                    "text-danger-foreground",
                  line.count !== null &&
                    line.count > 0 &&
                    line.tone === "warning" &&
                    "text-warning-foreground",
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
          {standings.length > 0 ? (
            <dl
              className={cn(
                "grid grid-cols-3 gap-x-2",
                shown.length > 0 && "border-border/60 mt-2 border-t pt-2",
              )}
            >
              {standings}
            </dl>
          ) : null}
          <p className="text-muted-foreground mt-1.5 px-1.5 text-xs">
            {t("{0, plural, one {# active worker} other {# active workers}}", data.activeWorkers)}
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
 * filtered list. The server sends null for a figure the reader may not read or
 * whose add-on the plan leaves out, and the figure is left out.
 */
function Standing({ label, value, muted }: { label: string; value: string; muted: boolean }) {
  return (
    <div className="px-1.5">
      <dt className="text-muted-foreground text-xs">{label}</dt>
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
