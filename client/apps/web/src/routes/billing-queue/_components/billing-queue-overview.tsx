import { useT } from "@trenova/shared/i18n/use-t";
import {
  KpiStrip,
  KpiStripItem,
  KPI_STRIP_CELL_CLASS,
  KPI_VALUE_CLASS,
} from "@/components/kpi/kpi-strip";
import { useTypewriter } from "@/hooks/use-typewriter";
import { billingQueue } from "@/lib/queries/billing-queue";
import { billingQueueNova } from "@/lib/queries/billing-queue-nova";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useNow } from "@trenova/shared/hooks/use-now";
import {
  formatDurationFromSeconds,
  formatSecondsAgo,
  formatUnixInUserTimezone,
} from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BillingQueueBrief,
  BillingQueueLane,
  BillingQueueNovaGeneration,
  BillingQueueStats,
} from "@trenova/shared/types/billing-queue";
import { BILLING_QUEUE_OPEN_LANES } from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import { billingQueueStatsQuery } from "../billing-queue-queries";
import { LANE_DOT, laneLabel } from "./billing-queue-lanes";

/** Briefs already typed out in this session; a brief types once, then reads still. */
const typedBriefs = new Set<number>();

/**
 * Realtime moves the generation along; this reads it again only in case a
 * push was missed while it runs.
 */
const GENERATION_BACKSTOP_MS = 30_000;

type Translate = ReturnType<typeof useT>;

/**
 * The top of the queue: Nova's brief of what is waiting, typed in the way Nova
 * speaks, over the queue's pulse — what is waiting by lane, how fast delivered
 * freight turns into invoices, and what was billed today. Clicking a lane on the
 * bar narrows the list to it.
 */
export function BillingQueueOverview({
  activeLanes,
  onLaneSelect,
}: {
  activeLanes: readonly BillingQueueLane[];
  onLaneSelect: (lane: BillingQueueLane) => void;
}) {
  const { data: assist } = useQuery(billingQueue.assist());
  const { data: brief } = useQuery({ ...billingQueue.brief(), enabled: assist?.brief === true });
  const { data: generation } = useQuery({
    ...billingQueueNova.generation(),
    enabled: assist?.review === true || assist?.brief === true,
    refetchInterval: (query) => generationRefetchMs(query.state.data),
  });
  const { data: stats } = useQuery(billingQueueStatsQuery());
  const hasBrief = assist?.brief === true && brief?.available === true && brief.text !== "";

  return (
    <div className="flex shrink-0 flex-col gap-3.5">
      {hasBrief && brief ? (
        <NovaBrief key={brief.generatedAt} brief={brief} generation={generation ?? null} />
      ) : generation?.phase === "Left" ? (
        <NovaLeft generation={generation} />
      ) : generation ? (
        <NovaGenerating generation={generation} />
      ) : null}
      <QueuePulse
        stats={stats}
        nova={assist?.review === true}
        activeLanes={activeLanes}
        onLaneSelect={onLaneSelect}
      />
    </div>
  );
}

/**
 * When to read the generation again: while one runs, in case a push was
 * missed; with items left, when opening the queue may try them again, which
 * reading it then does.
 */
function generationRefetchMs(generation: BillingQueueNovaGeneration | undefined): number | false {
  if (!generation) return false;
  if (generation.phase === "Left" && generation.retryAt) {
    return Math.max(generation.retryAt * 1000 - Date.now(), GENERATION_BACKSTOP_MS);
  }
  return GENERATION_BACKSTOP_MS;
}

/** When Nova tries what it left again, as a time of day. */
function retryTime(generation: NonNullable<BillingQueueNovaGeneration>): string {
  return formatUnixInUserTimezone(generation.retryAt, { hour: "numeric", minute: "2-digit" }, "");
}

/** Items the last generation could not review, where the brief will be. */
function NovaLeft({ generation }: { generation: NonNullable<BillingQueueNovaGeneration> }) {
  const t = useT();

  return (
    <section
      aria-label={t("Nova left items unreviewed")}
      data-testid="billing-queue-nova-left"
      className="flex w-full items-start gap-2.5"
    >
      <NovaBadge size="lg" className="mt-px" />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <p className="text-foreground text-base leading-5">
          {t("Nova couldn't review some items")}
        </p>
        <span className="text-foreground-muted text-xs">
          {t("It tries them again after {0}", retryTime(generation))}
        </span>
      </div>
    </section>
  );
}

