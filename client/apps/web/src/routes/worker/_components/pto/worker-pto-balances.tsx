import { ptoTypeChoices } from "@/lib/choices";
import type { WorkerPTOBalanceView } from "@/lib/graphql/pto-policy";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { formatPtoDayTotal } from "@trenova/shared/lib/pto";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState, type CSSProperties } from "react";

export function ptoTypeLabel(ptoType: string): string {
  return ptoTypeChoices.find((choice) => choice.value === ptoType)?.label ?? ptoType;
}

/** The most days a cap bar draws one tick per day for; a longer cap would be a solid comb. */
const MAX_DAY_TICKS = 40;

export type CapBar = {
  /** Days the full width of the bar stands for. */
  scale: number;
  /** Share of the bar, 0 to 100, for days free to take now. */
  availablePercent: number;
  /** Share of the bar for days held by requests awaiting a decision, hatched. */
  pendingPercent: number;
  /** The balance stands above the cap, so the bar is drawn longer than the cap. */
  overCap: boolean;
  /** One tick per day of the cap, or null when there is no cap or it is too long to tick. */
  ticks: number | null;
};

/**
 * A balance against its cap: what is free now and what pending requests hold, as shares of the
 * cap (or of the balance when the policy sets no cap). Neither part draws below zero.
 */
export function capBarParts({
  balanceDays,
  pendingDays,
  availableDays,
  maxBalanceDays,
}: Pick<
  WorkerPTOBalanceView,
  "balanceDays" | "pendingDays" | "availableDays" | "maxBalanceDays"
>): CapBar {
  const available = Math.max(0, Number(availableDays) || 0);
  const pending = Math.max(0, Number(pendingDays) || 0);
  const balance = Math.max(0, Number(balanceDays) || 0);
  const cap = maxBalanceDays ? Number(maxBalanceDays) : 0;
  const held = available + pending;
  const hasCap = Number.isFinite(cap) && cap > 0;
  const scale = hasCap ? Math.max(cap, held) : Math.max(held, balance, 1);
  return {
    scale,
    availablePercent: (available / scale) * 100,
    pendingPercent: (pending / scale) * 100,
    overCap: hasCap && held > cap,
    ticks: hasCap && cap <= MAX_DAY_TICKS ? Math.round(scale) : null,
  };
}

function oneDecimal(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed.toFixed(1) : value;
}

const TICKS_STYLE: CSSProperties = {
  backgroundImage:
    "repeating-linear-gradient(to right, transparent 0, transparent calc(var(--tick) - 1px), var(--color-card) calc(var(--tick) - 1px), var(--color-card) var(--tick))",
};

const PENDING_STYLE: CSSProperties = {
  backgroundImage:
    "repeating-linear-gradient(135deg, var(--color-ink) 0, var(--color-ink) 1px, transparent 1px, transparent 4px)",
};

/**
 * The tracked balances as one joined strip: each type's days free now, a cap bar with the part
 * pending requests hold hatched, and the figures the balance is built from.
 */
export function WorkerPTOBalances({ balances }: { balances: readonly WorkerPTOBalanceView[] }) {
  const t = useT();
  const tracked = balances.filter((balance) => balance.tracked);
  if (tracked.length === 0) {
    return (
      <p className="rounded-lg border bg-card px-4 py-3 text-xs text-muted-foreground">
        {t("The policy does not track any type of time off yet.")}
      </p>
    );
  }
  return (
    <section aria-label={t("Balances")} className="@container/balances rounded-lg border bg-card">
      <div className="grid grid-cols-1 @min-[44rem]/balances:auto-cols-fr @min-[44rem]/balances:grid-flow-col">
        {tracked.map((balance) => (
          <BalanceCell key={balance.ptoType} balance={balance} />
        ))}
      </div>
    </section>
  );
}

