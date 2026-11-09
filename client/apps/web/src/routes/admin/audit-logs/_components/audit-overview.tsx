import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { NarrativeSegments } from "@/components/narrative-segments";
import type { AuditActorStat, AuditOverview, AuditReview } from "@/lib/graphql/audit-overview";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatNumber, formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { cn } from "@trenova/shared/lib/utils";
import { AuditActivityChart } from "./audit-activity-chart";
import { AuditOverviewSkeleton } from "./audit-overview-skeleton";
import { percentChange, shareOf } from "./audit-overview-model";
import {
  AUDIT_WINDOWS,
  auditWindowOf,
  type AuditQuickView,
  type AuditWindow,
} from "./audit-page-state";
import { AuditReviewCard } from "./audit-review-card";
import { AuditTopActors } from "./audit-top-actors";
import { useAuditOverview } from "./use-audit-overview";

export type AuditOverviewProps = {
  days: AuditWindow;
  onDaysChange: (days: AuditWindow) => void;
  range: { from: number; to: number } | null;
  onRangeChange: (range: { from: number; to: number }) => void;
  selectedActorId: string | null;
  onSelectActor: (actor: AuditActorStat | null) => void;
  onQuickView: (view: AuditQuickView) => void;
  onOpenReview: (review: AuditReview) => void;
};

function UpdatedAgo({ at }: { at: number }) {
  const t = useT();
  const now = useNow("minute");
  return (
    <span className="text-foreground-subtle text-xs">
      {t("Updated {0}", formatRelativeTime(Math.min(Math.round(at / 1000) - now, 0)))}
    </span>
  );
}

function Figures({
  overview,
  onQuickView,
}: {
  overview: AuditOverview;
  onQuickView: (view: AuditQuickView) => void;
}) {
  const t = useT();
  const { totals, actors, days } = overview;
  const agents = totals.agentsAsked + totals.agentsOwn;
  const delta = percentChange(totals.entries, overview.previousEntries);
  const offHours = shareOf(totals.peopleOffHours, totals.people);
  const percent = (share: number) => `${Math.round(share * 100)}%`;

  return (
    <KpiStrip bare aria-label={t("Audit figures")}>
      <KpiStripItem
        size="lg"
        label={t("Changes")}
        value={formatNumber(totals.entries)}
        delta={delta}
        deltaLabel="%"
        deltaTone="muted"
        sub={t(
          "{0} in the {1, plural, one {day} other {# days}} before",
          formatNumber(overview.previousEntries),
          days,
        )}
      />
      <KpiStripItem
        size="lg"
        label={t("Who acted")}
        value={formatNumber(actors.people + actors.agents + actors.integrations + actors.jobs)}
        sub={t(
          "{0, plural, one {# person} other {# people}} · {1, plural, one {# agent} other {# agents}} · {2, plural, one {# key} other {# keys}}",
          actors.people,
          actors.agents,
          actors.integrations,
        )}
      />
      <KpiStripItem
        size="lg"
        label={t("Sensitive actions")}
        tone={totals.sensitive > 0 ? "warning" : undefined}
        value={formatNumber(totals.sensitive)}
        sub={
          overview.reviewsIncluded
            ? t("Roles, keys, deletes and exports")
            : t("Deletes and exports; security entries need security access")
        }
        onClick={() => onQuickView(overview.reviewsIncluded ? "security" : "deletes")}
      />
      <KpiStripItem
        size="lg"
        label={t("Made by agents")}
        value={percent(shareOf(agents, totals.entries))}
        sub={t(
          "{0, plural, one {# change} other {# changes}} · {1} asked for in a conversation",
          agents,
          formatNumber(totals.agentsAsked),
        )}
        onClick={() => onQuickView("agents")}
      />
      <KpiStripItem
        size="lg"
        label={t("People working off hours")}
        value={percent(offHours)}
        sub={t(
          "{0, plural, one {# change} other {# changes}} between 8 PM and 6 AM",
          totals.peopleOffHours,
        )}
      />
    </KpiStrip>
  );
}

/**
 * What happened in the audit log over the window, from one read: Nova's sentence, five figures,
 * when changes happened and by whom, and what the alert rules want a person to look at.
 */
