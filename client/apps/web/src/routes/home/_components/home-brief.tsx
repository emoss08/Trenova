import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import type { Tone } from "@/components/kpi/tone";
import { ATTENTION_ROWS, type AttentionSummary } from "@/config/attention-rows";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { ShipmentAnalyticsData } from "@/lib/shipment-analytics";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { dateTimeFormatter, formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { partOfDay, resolveUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { TimeFormat } from "@trenova/shared/types/user";

const SHIPMENTS_HREF = "/shipment-management/shipments";

/** The clock moves by the minute; checking this often keeps it within a few seconds of true. */
const CLOCK_TICK_MS = 10_000;

/**
 * The day and the time in the viewer's profile timezone, time format and language, so it
 * can never disagree with the times the rest of the app shows. A home screen is glanced
 * at, so it reads to the minute.
 */
function useUserClock() {
  const user = useAuthStore((state) => state.user);
  const now = new Date(useNowSeconds(CLOCK_TICK_MS) * 1000);
  const timeZone = resolveUserTimezone(user?.timezone);

  return {
    date: dateTimeFormatter({ weekday: "long", month: "long", day: "numeric", timeZone }).format(
      now,
    ),
    time: dateTimeFormatter({
      hour: "numeric",
      minute: "2-digit",
      hour12: user?.timeFormat === TimeFormat.enum["12-hour"],
      timeZoneName: "short",
      timeZone,
    }).format(now),
    hour: Number(
      dateTimeFormatter({ hour: "numeric", hourCycle: "h23", timeZone }, "en-US").format(now),
    ),
  };
}

export type BriefQueue = {
  key: string;
  /** What is waiting, as the queue's page names it. */
  label: string;
  count: number;
  href: string;
  tone: Tone;
};

/**
 * The queues with work in them, in the order a day is worked: freight that is not
 * covered or is slipping first, then what billing, EDI and the desk are holding. A
 * queue with nothing in it is left out, so the strip is exactly what needs someone.
 */
export function briefQueues(
  t: (message: string) => string,
  attention: AttentionSummary | undefined,
  analytics: ShipmentAnalyticsData,
  analyticsReady: boolean,
): BriefQueue[] {
  const queues: BriefQueue[] = [];

  if (analyticsReady) {
    queues.push(
      {
        key: "unassigned",
        label: t("Unassigned"),
        count: analytics.unassigned.count,
        href: SHIPMENTS_HREF,
        tone: "warning",
      },
      {
        key: "atRisk",
        label: t("At risk"),
        count: analytics.atRisk.count,
        href: SHIPMENTS_HREF,
        tone: "danger",
      },
    );
  }

  for (const row of ATTENTION_ROWS) {
    queues.push({
      key: row.key,
      label: t(row.label),
      count: attention?.[row.key] ?? 0,
      href: row.path,
      tone: row.tone === "destructive" ? "danger" : row.tone === "warning" ? "warning" : "info",
    });
  }

  return queues.filter((queue) => queue.count > 0);
}

export type HomeBriefProps = {
  attention: AttentionSummary | undefined;
  analytics: ShipmentAnalyticsData;
  /** True once the shipment payload holds real numbers rather than defaults. */
  analyticsReady: boolean;
  loading: boolean;
};

/**
 * What the home screen opens on, above anything a person arranged: the day, a
 * greeting, and what needs them. The sentence and the strip are built from the same
 * counts, so the headline can never disagree with the queues under it, and every
 * figure in the strip is the way into its queue.
 */
export function HomeBrief({ attention, analytics, analyticsReady, loading }: HomeBriefProps) {
  const t = useT();

  const user = useAuthStore((state) => state.user);
  const clock = useUserClock();
  const queues = briefQueues((message) => t(message), attention, analytics, analyticsReady);
  const total = queues.reduce((sum, queue) => sum + queue.count, 0);
  const firstName = user?.name?.trim().split(/\s+/)[0];

  return (
    <section aria-label={t("Today")} className="flex flex-col gap-3">
      <div className="flex flex-col gap-1">
        <p className="text-xs text-muted-foreground tabular-nums">
          {clock.date} · {clock.time}
        </p>
        <h2 className="text-2xl font-semibold">
          <Greeting part={partOfDay(clock.hour)} name={firstName} />
        </h2>
        {loading ? (
          <Skeleton className="h-4.5 w-56" />
        ) : total === 0 ? (
          <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
            <CheckIcon aria-hidden className="size-3.5 text-success" />
            {analyticsReady
              ? t(
                  "You're clear — {0, plural, one {# load} other {# loads}} moving, nothing flagged.",
                  analytics.activeShipments.count,
                )
              : t("You're clear — nothing flagged.")}
          </p>
        ) : (
          <p className="text-sm text-muted-foreground">
            {t(
              "{0, plural, one {# item needs you} other {# items need you}} across {1, plural, one {# queue} other {# queues}}.",
              total,
              queues.length,
            )}
          </p>
        )}
      </div>

      {loading ? (
        <KpiStripSkeleton count={4} bare sub={false} minItemWidth="9rem" />
      ) : queues.length > 0 ? (
        <KpiStrip bare minItemWidth="9rem" aria-label={t("What needs you")}>
          {queues.map((queue) => (
            <KpiStripItem
              key={queue.key}
              label={queue.label}
              value={formatNumber(queue.count)}
              tone={queue.tone}
              to={queue.href}
            />
          ))}
        </KpiStrip>
      ) : null}
    </section>
  );
}

/** One whole sentence per part of the day, so a language can put the name where it belongs. */
function Greeting({
  part,
  name,
}: {
  part: ReturnType<typeof partOfDay>;
  name: string | undefined;
}) {
  const t = useT();

  if (!name) {
    return part === "morning"
      ? t("Good morning")
      : part === "afternoon"
        ? t("Good afternoon")
        : t("Good evening");
  }
  return part === "morning"
    ? t("Good morning, {0}", name)
    : part === "afternoon"
      ? t("Good afternoon, {0}", name)
      : t("Good evening, {0}", name);
}
