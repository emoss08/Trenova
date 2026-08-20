import { ATTENTION_ROWS, type AttentionSummary } from "@/config/attention-rows";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { TimeFormat } from "@trenova/shared/types/user";
import { ArrowRightIcon, CheckIcon, LayoutGridIcon, LockIcon } from "lucide-react";
import { m } from "motion/react";
import { useEffect, useState } from "react";
import { Link } from "react-router";
import type { ShipmentAnalyticsData } from "@/lib/shipment-analytics";

function greeting(hour: number): string {
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}

/**
 * The clock ticks on the minute rather than the second: a home screen is
 * glanced at, and a seconds hand is motion without information. It renders in
 * the viewer's profile timezone and time format so it can never disagree with
 * the clock the app header already shows.
 */
function useUserClock() {
  const user = useAuthStore((state) => state.user);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const msToNextMinute = 60_000 - (Date.now() % 60_000);
    let interval: ReturnType<typeof setInterval> | undefined;
    const timeout = setTimeout(() => {
      setNow(new Date());
      interval = setInterval(() => setNow(new Date()), 60_000);
    }, msToNextMinute);

    return () => {
      clearTimeout(timeout);
      if (interval) clearInterval(interval);
    };
  }, []);

  const timeZone = resolveUserTimezone(user?.timezone);

  return {
    date: now.toLocaleDateString("en-US", {
      weekday: "long",
      month: "long",
      day: "numeric",
      timeZone,
    }),
    time: now.toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: user?.timeFormat === TimeFormat.enum["12-hour"],
      timeZoneName: "short",
      timeZone,
    }),
    hour: Number(now.toLocaleString("en-US", { hour: "numeric", hourCycle: "h23", timeZone })),
  };
}

type Chip = {
  label: string;
  count: number;
  href: string;
  tone: "danger" | "warning" | "brand";
};

function buildChips(
  attention: AttentionSummary | undefined,
  analytics: ShipmentAnalyticsData,
  analyticsReady: boolean,
): Chip[] {
  const chips: Chip[] = [];

  if (analyticsReady) {
    chips.push(
      {
        label: "unassigned",
        count: analytics.unassigned.count,
        href: "/shipment-management/shipments",
        tone: "warning",
      },
      {
        label: "at risk",
        count: analytics.atRisk.count,
        href: "/shipment-management/shipments",
        tone: "danger",
      },
    );
  }

  for (const row of ATTENTION_ROWS) {
    chips.push({
      label: row.label.toLowerCase(),
      count: attention?.[row.key] ?? 0,
      href: row.path,
      tone: row.tone === "destructive" ? "danger" : row.tone === "warning" ? "warning" : "brand",
    });
  }

  return chips.filter((chip) => chip.count > 0);
}

const CHIP_DOT: Record<Chip["tone"], string> = {
  danger: "bg-destructive",
  warning: "bg-warning",
  brand: "bg-brand",
};

export type BriefingBarProps = {
  attention: AttentionSummary | undefined;
  analytics: ShipmentAnalyticsData;
  /** True once the shipment payload holds real numbers rather than defaults. */
  analyticsReady: boolean;
  loading: boolean;
  canCustomize: boolean;
  locked: boolean;
  editing: boolean;
  saving: boolean;
  onCustomize: () => void;
};

/**
 * The one fixed band on the home screen. It answers "what is today" in a
 * sentence built from the same counts the chips link to, so the headline can
 * never disagree with what it sits above.
 */
export function BriefingBar({
  attention,
  analytics,
  analyticsReady,
  loading,
  canCustomize,
  locked,
  editing,
  saving,
  onCustomize,
}: BriefingBarProps) {
  const user = useAuthStore((state) => state.user);
  const clock = useUserClock();
  const chips = buildChips(attention, analytics, analyticsReady);
  const total = chips.reduce((sum, chip) => sum + chip.count, 0);
  const firstName = user?.name?.split(" ")[0] ?? "there";

  return (
    <div className="border-border relative isolate overflow-hidden border-b">
      <div
        aria-hidden
        className="from-brand/[0.05] pointer-events-none absolute inset-0 bg-gradient-to-b via-transparent to-transparent"
      />

      <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3 px-5 py-4">
        <div className="flex min-w-0 flex-col gap-1.5">
          <h1 className="text-lg leading-none font-semibold tracking-tight">
            {greeting(clock.hour)}, {firstName}
          </h1>

          <div className="flex min-h-5.5 flex-wrap items-center gap-x-2 gap-y-1">
            {loading ? (
              <>
                <Skeleton className="h-5 w-28 rounded-full" />
                <Skeleton className="h-5 w-20 rounded-full" />
                <Skeleton className="h-5 w-24 rounded-full" />
              </>
            ) : total === 0 ? (
              <m.span
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25, ease: "easeOut" }}
                className="text-muted-foreground flex items-center gap-1.5 text-xs"
              >
                <span className="bg-success/15 flex size-4 items-center justify-center rounded-full">
                  <CheckIcon className="text-success size-2.5" />
                </span>
                {analyticsReady
                  ? `You're clear — ${analytics.activeShipments.count} ${
                      analytics.activeShipments.count === 1 ? "load" : "loads"
                    } moving, nothing flagged.`
                  : "You're clear — nothing flagged."}
              </m.span>
            ) : (
              <>
                <m.span
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.25, ease: "easeOut" }}
                  className="text-xs font-medium"
                >
                  {total} {total === 1 ? "item needs" : "items need"} you
                </m.span>
                {chips.map((chip, index) => (
                  <m.span
                    key={`${chip.label}-${chip.href}`}
                    initial={{ opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.25, delay: 0.04 * (index + 1), ease: "easeOut" }}
                  >
                    <Link
                      to={chip.href}
                      className="group border-border/70 bg-card text-2xs text-muted-foreground hover:border-border hover:text-foreground flex h-5.5 items-center gap-1.5 rounded-full border px-2 transition-colors"
                    >
                      <span className={cn("size-1.5 rounded-full", CHIP_DOT[chip.tone])} />
                      <span className="font-table text-foreground font-medium tabular-nums">
                        {chip.count}
                      </span>
                      <span>{chip.label}</span>
                      <ArrowRightIcon className="-ml-0.5 size-2.5 -translate-x-0.5 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
                    </Link>
                  </m.span>
                ))}
              </>
            )}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-3">
          <div className="flex flex-col items-end gap-1">
            <span className="flex items-center gap-1.5 font-mono text-sm leading-none font-medium tabular-nums">
              <span className="relative flex size-1.5">
                <span className="bg-success/60 absolute inline-flex size-full animate-ping rounded-full" />
                <span className="bg-success relative inline-flex size-1.5 rounded-full" />
              </span>
              {clock.time}
            </span>
            <span className="text-2xs text-muted-foreground leading-none">{clock.date}</span>
          </div>

          <div className="bg-border h-7 w-px" aria-hidden />

          {locked ? (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button variant="outline" size="sm" disabled>
                    <LockIcon className="size-3.5" />
                    Customize
                  </Button>
                }
              />
              <TooltipContent side="bottom">
                Your administrator manages this home screen.
              </TooltipContent>
            </Tooltip>
          ) : (
            <Button
              variant={editing ? "default" : "outline"}
              size="sm"
              onClick={onCustomize}
              disabled={!canCustomize || saving}
            >
              <LayoutGridIcon className="size-3.5" />
              {editing ? "Done" : "Customize"}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
