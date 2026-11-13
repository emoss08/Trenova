import { formatUnixDayTimeRange, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import type { CSSProperties, ReactNode } from "react";
import { tourTarget } from "@/components/tour/tour-model";
import type { BalanceLevel, Window } from "./outlook-model";

/** The heat ramp, as detention intelligence draws one: a hue mixed into the page by level. */
const SHADE: Record<1 | 2 | 3, number> = { 1: 22, 2: 48, 3: 82 };

export function balanceStyle(balance: BalanceLevel): CSSProperties | undefined {
  if (balance.kind === "even" || balance.level === 0) {
    return undefined;
  }
  const hue = balance.kind === "short" ? "var(--color-warning)" : "var(--color-accent-sky)";
  return {
    background: `color-mix(in oklch, ${hue} ${SHADE[balance.level]}%, var(--color-background))`,
  };
}

export function balanceTextClass(balance: BalanceLevel): string {
  if (balance.kind === "even") {
    return "text-foreground-faint";
  }
  if (balance.level === 3) {
    return balance.kind === "short"
      ? "text-warning-on-solid"
      : "text-foreground-on-solid dark:text-foreground";
  }
  return balance.kind === "short" ? "text-warning-foreground" : "text-accent-sky-on-subtle";
}

/** "Thu 4 PM": a window by its weekday and start hour. */
export function slotLabel(start: number): string {
  return formatUnixInUserTimezone(start, { weekday: "short", hour: "numeric" });
}

/** "4 PM": a window by its start hour alone, beside a weekday already shown. */
export function hourLabel(start: number): string {
  return formatUnixInUserTimezone(start, { hour: "numeric" });
}

/** "Thu": a window's weekday. */
export function dayLabel(start: number): string {
  return formatUnixInUserTimezone(start, { weekday: "short" });
}

/** The calendar day a moment falls on in the reader's time zone, to tell where a day begins. */
export function dayKey(at: number): string {
  return formatUnixInUserTimezone(at, { dateStyle: "short" });
}

/**
 * "Thu 4:00 PM – 8:00 PM": a whole window, from its own start and end. A window is four hours
 * on the organization's clock, which is three or five real hours on the two nights the clocks
 * change, so the end is read, never added.
 */
export function slotRange(window: Window | undefined): string {
  return window ? formatUnixDayTimeRange(window.start, window.end) : "";
}

/**
 * One block of the brief panel, as the shipment board's side panel sets its sections: a quiet
 * label, the block's one figure on the right, and a hairline above every block but the first.
 */
export function PanelSection({
  id,
  title,
  figure,
  children,
}: {
  id: string;
  title: string;
  figure?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section
      id={id}
      aria-label={title}
      className="flex min-w-0 scroll-mt-2 flex-col gap-2.5 border-t border-border-subtle pt-3 first:border-t-0 first:pt-0"
      {...tourTarget(id)}
    >
      <header className="flex items-baseline justify-between gap-2">
        <span className="text-xs font-medium text-muted-foreground">{title}</span>
        {figure ? (
          <span className="font-mono text-sm font-medium tabular-nums">{figure}</span>
        ) : null}
      </header>
      {children}
    </section>
  );
}

export function Strong({ children }: { children: ReactNode }) {
  return <span className="font-medium text-foreground">{children}</span>;
}