/** What Nova is doing for the whole queue, in a few words. */
function generationStatus(generation: NonNullable<BillingQueueNovaGeneration>, t: Translate) {
  if (generation.phase === "Briefing") return t("Writing the brief");
  if (generation.total === 0) return t("Reading the queue");
  return t("Reviewing {0} of {1}", generation.done, generation.total);
}

/** Nova generating the queue's reviews and brief, where the brief will be. */
function NovaGenerating({ generation }: { generation: NonNullable<BillingQueueNovaGeneration> }) {
  const t = useT();
  const reviewing = generation.phase === "Reviewing" && generation.total > 0;

  return (
    <section
      aria-label={t("Nova is generating")}
      aria-busy
      data-testid="billing-queue-nova-generating"
      className="flex w-full items-start gap-2.5"
    >
      <NovaBadge size="lg" spin="always" className="mt-px" />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <p className="text-base leading-5" aria-live="polite">
          <TextShimmer as="span">
            {generation.phase === "Briefing"
              ? t("Nova is writing the brief")
              : generation.total === 0
                ? t("Nova is reading the queue")
                : t(
                    "{0, plural, one {Nova is reviewing # item} other {Nova is reviewing # items}}",
                    generation.total,
                  )}
          </TextShimmer>
        </p>
        <span className="text-foreground-muted text-xs">
          {reviewing
            ? t(
                "{0} · each review appears on its item as it finishes",
                generationStatus(generation, t),
              )
            : t("The brief appears here when it is written")}
        </span>
      </div>
    </section>
  );
}

function NovaBrief({
  brief,
  generation,
}: {
  brief: BillingQueueBrief;
  generation: BillingQueueNovaGeneration;
}) {
  const t = useT();
  const now = useNow("minute");
  const { revealed, thinking, typing } = useTypewriter(brief.text, {
    animate: !typedBriefs.has(brief.generatedAt),
    onDone: () => typedBriefs.add(brief.generatedAt),
  });

  return (
    <section aria-label={t("Nova's brief")} className="flex w-full items-start gap-2.5">
      <NovaBadge size="lg" spin={thinking || typing ? "always" : "none"} className="mt-px" />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <p className="text-foreground text-base leading-5" aria-live="polite">
          <span className="sr-only">{brief.text}</span>
          <span aria-hidden>
            {thinking ? (
              <TextShimmer as="span">{t("Typing")}</TextShimmer>
            ) : (
              brief.text.slice(0, revealed)
            )}
            {typing ? (
              <span className="bg-foreground ml-px inline-block h-3.5 w-px align-middle" />
            ) : null}
          </span>
        </p>
        <span className="text-foreground-muted flex flex-wrap items-center gap-x-1.5 text-xs">
          <span>{t("Nova · written {0}", formatSecondsAgo(now - brief.generatedAt))}</span>
          {brief.stale && !generation ? (
            <span data-testid="billing-queue-brief-stale" aria-live="polite">
              <TextShimmer as="span">{t("Updating for the queue as it is now")}</TextShimmer>
            </span>
          ) : null}
          {generation?.phase === "Left" ? (
            <span data-testid="billing-queue-nova-left">
              {t("Couldn't review some items · tries again after {0}", retryTime(generation))}
            </span>
          ) : generation ? (
            <span
              data-testid="billing-queue-nova-generating"
              aria-live="polite"
              className="inline-flex items-center gap-1"
            >
              <NovaBadge size="xs" spin="always" className="size-2.5" />
              <TextShimmer as="span">{generationStatus(generation, t)}</TextShimmer>
            </span>
          ) : null}
        </span>
      </div>
    </section>
  );
}

