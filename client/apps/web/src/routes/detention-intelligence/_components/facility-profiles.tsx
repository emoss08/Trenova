import { DwellSpreadRail, Meter } from "@/components/detention/detention-charts";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  deltaToneClass,
  formatDetentionMinutes,
  formatSignedCurrency,
} from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { FacilityDetentionStat } from "@trenova/shared/types/detention";
import { ChevronDownIcon, WarehouseIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useMemo, useState } from "react";
import {
  MetricCell,
  Panel,
  PanelEmpty,
  PanelError,
  PanelExpandToggle,
  PanelRowsSkeleton,
} from "./intelligence-panel";

const COLLAPSED_ROWS = 8;

type FacilitySort = "billed" | "margin" | "breach" | "p90";

const SORT_OPTIONS: { value: FacilitySort; label: string }[] = [
  { value: "billed", label: "Billed" },
  { value: "margin", label: "Margin" },
  { value: "breach", label: "Breach" },
  { value: "p90", label: "p90" },
];

function breachRateOf(row: FacilityDetentionStat): number {
  return row.stopCount > 0 ? row.breachCount / row.stopCount : 0;
}

function sortFacilities(
  rows: FacilityDetentionStat[],
  sort: FacilitySort,
): FacilityDetentionStat[] {
  const sorted = [...rows];

  switch (sort) {
    case "margin":
      return sorted.sort((a, b) => a.netMargin - b.netMargin);
    case "breach":
      return sorted.sort((a, b) => breachRateOf(b) - breachRateOf(a));
    case "p90":
      return sorted.sort((a, b) => b.p90DwellMinutes - a.p90DwellMinutes);
    default:
      return sorted.sort((a, b) => b.billedAmount - a.billedAmount);
  }
}

function breachToneClass(rate: number): string {
  if (rate >= 0.5) return "bg-red-500 dark:bg-red-400";
  if (rate >= 0.25) return "bg-amber-500 dark:bg-amber-400";
  return "bg-emerald-500 dark:bg-emerald-400";
}

function FacilityRow({
  row,
  rank,
  index,
  dwellScale,
}: {
  row: FacilityDetentionStat;
  rank: number;
  index: number;
  dwellScale: number;
}) {
  const [open, setOpen] = useState(false);

  const breachRate = breachRateOf(row);
  const marginPerStop = row.stopCount > 0 ? row.netMargin / row.stopCount : 0;

  return (
    <m.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.26, delay: Math.min(index, 10) * 0.03, ease: "easeOut" }}
    >
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-expanded={open}
        className="group flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors outline-none hover:bg-muted/40 focus-visible:bg-muted/40"
      >
        <span className="w-4 shrink-0 text-2xs font-medium text-muted-foreground tabular-nums">
          {rank}
        </span>

        <div className="min-w-0 flex-1">
          <p className="truncate text-sm leading-tight font-medium">
            {row.locationName || row.locationId}
          </p>
          <p className="mt-0.5 truncate text-2xs text-muted-foreground tabular-nums">
            {row.stopCount} {row.stopCount === 1 ? "stop" : "stops"} ·{" "}
            {row.breachCount} past free time
            {row.disputeCount > 0 ? ` · ${row.disputeCount} disputed` : ""}
          </p>
        </div>

        <div className="hidden w-28 shrink-0 md:block">
          <DwellSpreadRail
            median={row.medianDwellMinutes}
            p90={row.p90DwellMinutes}
            scale={dwellScale}
            delay={Math.min(index, 10) * 0.03}
          />
          <p className="mt-1.5 truncate text-2xs text-muted-foreground tabular-nums">
            {formatDetentionMinutes(Math.round(row.medianDwellMinutes))} med ·{" "}
            {formatDetentionMinutes(Math.round(row.p90DwellMinutes))} p90
          </p>
        </div>

        <div className="hidden w-16 shrink-0 sm:block">
          <Meter
            value={breachRate}
            barClassName={breachToneClass(breachRate)}
            delay={Math.min(index, 10) * 0.03}
          />
          <p className="mt-1.5 text-2xs text-muted-foreground tabular-nums">
            {Math.round(breachRate * 100)}% breach
          </p>
        </div>

        <div className="w-28 shrink-0 text-right">
          <p className="truncate text-sm leading-tight font-medium tabular-nums">
            {formatCurrency(row.billedAmount)}
          </p>
          <p
            className={cn(
              "mt-0.5 truncate text-2xs tabular-nums",
              deltaToneClass(row.netMargin),
            )}
          >
            {formatSignedCurrency(row.netMargin)} net
          </p>
        </div>

        <ChevronDownIcon
          className={cn(
            "size-3.5 shrink-0 text-muted-foreground/60 transition-transform duration-200 group-hover:text-foreground",
            open && "rotate-180",
          )}
        />
      </button>

      <AnimatePresence initial={false}>
        {open ? (
          <m.div
            key="detail"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.24, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden"
          >
            <div className="grid grid-cols-2 gap-x-4 gap-y-3 border-t border-dashed border-border bg-muted/20 px-3 py-3 sm:grid-cols-4">
              <MetricCell
                label="Average dwell"
                value={formatDetentionMinutes(Math.round(row.avgDwellMinutes))}
                detail={`${formatDetentionMinutes(
                  Math.round(row.p90DwellMinutes - row.medianDwellMinutes),
                )} tail over median`}
              />
              <MetricCell
                label="Driver pay"
                value={formatCurrency(row.driverPayAmount)}
                detail={
                  row.billedAmount > 0
                    ? `${Math.round(
                        (row.driverPayAmount / row.billedAmount) * 100,
                      )}% of billed`
                    : "nothing billed"
                }
              />
              <MetricCell
                label="Margin per stop"
                value={formatCurrency(marginPerStop)}
                valueClassName={deltaToneClass(marginPerStop)}
                detail={`${formatCurrency(row.netMargin)} total`}
              />
              <MetricCell
                label="Leakage"
                value={formatCurrency(row.waivedAmount)}
                valueClassName={
                  row.waivedAmount > 0 ? "text-amber-600 dark:text-amber-400" : undefined
                }
                detail={
                  row.suppressedCount > 0
                    ? `${row.suppressedCount} suppressed, no notice`
                    : "every charge noticed"
                }
              />
            </div>
          </m.div>
        ) : null}
      </AnimatePresence>
    </m.div>
  );
}

