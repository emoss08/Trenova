import { DivergingBar } from "@/components/detention/detention-charts";
import { Badge } from "@trenova/shared/components/ui/badge";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { deltaToneClass, formatSignedCurrency } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { CustomerDetentionStat } from "@trenova/shared/types/detention";
import { UsersIcon } from "lucide-react";
import { m } from "motion/react";
import { useMemo, useState } from "react";
import {
  Panel,
  PanelEmpty,
  PanelError,
  PanelExpandToggle,
  PanelRowsSkeleton,
} from "./intelligence-panel";

const COLLAPSED_ROWS = 8;

type CustomerSort = "margin" | "billed";

const SORT_OPTIONS: { value: CustomerSort; label: string }[] = [
  { value: "margin", label: "Worst margin" },
  { value: "billed", label: "Most billed" },
];

function sortCustomers(rows: CustomerDetentionStat[], sort: CustomerSort): CustomerDetentionStat[] {
  const sorted = [...rows];

  return sort === "billed"
    ? sorted.sort((a, b) => b.billedAmount - a.billedAmount)
    : sorted.sort((a, b) => a.netMargin - b.netMargin);
}

function CustomerRow({
  row,
  index,
  scale,
}: {
  row: CustomerDetentionStat;
  index: number;
  scale: number;
}) {
  const losing = row.netMargin < 0;

  return (
    <m.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.26, delay: Math.min(index, 10) * 0.03, ease: "easeOut" }}
      className="hover:bg-muted/40 flex items-center gap-3 px-3 py-2.5 transition-colors"
    >
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-center gap-1.5">
          <p className="truncate text-sm leading-tight font-medium">
            {row.customerName || row.customerId}
          </p>
          {losing ? (
            <Badge variant="inactive" className="text-2xs h-4 shrink-0 px-1">
              Loss
            </Badge>
          ) : null}
          {row.disputeCount > 0 ? (
            <Badge variant="outline" className="text-2xs h-4 shrink-0 px-1">
              {row.disputeCount} disputed
            </Badge>
          ) : null}
        </div>
        <p className="text-2xs text-muted-foreground mt-0.5 truncate tabular-nums">
          {formatCurrency(row.billedAmount)} billed · {formatCurrency(row.driverPayAmount)} paid out
          · {row.stopCount} {row.stopCount === 1 ? "stop" : "stops"}
        </p>
      </div>

      <DivergingBar
        value={row.netMargin}
        scale={scale}
        className="hidden sm:block"
        delay={Math.min(index, 10) * 0.03}
      />

      <span
        className={cn(
          "w-24 shrink-0 text-right text-sm font-medium tabular-nums",
          deltaToneClass(row.netMargin),
        )}
      >
        {formatSignedCurrency(row.netMargin)}
      </span>
    </m.div>
  );
}

export function CustomerMargin({
  rows,
  isLoading,
  isError,
  onRetry,
  index,
}: {
  rows: CustomerDetentionStat[];
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
  index: number;
}) {
  const [sort, setSort] = useState<CustomerSort>("margin");
  const [expanded, setExpanded] = useState(false);

  const sorted = useMemo(() => sortCustomers(rows, sort), [rows, sort]);
  const scale = useMemo(
    () => sorted.reduce((max, row) => Math.max(max, Math.abs(row.netMargin)), 0),
    [sorted],
  );
  const losingCount = useMemo(() => rows.filter((row) => row.netMargin < 0).length, [rows]);

  const visible = expanded ? sorted : sorted.slice(0, COLLAPSED_ROWS);
  const hidden = sorted.length - visible.length;

  return (
    <Panel
      index={index}
      icon={UsersIcon}
      title="Customer margin"
      description="Detention billed against detention paid. A negative margin means the customer's free-time concession is wider than the driver contract grants."
      action={
        rows.length > 1 ? (
          <SegmentedControl
            items={SORT_OPTIONS}
            value={sort}
            onValueChange={setSort}
            aria-label="Rank customers by"
          />
        ) : null
      }
      footer={
        sorted.length > 0 ? (
          <div className="flex items-center justify-between gap-3">
            <p className="text-2xs text-muted-foreground tabular-nums">
              {losingCount > 0
                ? `${losingCount} of ${rows.length} lose money on detention`
                : "Every customer clears a positive detention margin"}
            </p>
            {hidden > 0 || expanded ? (
              <PanelExpandToggle
                expanded={expanded}
                hiddenCount={hidden}
                noun="customers"
                onToggle={() => setExpanded((current) => !current)}
              />
            ) : null}
          </div>
        ) : null
      }
    >
      {isError ? (
        <PanelError onRetry={onRetry} />
      ) : isLoading ? (
        <PanelRowsSkeleton rows={5} />
      ) : sorted.length === 0 ? (
        <PanelEmpty
          icon={UsersIcon}
          message="No customer accrued settled detention in this window."
        />
      ) : (
        <div key={sort} className="divide-border divide-y">
          {visible.map((row, rowIndex) => (
            <CustomerRow key={row.customerId} row={row} index={rowIndex} scale={scale} />
          ))}
        </div>
      )}
    </Panel>
  );
}