function QueuePulse({
  stats,
  nova,
  activeLanes,
  onLaneSelect,
}: {
  stats: BillingQueueStats | undefined;
  nova: boolean;
  activeLanes: readonly BillingQueueLane[];
  onLaneSelect: (lane: BillingQueueLane) => void;
}) {
  const t = useT();
  const now = useNow("minute");
  const pulse = stats?.pulse;

  const turn = pulse?.deliveryToInvoiceDays != null ? Number(pulse.deliveryToInvoiceDays) : null;
  const prior =
    pulse?.priorDeliveryToInvoiceDays != null ? Number(pulse.priorDeliveryToInvoiceDays) : null;
  const turnDelta = turn != null && prior != null ? Math.round((turn - prior) * 10) / 10 : null;
  const oldestDays =
    pulse?.oldestWaitingAt != null ? Math.floor((now - pulse.oldestWaitingAt) / 86400) : null;

  return (
    <KpiStrip bare aria-label={t("Queue pulse")} minItemWidth="9rem">
      <div className={cn(KPI_STRIP_CELL_CLASS, "col-span-2 flex flex-col gap-2")}>
        <div className="flex min-w-0 flex-col gap-1.5">
          <span className="text-foreground-subtle truncate text-xs font-medium">
            {t("Waiting to bill")}
          </span>
          <div className="flex min-w-0 items-baseline gap-2">
            <span className={cn("text-foreground", KPI_VALUE_CLASS)}>
              {formatCurrency(Number(stats?.openTotal ?? 0))}
            </span>
            <span className="text-foreground-muted truncate font-mono text-xs">
              {t("{0, plural, one {# shipment} other {# shipments}}", stats?.openCount ?? 0)}
            </span>
          </div>
        </div>
        <LaneBar stats={stats} activeLanes={activeLanes} onLaneSelect={onLaneSelect} />
      </div>
      <KpiStripItem
        label={t("Delivery to invoice")}
        value={turn != null ? t("{0} days", turn.toFixed(1)) : "—"}
        delta={turnDelta}
        deltaTone={turnDelta != null && turnDelta <= 0 ? "success" : "danger"}
        deltaLabel={t("vs last week")}
        sub={oldestDays != null && oldestDays > 0 ? t("oldest {0}d", oldestDays) : undefined}
      />
      <KpiStripItem
        label={t("Billed today")}
        value={pulse?.approvedTodayCount ?? 0}
        sub={
          <span className="font-mono">
            {formatCurrency(Number(pulse?.approvedTodayTotal ?? 0))}
          </span>
        }
      />
      <KpiStripItem
        label={t("Median review")}
        value={
          pulse?.medianReviewSecondsToday != null
            ? formatDurationFromSeconds(pulse.medianReviewSecondsToday)
            : "—"
        }
        sub={nova ? t("Nova reviews before you do") : t("Across today's reviews")}
      />
    </KpiStrip>
  );
}

function LaneBar({
  stats,
  activeLanes,
  onLaneSelect,
}: {
  stats: BillingQueueStats | undefined;
  activeLanes: readonly BillingQueueLane[];
  onLaneSelect: (lane: BillingQueueLane) => void;
}) {
  const t = useT();
  const counts = new Map(stats?.lanes.map((lane) => [lane.lane, lane]) ?? []);
  const lanes = BILLING_QUEUE_OPEN_LANES.filter((lane) => (counts.get(lane)?.count ?? 0) > 0).sort(
    (a, b) => Number(counts.get(b)?.total ?? 0) - Number(counts.get(a)?.total ?? 0),
  );

  if (lanes.length === 0) {
    return <div className="bg-muted h-2 rounded-sm" aria-hidden />;
  }

  return (
    <>
      <div className="flex h-2 gap-0.5 overflow-hidden rounded-sm">
        {lanes.map((lane) => {
          const stat = counts.get(lane);
          const dimmed = activeLanes.length > 0 && !activeLanes.includes(lane);
          return (
            <Tooltip key={lane}>
              <TooltipTrigger
                render={
                  <button
                    type="button"
                    aria-label={t("Filter to {0}", laneLabel(lane, t))}
                    aria-pressed={activeLanes.includes(lane)}
                    onClick={() => onLaneSelect(lane)}
                    className={cn(
                      LANE_DOT[lane],
                      "h-full transition-[filter,opacity] hover:brightness-90",
                      dimmed && "opacity-30",
                    )}
                    style={{ flex: `${Number(stat?.total ?? 0) || stat?.count || 1} 1 0` }}
                  />
                }
              />
              <TooltipContent>
                {t(
                  "{0} · {1} · {2}",
                  laneLabel(lane, t),
                  stat?.count ?? 0,
                  formatCurrency(Number(stat?.total ?? 0)),
                )}
              </TooltipContent>
            </Tooltip>
          );
        })}
      </div>
      <div className="flex flex-wrap gap-x-3.5 gap-y-0.5 text-xs">
        {lanes.map((lane) => (
          <span key={lane} className="text-foreground-muted inline-flex items-center gap-1.5">
            <span aria-hidden className={cn("size-1.5 rounded-full", LANE_DOT[lane])} />
            {laneLabel(lane, t)}
            <span className="text-foreground font-mono">
              {formatCurrency(Number(counts.get(lane)?.total ?? 0))}
            </span>
          </span>
        ))}
      </div>
    </>
  );
}