export function FacilityProfiles({
  rows,
  isLoading,
  isError,
  onRetry,
  index,
}: {
  rows: FacilityDetentionStat[];
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
  index: number;
}) {
  const [sort, setSort] = useState<FacilitySort>("billed");
  const [expanded, setExpanded] = useState(false);

  const sorted = useMemo(() => sortFacilities(rows, sort), [rows, sort]);
  const dwellScale = useMemo(
    () => sorted.reduce((max, row) => Math.max(max, row.p90DwellMinutes), 0),
    [sorted],
  );

  const visible = expanded ? sorted : sorted.slice(0, COLLAPSED_ROWS);
  const hidden = sorted.length - visible.length;

  return (
    <Panel
      index={index}
      icon={WarehouseIcon}
      title="Facility profiles"
      description="Where detention actually accrues — the docks worth renegotiating or replanning around. Open a row for the full ledger behind it."
      action={
        rows.length > 1 ? (
          <SegmentedControl
            items={SORT_OPTIONS}
            value={sort}
            onValueChange={setSort}
            aria-label="Rank facilities by"
          />
        ) : null
      }
      footer={
        sorted.length > 0 ? (
          <div className="flex items-center justify-between gap-3">
            <div className="hidden items-center gap-3 text-2xs text-muted-foreground md:flex">
              <span className="inline-flex items-center gap-1.5">
                <span className="h-1.5 w-4 rounded-full bg-foreground" />
                median dwell
              </span>
              <span className="inline-flex items-center gap-1.5">
                <span className="h-1.5 w-4 rounded-full bg-foreground/25" />
                p90 tail
              </span>
            </div>
            {hidden > 0 || expanded ? (
              <PanelExpandToggle
                expanded={expanded}
                hiddenCount={hidden}
                noun="facilities"
                onToggle={() => setExpanded((current) => !current)}
              />
            ) : (
              <span />
            )}
          </div>
        ) : null
      }
    >
      {isError ? (
        <PanelError onRetry={onRetry} />
      ) : isLoading ? (
        <PanelRowsSkeleton rows={6} />
      ) : sorted.length === 0 ? (
        <PanelEmpty
          icon={WarehouseIcon}
          message="No detention settled at any facility in this window. Widen the range, or check that the detention engine is switched on for this organization."
        />
      ) : (
        <div key={sort} className="divide-y divide-border">
          {visible.map((row, rowIndex) => (
            <FacilityRow
              key={row.locationId}
              row={row}
              rank={rowIndex + 1}
              index={rowIndex}
              dwellScale={dwellScale}
            />
          ))}
        </div>
      )}
    </Panel>
  );
}