function BalanceCell({ balance }: { balance: WorkerPTOBalanceView }) {
  const t = useT();
  const [drawn, setDrawn] = useState(false);
  useEffect(() => {
    if (drawn) return;
    const frame = requestAnimationFrame(() => setDrawn(true));
    return () => cancelAnimationFrame(frame);
  }, [drawn]);
  const bar = capBarParts(balance);
  const available = Number(balance.availableDays);
  const name = t(ptoTypeLabel(balance.ptoType));
  const pending = Number(balance.pendingDays) || 0;
  const line = [
    t("{0} balance", oneDecimal(balance.balanceDays)),
    pending > 0 ? t("{0} pending", formatPtoDayTotal(balance.pendingDays)) : null,
    t("{0} accrued", formatPtoDayTotal(balance.accruedYtdDays)),
    t("{0} used this year", formatPtoDayTotal(balance.usedYtdDays)),
  ].filter((part): part is string => part !== null);

  return (
    <div
      data-testid={`pto-balance-${balance.ptoType}`}
      className="flex min-w-0 flex-col border-border-subtle px-4 pt-3 pb-3.5 not-first:border-t @min-[44rem]/balances:not-first:border-t-0 @min-[44rem]/balances:not-first:border-l"
    >
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs font-medium text-foreground-subtle">{name}</span>
        <span className="flex items-center gap-1.5 text-2xs text-foreground-subtle tabular-nums">
          {balance.enforced ? null : <Badge variant="neutral">{t("Not enforced")}</Badge>}
          {balance.maxBalanceDays
            ? t("cap {0}", formatPtoDayTotal(balance.maxBalanceDays))
            : t("no cap")}
        </span>
      </div>
      <div className="mt-1 mb-2.5 flex items-baseline gap-2">
        <b
          className={cn(
            "text-2xl font-semibold tabular-nums",
            available < 0 && "text-danger-foreground",
          )}
        >
          {oneDecimal(balance.availableDays)}
        </b>
        <span className="text-muted-foreground">{t("days available")}</span>
      </div>
      <div
        role="img"
        aria-label={
          balance.maxBalanceDays
            ? t(
                "{0} available of a {1} day cap, {2} pending.",
                oneDecimal(balance.availableDays),
                formatPtoDayTotal(balance.maxBalanceDays),
                formatPtoDayTotal(balance.pendingDays),
              )
            : t(
                "{0} available, {1} pending.",
                oneDecimal(balance.availableDays),
                formatPtoDayTotal(balance.pendingDays),
              )
        }
        className="relative flex h-2.5 overflow-hidden rounded-xs bg-surface-active"
        style={bar.ticks ? ({ "--tick": `calc(100% / ${bar.ticks})` } as CSSProperties) : undefined}
      >
        <span
          className="h-full origin-left bg-ink transition-[flex-basis,transform] duration-320 ease-settle motion-reduce:transition-none"
          style={{
            flex: `0 0 ${bar.availablePercent}%`,
            transform: `scaleX(${drawn ? 1 : 0})`,
          }}
        />
        {bar.pendingPercent > 0 ? (
          <span
            data-slot="cap-bar-pending"
            className="h-full bg-card ring-1 ring-ink transition-[flex-basis] duration-320 ease-settle ring-inset motion-reduce:transition-none"
            style={{ ...PENDING_STYLE, flex: `0 0 ${bar.pendingPercent}%` }}
          />
        ) : null}
        {bar.ticks ? (
          <span aria-hidden className="pointer-events-none absolute inset-0" style={TICKS_STYLE} />
        ) : null}
      </div>
      <p className="mt-2 text-xs text-muted-foreground tabular-nums">{line.join(" · ")}</p>
      {balance.nextAccrual ? (
        <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
          {t(
            "+{0} on {1}",
            formatPtoDayTotal(balance.nextAccrual.nominalDays),
            formatUnixMonthDay(balance.nextAccrual.effectiveAt),
          )}
        </p>
      ) : null}
    </div>
  );
}
