import { ATTENTION_ROWS, type AttentionSummary } from "@/config/attention-rows";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { CheckIcon, LayoutGridIcon, LockIcon } from "lucide-react";
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
 * glanced at, and a seconds hand is motion without information.
 */
function useOrganizationClock(timezone: string) {
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

  return {
    date: now.toLocaleDateString(undefined, {
      weekday: "long",
      month: "long",
      day: "numeric",
      timeZone: timezone,
    }),
    time: now.toLocaleTimeString(undefined, {
      hour: "numeric",
      minute: "2-digit",
      timeZoneName: "short",
      timeZone: timezone,
    }),
    hour: Number(
      now.toLocaleString("en-US", { hour: "numeric", hour12: false, timeZone: timezone }),
    ),
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
): Chip[] {
  const chips: Chip[] = [
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
  ];

  for (const row of ATTENTION_ROWS) {
    const count = attention?.[row.key] ?? 0;
    if (count > 0) {
      chips.push({
        label: row.label.toLowerCase(),
        count,
        href: row.path,
        tone: row.tone === "destructive" ? "danger" : row.tone === "warning" ? "warning" : "brand",
      });
    }
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
  loading: boolean;
  canCustomize: boolean;
  locked: boolean;
  editing: boolean;
  onCustomize: () => void;
  timezone: string;
};

/**
 * The one fixed band on the home screen. It answers "what is today" in a
 * sentence built from the same counts the chips link to, so the headline can
 * never disagree with what it sits above.
 */
export function BriefingBar({
  attention,
  analytics,
  loading,
  canCustomize,
  locked,
  editing,
  onCustomize,
  timezone,
}: BriefingBarProps) {
  const user = useAuthStore((state) => state.user);
  const clock = useOrganizationClock(timezone);
  const chips = buildChips(attention, analytics);
  const total = chips.reduce((sum, chip) => sum + chip.count, 0);
  const firstName = user?.name?.split(" ")[0] ?? "there";

  return (
    <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border px-4 py-3">
      <div className="flex min-w-0 flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">
          {greeting(clock.hour)}, {firstName}
        </h1>
        <p className="text-2xs text-muted-foreground">
          {clock.date} · {clock.time}
        </p>

        {!loading && (
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1">
            {total === 0 ? (
              <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <CheckIcon className="size-3 text-success" />
                You&rsquo;re clear — {analytics.activeShipments.count} loads moving, nothing
                flagged.
              </span>
            ) : (
              <>
                <span className="text-xs font-medium">
                  {total} {total === 1 ? "item needs" : "items need"} you
                </span>
                {chips.map((chip) => (
                  <Link
                    key={`${chip.label}-${chip.href}`}
                    to={chip.href}
                    className="flex items-center gap-1.5 text-2xs text-muted-foreground transition-colors hover:text-foreground"
                  >
                    <span className={cn("size-1.5 rounded-full", CHIP_DOT[chip.tone])} />
                    <span className="font-table tabular-nums">{chip.count}</span>
                    <span>{chip.label}</span>
                  </Link>
                ))}
              </>
            )}
          </div>
        )}
      </div>

      <div className="flex shrink-0 items-center gap-2">
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
            disabled={!canCustomize}
          >
            <LayoutGridIcon className="size-3.5" />
            {editing ? "Done" : "Customize"}
          </Button>
        )}
      </div>
    </div>
  );
}
