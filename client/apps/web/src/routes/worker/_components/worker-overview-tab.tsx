import {
  fetchWorkerOverview,
  WORKER_OVERVIEW_KEY,
  type OverviewChecklist,
  type OverviewCredentials,
  type OverviewPTOBalance,
  type OverviewReview,
  type OverviewSafety,
  type OverviewTraining,
  type WorkerConcern,
  type WorkerOverview,
} from "@/lib/graphql/worker-overview";
import { useQuery } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Progress } from "@trenova/shared/components/ui/progress";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  concernSeverityMeta,
  groupConcernsBySeverity,
  workerStandingMeta,
} from "@trenova/shared/lib/worker-standing";
import {
  CalendarRangeIcon,
  CheckCircle2Icon,
  ChevronRightIcon,
  ClipboardCheckIcon,
  ClipboardListIcon,
  GraduationCapIcon,
  IdCardIcon,
  ShieldAlertIcon,
  type LucideIcon,
} from "lucide-react";

type WorkerOverviewTabProps = {
  workerId: string;
  /** Hands the reader to the tab that fixes what they just clicked. */
  onOpenTab: (tab: string) => void;
};

export default function WorkerOverviewTab({ workerId, onOpenTab }: WorkerOverviewTabProps) {
  const { data, isLoading } = useQuery({
    queryKey: [WORKER_OVERVIEW_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerOverview(workerId, { signal }),
    enabled: Boolean(workerId),
  });

  if (isLoading || !data) {
    return (
      <div className="flex items-center justify-center py-12">
        <ComponentLoader message="Loading..." />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <StandingHeader overview={data} />
      <ConcernList concerns={data.concerns} onOpenTab={onOpenTab} />
      <SectionCards overview={data} onOpenTab={onOpenTab} />
    </div>
  );
}

function StandingHeader({ overview }: { overview: WorkerOverview }) {
  const meta = workerStandingMeta(overview.standing);
  const criticals = overview.concerns.filter(
    (concern: WorkerConcern) => concern.severity === "Critical",
  ).length;
  const ringValue = meta.rank === 0 ? 100 : Math.max(12, 100 - meta.rank * 30);

  return (
    <div
      data-testid="worker-standing"
      className="bg-card border-border flex flex-wrap items-center gap-4 rounded-xl border p-4"
    >
      <RingGauge
        value={ringValue}
        size={64}
        strokeWidth={6}
        tone={meta.ringTone}
        aria-label="Standing"
      >
        {meta.ringTone === "success" ? (
          <CheckCircle2Icon className="size-5 text-green-600 dark:text-green-400" />
        ) : (
          <span className="text-sm font-semibold tabular-nums">{criticals}</span>
        )}
      </RingGauge>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-sm font-semibold">{meta.label}</h3>
          {overview.worker.fleetCode ? (
            <Badge variant="outline">{overview.worker.fleetCode.code}</Badge>
          ) : null}
          {overview.worker.profile?.isQualified ? <Badge variant="active">Qualified</Badge> : null}
        </div>
        <p className={cn("text-xs", meta.textClass)}>{meta.blurb}</p>
        <p className="text-muted-foreground text-xs">
          {describeTenure(overview.worker.profile?.hireDate ?? null, overview.asOf)}
        </p>
      </div>
    </div>
  );
}

function ConcernList({
  concerns,
  onOpenTab,
}: {
  concerns: readonly WorkerConcern[];
  onOpenTab: (tab: string) => void;
}) {
  if (concerns.length === 0) {
    return (
      <div
        data-testid="overview-concerns"
        className="border-border text-muted-foreground flex items-center gap-2 rounded-xl border border-dashed p-4 text-xs"
      >
        <CheckCircle2Icon className="size-4 text-green-600 dark:text-green-400" />
        <span>Nothing needs attention right now.</span>
      </div>
    );
  }

  return (
    <div data-testid="overview-concerns" className="flex flex-col gap-3">
      {groupConcernsBySeverity(concerns).map((group) => {
        const meta = concernSeverityMeta(group.severity);
        return (
          <div key={group.severity} className="flex flex-col gap-1.5">
            <p className={cn("text-[11px] font-semibold tracking-wide uppercase", meta.textClass)}>
              {meta.label}
            </p>
            {group.items.map((concern) => (
              <button
                key={concern.code}
                type="button"
                data-testid={`concern-${concern.code}`}
                onClick={() => onOpenTab(concern.tab)}
                className={cn(
                  "hover:bg-muted/40 flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors",
                  meta.borderClass,
                )}
              >
                <div className="min-w-0 flex-1">
                  <p className={cn("text-sm font-medium", meta.textClass)}>{concern.headline}</p>
                  <p className="text-muted-foreground text-xs">{concern.detail}</p>
                </div>
                <ChevronRightIcon className="text-muted-foreground size-4 shrink-0" />
              </button>
            ))}
          </div>
        );
      })}
    </div>
  );
}

function SectionCards({
  overview,
  onOpenTab,
}: {
  overview: WorkerOverview;
  onOpenTab: (tab: string) => void;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {overview.credentials ? (
        <CredentialsCard summary={overview.credentials} onOpen={() => onOpenTab("credentials")} />
      ) : null}
      {overview.training ? (
        <TrainingCard summary={overview.training} onOpen={() => onOpenTab("training")} />
      ) : null}
      {overview.safety ? (
        <SafetyCard card={overview.safety} onOpen={() => onOpenTab("safety")} />
      ) : null}
      {overview.checklist ? (
        <ChecklistCard checklist={overview.checklist} onOpen={() => onOpenTab("checklist")} />
      ) : null}
      {overview.pto ? <PTOCard balances={overview.pto} onOpen={() => onOpenTab("pto")} /> : null}
      {overview.lastReview || overview.openReview || overview.nextReviewAt ? (
        <ReviewsCard
          last={overview.lastReview ?? null}
          open={overview.openReview ?? null}
          nextReviewAt={overview.nextReviewAt ?? null}
          onOpen={() => onOpenTab("reviews")}
        />
      ) : null}
    </div>
  );
}

type CardTone = "success" | "warning" | "critical" | "muted";

function SectionCard({
  testId,
  title,
  icon: Icon,
  headline,
  tone,
  children,
  onOpen,
}: {
  testId: string;
  title: string;
  icon: LucideIcon;
  headline: string;
  tone: CardTone;
  children?: React.ReactNode;
  onOpen: () => void;
}) {
  return (
    <button
      type="button"
      data-testid={testId}
      onClick={onOpen}
      className="bg-card hover:border-primary/40 flex flex-col gap-2 rounded-xl border p-3 text-left transition-colors"
    >
      <div className="flex items-center gap-2">
        <Icon className="text-muted-foreground size-3.5" />
        <span className="text-muted-foreground text-xs font-medium">{title}</span>
      </div>
      <p
        className={cn(
          "text-sm font-semibold",
          tone === "success" && "text-green-600 dark:text-green-400",
          tone === "warning" && "text-amber-600 dark:text-amber-400",
          tone === "critical" && "text-red-600 dark:text-red-400",
        )}
      >
        {headline}
      </p>
      {children}
    </button>
  );
}

function CredentialsCard({
  summary,
  onOpen,
}: {
  summary: OverviewCredentials;
  onOpen: () => void;
}) {
  const problems = summary.expiredCount + summary.missingCount;
  const tone: CardTone =
    problems > 0 ? "critical" : summary.expiringCount > 0 ? "warning" : "success";

  return (
    <SectionCard
      testId="overview-card-credentials"
      title="Credentials"
      icon={IdCardIcon}
      headline={`${summary.validCount} of ${summary.requiredCount} valid`}
      tone={tone}
      onOpen={onOpen}
    >
      <p className="text-muted-foreground text-xs">
        {describeCounts([
          [summary.expiredCount, "expired"],
          [summary.missingCount, "missing"],
          [summary.expiringCount, "expiring soon"],
        ]) ?? "Everything on file is in date."}
      </p>
    </SectionCard>
  );
}

function TrainingCard({ summary, onOpen }: { summary: OverviewTraining; onOpen: () => void }) {
  const problems = summary.expiredCount + summary.missingCount + summary.overdueCount;
  const soon = summary.dueCount + summary.expiringCount;
  const tone: CardTone = problems > 0 ? "critical" : soon > 0 ? "warning" : "success";

  return (
    <SectionCard
      testId="overview-card-training"
      title="Training"
      icon={GraduationCapIcon}
      headline={`${summary.currentCount} of ${summary.requiredCount} current`}
      tone={tone}
      onOpen={onOpen}
    >
      <p className="text-muted-foreground text-xs">
        {describeCounts([
          [summary.overdueCount, "overdue"],
          [summary.expiredCount, "expired"],
          [summary.missingCount, "never assigned"],
          [soon, "due soon"],
        ]) ?? "Every required course is in date."}
      </p>
    </SectionCard>
  );
}

function SafetyCard({ card, onOpen }: { card: OverviewSafety; onOpen: () => void }) {
  const tone: CardTone =
    card.rating === "AtRisk" ? "critical" : card.rating === "Watch" ? "warning" : "success";

  return (
    <SectionCard
      testId="overview-card-safety"
      title="Safety"
      icon={ShieldAlertIcon}
      headline={`Score ${card.score}`}
      tone={tone}
      onOpen={onOpen}
    >
      <p className="text-muted-foreground text-xs">
        {describeCounts([
          [card.activePoints, "active points"],
          [card.preventableAccidents, "preventable"],
          [card.outOfServiceOrders, "out of service"],
          [card.activeDiscipline, "active actions"],
        ]) ?? "No points and nothing outstanding."}
      </p>
    </SectionCard>
  );
}

function ChecklistCard({
  checklist,
  onOpen,
}: {
  checklist: OverviewChecklist;
  onOpen: () => void;
}) {
  const tone: CardTone = checklist.progress.overdue > 0 ? "warning" : "muted";

  return (
    <SectionCard
      testId="overview-card-checklist"
      title="Checklist"
      icon={ClipboardListIcon}
      headline={`${checklist.name} — ${checklist.progress.percent}%`}
      tone={tone}
      onOpen={onOpen}
    >
      <Progress value={checklist.progress.percent} className="h-1.5" />
      <p className="text-muted-foreground text-xs">
        {checklist.progress.requiredDone} of {checklist.progress.requiredTotal} required items
        settled
        {checklist.progress.overdue > 0 ? `, ${checklist.progress.overdue} overdue` : ""}
      </p>
    </SectionCard>
  );
}

function PTOCard({
  balances,
  onOpen,
}: {
  balances: readonly OverviewPTOBalance[];
  onOpen: () => void;
}) {
  const tracked = balances.filter((balance) => balance.tracked);
  const headline =
    tracked.length > 0
      ? `${formatDays(tracked[0].availableDays)} ${tracked[0].ptoType.toLowerCase()} days`
      : "No tracked balances";

  return (
    <SectionCard
      testId="overview-card-pto"
      title="Time off"
      icon={CalendarRangeIcon}
      headline={headline}
      tone="muted"
      onOpen={onOpen}
    >
      <p className="text-muted-foreground text-xs">
        {tracked.length > 1
          ? tracked
              .slice(1)
              .map(
                (balance) =>
                  `${formatDays(balance.availableDays)} ${balance.ptoType.toLowerCase()}`,
              )
              .join(" · ")
          : "Available after pending requests."}
      </p>
    </SectionCard>
  );
}

function ReviewsCard({
  last,
  open,
  nextReviewAt,
  onOpen,
}: {
  last: OverviewReview | null;
  open: OverviewReview | null;
  nextReviewAt: number | null;
  onOpen: () => void;
}) {
  const headline = open
    ? `${open.title} in progress`
    : last?.overallScore
      ? `Scored ${formatScore(last.overallScore)}`
      : (last?.title ?? "No reviews yet");

  return (
    <SectionCard
      testId="overview-card-reviews"
      title="Reviews"
      icon={ClipboardCheckIcon}
      headline={headline}
      tone="muted"
      onOpen={onOpen}
    >
      <p className="text-muted-foreground text-xs">
        {last ? `Last: ${last.title}` : "Nothing closed yet."}
        {nextReviewAt ? ` · Next due ${formatUnixDate(nextReviewAt)}` : ""}
      </p>
    </SectionCard>
  );
}

/**
 * Joins the non-zero counts into one line, or returns null when there is
 * nothing to report so the caller can say something reassuring instead.
 */
function describeCounts(counts: readonly [number, string][]): string | null {
  const parts = counts.filter(([value]) => value > 0).map(([value, label]) => `${value} ${label}`);
  return parts.length > 0 ? parts.join(" · ") : null;
}

function formatDays(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed)
    ? parsed.toLocaleString(undefined, { maximumFractionDigits: 2 })
    : value;
}

function formatScore(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed)
    ? parsed.toLocaleString(undefined, { maximumFractionDigits: 2 })
    : value;
}

function describeTenure(hireDate: number | null, asOf: number): string {
  if (!hireDate || hireDate <= 0) return "No hire date on the record.";
  const years = (asOf - hireDate) / (365.25 * 24 * 60 * 60);
  if (years < 1) {
    const months = Math.max(1, Math.round(years * 12));
    return `Hired ${formatUnixDate(hireDate)}, ${months} month${months === 1 ? "" : "s"} in.`;
  }
  const rounded = Math.round(years * 10) / 10;
  return `Hired ${formatUnixDate(hireDate)}, ${rounded} year${rounded === 1 ? "" : "s"} in.`;
}
