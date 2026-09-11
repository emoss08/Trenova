import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
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
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn, initials } from "@trenova/shared/lib/utils";
import {
  concernSeverityMeta,
  groupConcernsBySeverity,
  workerStandingMeta,
} from "@trenova/shared/lib/worker-standing";
import {
  CalendarRangeIcon,
  CheckIcon,
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

/**
 * The first thing anybody sees on a worker. It answers three questions in
 * order: who is this, what needs doing, and how does each part of the record
 * stand. Colour is spent on the standing and on anything that is wrong;
 * everything else is type and space.
 */
export default function WorkerOverviewTab({ workerId, onOpenTab }: WorkerOverviewTabProps) {
  const { data, isLoading } = useQuery({
    queryKey: [WORKER_OVERVIEW_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerOverview(workerId, { signal }),
    enabled: Boolean(workerId),
  });

  if (isLoading || !data) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-20 w-full rounded-lg" />
        <Skeleton className="h-16 w-full rounded-lg" />
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <Skeleton className="h-28 rounded-lg" />
          <Skeleton className="h-28 rounded-lg" />
          <Skeleton className="h-28 rounded-lg" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <Identity overview={data} />
      <Attention concerns={data.concerns} onOpenTab={onOpenTab} />
      <RecordSections overview={data} onOpenTab={onOpenTab} />
      <AtAGlance overview={data} />
    </div>
  );
}

function Identity({ overview }: { overview: WorkerOverview }) {
  const t = useT();

  const meta = workerStandingMeta(overview.standing);
  const { worker } = overview;
  const facts = [
    worker.type,
    worker.driverType,
    worker.fleetCode?.code,
    describeTenure(worker.profile?.hireDate ?? null, overview.asOf),
  ].filter(Boolean);

  return (
    <div
      data-testid="worker-standing"
      className="flex flex-wrap items-center justify-between gap-4 rounded-lg border p-4"
    >
      <div className="flex min-w-0 items-center gap-3">
        <Avatar className="size-11">
          <AvatarFallback className="text-sm font-medium">
            {initials(worker.firstName, worker.lastName)}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate text-base font-semibold tracking-tight">
              {worker.firstName} {worker.lastName}
            </h3>
            <Badge variant={meta.badgeVariant}>{t(meta.label)}</Badge>
            <InfoPopover title={t("Standing")}>
              <p>
                {t("Status moves only through employment events on the Timeline tab. Dispatch says whether the worker can be assigned today; a leave or a suspension holds them until it ends.")}
              </p>
              <p>
                {t("Compliance comes from the credential file and reads Non-compliant while a required credential is expired or missing. Qualified is set when the onboarding checklist closes with every required item settled.")}
              </p>
            </InfoPopover>
          </div>
          <p className="text-muted-foreground mt-0.5 text-xs">{facts.join(" · ")}</p>
          <p className="mt-1 text-xs">{meta.blurb}</p>
        </div>
      </div>
      <dl className="grid shrink-0 grid-cols-2 gap-x-6 gap-y-1 text-xs">
        <Fact label={t("Status")} value={worker.status} />
        <Fact label={t("Dispatch")} value={worker.canBeAssigned ? "Assignable" : "Held"} />
        <Fact label={t("Compliance")} value={worker.profile?.complianceStatus ?? "—"} />
        <Fact label={t("Qualified")} value={worker.profile?.isQualified ? "Yes" : "No"} />
      </dl>
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <dt className="text-2xs text-muted-foreground uppercase">{label}</dt>
      <dd className="font-medium">{value}</dd>
    </div>
  );
}

function Attention({
  concerns,
  onOpenTab,
}: {
  concerns: readonly WorkerConcern[];
  onOpenTab: (tab: string) => void;
}) {
  const t = useT();

  return (
    <section data-testid="overview-concerns" className="flex flex-col gap-2">
      <SectionHeading count={concerns.length}>{t("Needs attention")}</SectionHeading>
      {concerns.length === 0 ? (
        <div className="text-muted-foreground flex items-center gap-2 rounded-lg border border-dashed p-4 text-xs">
          <CheckIcon className="size-4" />
          <span>{t("Nothing needs attention right now.")}</span>
        </div>
      ) : (
        <div className="divide-border divide-y rounded-lg border">
          {groupConcernsBySeverity(concerns).map((group) => {
            const meta = concernSeverityMeta(group.severity);
            return (
              <div key={group.severity} className="divide-border divide-y">
                <p className="text-2xs text-muted-foreground bg-muted/40 px-3 py-1 font-medium uppercase">
                  {t(meta.label)}
                </p>
                {group.items.map((concern) => (
                  <button
                    key={concern.code}
                    type="button"
                    data-testid={`concern-${concern.code}`}
                    onClick={() => onOpenTab(concern.tab)}
                    className="hover:bg-muted/40 flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors"
                  >
                    <span
                      className={cn("size-1.5 shrink-0 rounded-full", meta.dotClass)}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-medium">{concern.headline}</span>
                      <span className="text-muted-foreground block text-xs">{concern.detail}</span>
                    </span>
                    <ChevronRightIcon className="text-muted-foreground size-4 shrink-0" />
                  </button>
                ))}
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}

function RecordSections({
  overview,
  onOpenTab,
}: {
  overview: WorkerOverview;
  onOpenTab: (tab: string) => void;
}) {
  const t = useT();

  const cards = [
    overview.credentials ? (
      <CredentialsCard
        key="credentials"
        summary={overview.credentials}
        onOpen={() => onOpenTab("credentials")}
      />
    ) : null,
    overview.training ? (
      <TrainingCard
        key="training"
        summary={overview.training}
        onOpen={() => onOpenTab("training")}
      />
    ) : null,
    overview.safety ? (
      <SafetyCard key="safety" card={overview.safety} onOpen={() => onOpenTab("safety")} />
    ) : null,
    overview.checklist ? (
      <ChecklistCard
        key="checklist"
        checklist={overview.checklist}
        onOpen={() => onOpenTab("checklist")}
      />
    ) : null,
    overview.pto ? (
      <PTOCard key="pto" balances={overview.pto} onOpen={() => onOpenTab("pto")} />
    ) : null,
    overview.lastReview || overview.openReview || overview.nextReviewAt ? (
      <ReviewsCard
        key="reviews"
        last={overview.lastReview ?? null}
        open={overview.openReview ?? null}
        nextReviewAt={overview.nextReviewAt ?? null}
        onOpen={() => onOpenTab("reviews")}
      />
    ) : null,
  ].filter(Boolean);

  if (cards.length === 0) return null;

  return (
    <section className="flex flex-col gap-2">
      <SectionHeading>{t("The record")}</SectionHeading>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">{cards}</div>
    </section>
  );
}

function SectionHeading({ children, count }: { children: string; count?: number }) {
  return (
    <div className="flex items-baseline justify-between">
      <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">{children}</h4>
      {count !== undefined && count > 0 ? (
        <span className="text-muted-foreground font-mono text-[11px] tabular-nums">{count}</span>
      ) : null}
    </div>
  );
}

type CardState = { variant: BadgeVariant; label: string } | null;

/**
 * One part of the record as a figure, a unit and a line of detail. The badge
 * appears only when something is wrong — a card that is fine says so by
 * saying nothing.
 */
function MetricCard({
  testId,
  title,
  icon: Icon,
  value,
  unit,
  detail,
  state,
  children,
  onOpen,
}: {
  testId: string;
  title: string;
  icon: LucideIcon;
  value: string;
  unit?: string;
  detail: string;
  state?: CardState;
  children?: React.ReactNode;
  onOpen: () => void;
}) {
  const t = useT();

  return (
    <button
      type="button"
      data-testid={testId}
      onClick={onOpen}
      className="border-border/80 hover:border-border hover:bg-muted/30 group flex flex-col gap-2 rounded-lg border p-3 text-left transition-colors"
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground text-[11px] font-semibold uppercase">{title}</span>
        <span className="bg-accent inline-flex size-6 shrink-0 items-center justify-center rounded-md">
          <Icon className="size-3.5" />
        </span>
      </div>
      <div className="flex items-baseline gap-1">
        <span className="text-2xl leading-none font-semibold tracking-tight tabular-nums">
          {value}
        </span>{" "}
        {unit ? <span className="text-muted-foreground text-xs">{unit}</span> : null}
      </div>
      {children}
      <div className="mt-auto flex items-center justify-between gap-2">
        <span className="text-muted-foreground truncate text-[11px]">{detail}</span>
        {state ? (
          <Badge variant={state.variant} className="shrink-0">
            {t(state.label)}
          </Badge>
        ) : null}
      </div>
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
  const t = useT();

  const problems = summary.expiredCount + summary.missingCount;
  const state: CardState =
    problems > 0
      ? { variant: "inactive", label: `${problems} lapsed` }
      : summary.expiringCount > 0
        ? { variant: "warning", label: `${summary.expiringCount} expiring` }
        : null;

  return (
    <MetricCard
      testId="overview-card-credentials"
      title={t("Credentials")}
      icon={IdCardIcon}
      value={`${summary.validCount} of ${summary.requiredCount}`}
      unit="valid"
      detail={
        describeCounts([
          [summary.expiredCount, "expired"],
          [summary.missingCount, "missing"],
          [summary.expiringCount, "expiring soon"],
        ]) ?? "Everything on file is in date."
      }
      state={state}
      onOpen={onOpen}
    />
  );
}

function TrainingCard({ summary, onOpen }: { summary: OverviewTraining; onOpen: () => void }) {
  const t = useT();

  const problems = summary.expiredCount + summary.missingCount + summary.overdueCount;
  const soon = summary.dueCount + summary.expiringCount;
  const state: CardState =
    problems > 0
      ? { variant: "inactive", label: `${problems} behind` }
      : soon > 0
        ? { variant: "warning", label: `${soon} due soon` }
        : null;

  return (
    <MetricCard
      testId="overview-card-training"
      title={t("Training")}
      icon={GraduationCapIcon}
      value={`${summary.currentCount} of ${summary.requiredCount}`}
      unit="current"
      detail={
        describeCounts([
          [summary.overdueCount, "overdue"],
          [summary.expiredCount, "expired"],
          [summary.missingCount, "never assigned"],
          [soon, "due soon"],
        ]) ?? "Every required course is in date."
      }
      state={state}
      onOpen={onOpen}
    />
  );
}

function SafetyCard({ card, onOpen }: { card: OverviewSafety; onOpen: () => void }) {
  const t = useT();

  const state: CardState =
    card.rating === "AtRisk"
      ? { variant: "inactive", label: "At risk" }
      : card.rating === "Watch"
        ? { variant: "warning", label: "Watch" }
        : null;

  return (
    <MetricCard
      testId="overview-card-safety"
      title={t("Safety")}
      icon={ShieldAlertIcon}
      value={String(card.score)}
      unit="score"
      detail={
        describeCounts([
          [card.activePoints, "active points"],
          [card.preventableAccidents, "preventable"],
          [card.outOfServiceOrders, "out of service"],
          [card.activeDiscipline, "active actions"],
        ]) ?? "No points and nothing outstanding."
      }
      state={state}
      onOpen={onOpen}
    />
  );
}

function ChecklistCard({
  checklist,
  onOpen,
}: {
  checklist: OverviewChecklist;
  onOpen: () => void;
}) {
  const t = useT();

  const state: CardState =
    checklist.progress.overdue > 0
      ? { variant: "warning", label: `${checklist.progress.overdue} overdue` }
      : null;

  return (
    <MetricCard
      testId="overview-card-checklist"
      title={t("Checklist")}
      icon={ClipboardListIcon}
      value={`${checklist.progress.percent}%`}
      unit={checklist.name}
      detail={`${checklist.progress.requiredDone} of ${checklist.progress.requiredTotal} required items settled`}
      state={state}
      onOpen={onOpen}
    >
      <Progress value={checklist.progress.percent} className="h-1" />
    </MetricCard>
  );
}

function PTOCard({
  balances,
  onOpen,
}: {
  balances: readonly OverviewPTOBalance[];
  onOpen: () => void;
}) {
  const t = useT();

  const tracked = balances.filter((balance) => balance.tracked);
  const lead = tracked[0];

  return (
    <MetricCard
      testId="overview-card-pto"
      title={t("Time off")}
      icon={CalendarRangeIcon}
      value={lead ? formatDays(lead.availableDays) : "—"}
      unit={lead ? `${lead.ptoType.toLowerCase()} days` : undefined}
      detail={
        tracked.length > 1
          ? tracked
              .slice(1)
              .map(
                (balance) =>
                  `${formatDays(balance.availableDays)} ${balance.ptoType.toLowerCase()}`,
              )
              .join(" · ")
          : lead
            ? "Available after pending requests."
            : "No tracked balances."
      }
      onOpen={onOpen}
    />
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
  const t = useT();

  const value = open ? "Open" : last?.overallScore ? formatScore(last.overallScore) : "—";
  const unit = open ? open.title : last?.overallScore ? "last score" : undefined;
  const detail = [
    last ? `Last: ${last.title}` : "Nothing closed yet.",
    nextReviewAt ? `Next due ${formatUnixDate(nextReviewAt)}` : null,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <MetricCard
      testId="overview-card-reviews"
      title={t("Reviews")}
      icon={ClipboardCheckIcon}
      value={value}
      unit={unit}
      detail={detail}
      state={open ? { variant: "info", label: "In progress" } : null}
      onOpen={onOpen}
    />
  );
}

function AtAGlance({ overview }: { overview: WorkerOverview }) {
  const t = useT();

  const { worker } = overview;
  const profile = worker.profile;
  const rows: [string, string][] = [
    ["Worker type", worker.type],
    ["Driver type", worker.driverType],
    ["Fleet", worker.fleetCode?.code ?? "—"],
    ["Hired", profile?.hireDate ? formatUnixDate(profile.hireDate) : "—"],
    ["Terminated", profile?.terminationDate ? formatUnixDate(profile.terminationDate) : "—"],
    ["As of", formatUnixDate(overview.asOf)],
  ];

  return (
    <section className="flex flex-col gap-2">
      <SectionHeading>{t("At a glance")}</SectionHeading>
      <dl className="grid grid-cols-2 gap-x-6 gap-y-2 rounded-lg border p-3 text-xs sm:grid-cols-3">
        {rows.map(([label, value]) => (
          <div key={label} className="flex flex-col">
            <dt className="text-2xs text-muted-foreground uppercase">{label}</dt>
            <dd className="font-medium">{value}</dd>
          </div>
        ))}
      </dl>
    </section>
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

function describeTenure(hireDate: number | null, asOf: number): string | null {
  if (!hireDate || hireDate <= 0) return null;
  const years = (asOf - hireDate) / (365.25 * 24 * 60 * 60);
  if (years < 1) {
    const months = Math.max(1, Math.round(years * 12));
    return `${months} month${months === 1 ? "" : "s"} in`;
  }
  const rounded = Math.round(years * 10) / 10;
  return `${rounded} year${rounded === 1 ? "" : "s"} in`;
}