export default function AuditOverviewBand({
  days,
  onDaysChange,
  range,
  onRangeChange,
  selectedActorId,
  onSelectActor,
  onQuickView,
  onOpenReview,
}: AuditOverviewProps) {
  const t = useT();
  const query = useAuditOverview(days);
  const overview = query.data;

  const windowItems: SegmentedControlItem<string>[] = AUDIT_WINDOWS.map((value) => ({
    value: String(value),
    label: value === 1 ? t("Today") : t("{0, plural, other {# days}}", value),
  }));

  if (query.isError && !overview) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription>
          {describeError(query.error).detail ?? t("The audit overview could not be loaded.")}
        </AlertDescription>
        <AlertAction>
          <Button
            type="button"
            size="xs"
            variant="outline"
            isLoading={query.isFetching}
            onClick={() => void query.refetch()}
          >
            {t("Try again")}
          </Button>
        </AlertAction>
      </Alert>
    );
  }

  if (!overview) {
    return <AuditOverviewSkeleton />;
  }

  const openReview = (id: string) => {
    const review = overview.openReviews.find((candidate) => candidate.id === id);
    if (review) onOpenReview(review);
  };

  return (
    <section
      aria-label={t("Overview")}
      className={cn(
        "flex flex-col gap-4 transition-opacity",
        query.isPlaceholderData && "opacity-70",
      )}
    >
      <div className="grid items-start gap-x-8 gap-y-3 md:grid-cols-[minmax(0,1fr)_auto]">
        <div className="flex min-w-0 flex-col gap-1.5">
          <span className="text-muted-foreground flex items-center gap-2 text-xs">
            <NovaBadge size="sm" />
            <span className="text-foreground font-medium">{t("Nova")}</span>
            {days === 1
              ? t("Today in the audit log")
              : t("The last {0} days in the audit log", days)}
          </span>
          <p className="text-muted-foreground max-w-[112ch] text-lg leading-relaxed text-pretty">
            <NarrativeSegments
              segments={overview.summary}
              onTarget={(segment, id) => {
                if (segment.target === "Review") openReview(id);
                else onQuickView(id as AuditQuickView);
              }}
            />
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3 md:flex-col md:items-end md:gap-2">
          <SegmentedControl
            aria-label={t("Window")}
            items={windowItems}
            value={String(days)}
            onValueChange={(value) => onDaysChange(auditWindowOf(Number(value)))}
          />
          <UpdatedAgo at={query.dataUpdatedAt} />
        </div>
      </div>

      <Figures overview={overview} onQuickView={onQuickView} />

      <div
        className={cn(
          "grid gap-3",
          overview.reviewsIncluded
            ? "lg:grid-cols-[minmax(0,1fr)_18rem] xl:grid-cols-[minmax(0,1fr)_18rem_23rem]"
            : "lg:grid-cols-[minmax(0,1fr)_18rem]",
        )}
      >
        <section
          aria-label={t("When changes happened")}
          className="bg-card flex min-w-0 flex-col gap-2 rounded-md border px-3 pt-2.5 pb-2"
        >
          <header>
            <h3 className="text-sm font-semibold">{t("When changes happened")}</h3>
            <p className="text-muted-foreground text-xs">
              {overview.unit === "Hour"
                ? t("Entries per hour. Shaded hours are between 8 PM and 6 AM.")
                : t("Entries per day.")}
            </p>
          </header>
          <AuditActivityChart
            activity={overview.activity}
            unit={overview.unit}
            timezone={overview.timezone}
            reviews={overview.openReviews}
            range={range}
            onRangeChange={onRangeChange}
            onOpenReview={onOpenReview}
          />
        </section>
        <AuditTopActors
          actors={overview.topActors}
          otherActors={overview.otherActors}
          otherEntries={overview.otherEntries}
          selectedId={selectedActorId}
          onSelect={onSelectActor}
        />
        {overview.reviewsIncluded ? (
          <AuditReviewCard
            reviews={overview.openReviews}
            openCount={overview.openReviewCount}
            onOpenReview={onOpenReview}
          />
        ) : null}
      </div>
    </section>
  );
}
