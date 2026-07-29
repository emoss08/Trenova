import {
  Meter,
  ShareBreakdown,
  type ShareSegment,
} from "@/components/detention/detention-charts";
import { detentionWaiverReasonChoices, findChoice } from "@/lib/choices";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { WaiverLeakageStat } from "@trenova/shared/types/detention";
import { HandCoinsIcon } from "lucide-react";
import { m } from "motion/react";
import { useMemo } from "react";
import { Panel, PanelEmpty, PanelError, PanelRowsSkeleton } from "./intelligence-panel";

/**
 * A fixed ramp rather than a per-reason mapping: the reasons that matter are
 * whichever ones cost the most, and the ramp keeps the biggest slice the
 * loudest no matter which reason it turns out to be.
 */
const LEAKAGE_TONES = [
  "bg-amber-500 dark:bg-amber-400",
  "bg-orange-500 dark:bg-orange-400",
  "bg-rose-500 dark:bg-rose-400",
  "bg-violet-500 dark:bg-violet-400",
  "bg-sky-500 dark:bg-sky-400",
  "bg-teal-500 dark:bg-teal-400",
  "bg-lime-500 dark:bg-lime-400",
  "bg-slate-400 dark:bg-slate-500",
];

function reasonLabel(reason: string): string {
  return findChoice(detentionWaiverReasonChoices, reason)?.label ?? reason;
}

function reasonDescription(reason: string): string | undefined {
  return findChoice(detentionWaiverReasonChoices, reason)?.description;
}

function LeakageRow({
  row,
  total,
  index,
}: {
  row: WaiverLeakageStat;
  total: number;
  index: number;
}) {
  const share = total > 0 ? row.waivedAmount / total : 0;
  const perWaiver = row.waiverCount > 0 ? row.waivedAmount / row.waiverCount : 0;
  const description = reasonDescription(row.reason);

  return (
    <m.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.26, delay: Math.min(index, 10) * 0.04, ease: "easeOut" }}
      className="px-3 py-2.5 transition-colors hover:bg-muted/40"
    >
      <div className="flex items-center gap-3">
        <span
          className={cn(
            "size-2 shrink-0 rounded-full",
            LEAKAGE_TONES[index % LEAKAGE_TONES.length],
          )}
        />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm leading-tight font-medium">
            {reasonLabel(row.reason)}
          </p>
          {description ? (
            <p className="mt-0.5 truncate text-2xs text-muted-foreground">
              {description}
            </p>
          ) : null}
        </div>
        <div className="w-24 shrink-0 text-right">
          <p className="text-sm leading-tight font-medium tabular-nums">
            {formatCurrency(row.waivedAmount)}
          </p>
          <p className="mt-0.5 text-2xs text-muted-foreground tabular-nums">
            {Math.round(share * 100)}% of leakage
          </p>
        </div>
      </div>

      <div className="mt-2 flex items-center gap-3 pl-5">
        <Meter
          value={share}
          className="min-w-0 flex-1"
          barClassName={LEAKAGE_TONES[index % LEAKAGE_TONES.length]}
          delay={Math.min(index, 10) * 0.04}
        />
        <p className="shrink-0 text-2xs text-muted-foreground tabular-nums">
          {row.waiverCount} {row.waiverCount === 1 ? "waiver" : "waivers"} ·{" "}
          {row.approverCount} {row.approverCount === 1 ? "approver" : "approvers"} ·{" "}
          {formatCurrency(perWaiver)} each
        </p>
      </div>
    </m.div>
  );
}

export function WaiverLeakage({
  rows,
  isLoading,
  isError,
  onRetry,
  index,
}: {
  rows: WaiverLeakageStat[];
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
  index: number;
}) {
  const sorted = useMemo(
    () => [...rows].sort((a, b) => b.waivedAmount - a.waivedAmount),
    [rows],
  );

  const total = useMemo(
    () => sorted.reduce((sum, row) => sum + row.waivedAmount, 0),
    [sorted],
  );

  const segments = useMemo<ShareSegment[]>(
    () =>
      sorted.map((row, rowIndex) => ({
        key: row.reason,
        label: reasonLabel(row.reason),
        value: row.waivedAmount,
        className: LEAKAGE_TONES[rowIndex % LEAKAGE_TONES.length],
        caption: formatCurrency(row.waivedAmount),
      })),
    [sorted],
  );

  const waiverCount = sorted.reduce((sum, row) => sum + row.waiverCount, 0);
  const leader = sorted[0];
  const leaderShare = leader && total > 0 ? leader.waivedAmount / total : 0;

  return (
    <Panel
      index={index}
      icon={HandCoinsIcon}
      title="Waiver leakage"
      description="Revenue forgiven at someone's discretion, grouped by the coded reason given."
      footer={
        leader ? (
          <p className="text-2xs text-muted-foreground">
            <span className="font-medium text-foreground">
              {reasonLabel(leader.reason)}
            </span>{" "}
            accounts for {Math.round(leaderShare * 100)}% of everything forgiven —{" "}
            {formatCurrency(leader.waivedAmount)} across {leader.waiverCount}{" "}
            {leader.waiverCount === 1 ? "waiver" : "waivers"}.
          </p>
        ) : null
      }
    >
      {isError ? (
        <PanelError onRetry={onRetry} />
      ) : isLoading ? (
        <PanelRowsSkeleton rows={4} />
      ) : sorted.length === 0 ? (
        <PanelEmpty
          icon={HandCoinsIcon}
          message="Nothing was waived in this window. Every accrued detention charge stood."
        />
      ) : (
        <>
          <div className="border-b border-border px-3 py-3">
            <div className="flex items-baseline justify-between gap-3">
              <p className="text-2xl leading-none font-semibold tracking-tight text-amber-600 tabular-nums dark:text-amber-400">
                {formatCurrency(total)}
              </p>
              <p className="shrink-0 text-2xs text-muted-foreground tabular-nums">
                {waiverCount} {waiverCount === 1 ? "waiver" : "waivers"} ·{" "}
                {sorted.length} {sorted.length === 1 ? "reason" : "reasons"}
              </p>
            </div>
            <ShareBreakdown segments={segments} className="mt-3" />
          </div>

          <div className="divide-y divide-border">
            {sorted.map((row, rowIndex) => (
              <LeakageRow key={row.reason} row={row} total={total} index={rowIndex} />
            ))}
          </div>
        </>
      )}
    </Panel>
  );
}
