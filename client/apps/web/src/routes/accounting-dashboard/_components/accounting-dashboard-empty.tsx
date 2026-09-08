import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Link } from "react-router";

const GHOST_KPIS = 5;
const TREND_POINTS = "0,30 20,26 40,28 60,20 80,22 100,15 120,17 140,11 160,13 180,8";
const GHOST_BARS = [40, 65, 50, 80, 60, 90, 70, 55] as const;
const GHOST_AGING: readonly number[] = [70, 45, 30, 20, 10];

type AccountingDashboardEmptyProps = {
  title: string;
  description: string;
  className?: string;
};

/**
 * The dashboard as it will look once receivables exist: the KPI strip
 * along the top, the DSO line and the cash-flow bars beneath it, and the
 * aging buckets under those, all faded out where the figures would go.
 * The way in is billing, so the action leads to the queue.
 */
export function AccountingDashboardEmpty({
  title,
  description,
  className,
}: AccountingDashboardEmptyProps) {
  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        <Button variant="outline" size="sm" render={<Link to="/billing/queue" />}>
          Open the billing queue
        </Button>
      }
      sketch={
        <div className="flex flex-col gap-3 text-left">
          <div className="grid grid-cols-5 gap-2">
            {Array.from({ length: GHOST_KPIS }, (_, index) => (
              <div
                key={index}
                className="border-border/70 bg-card flex flex-col gap-2 rounded-md border p-3"
              >
                <GhostLine className="w-3/5" />
                <GhostLine className="h-3 w-12" />
                <GhostLine className="w-4/5" />
              </div>
            ))}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="border-border/70 bg-card rounded-md border p-3">
              <div className="flex items-center justify-between">
                <GhostLine className="h-2 w-20" />
                <span className="flex gap-1">
                  <GhostPill />
                  <GhostPill />
                  <GhostPill />
                </span>
              </div>
              <svg viewBox="0 0 180 36" className="mt-3 h-24 w-full" preserveAspectRatio="none">
                <polyline
                  points={TREND_POINTS}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  className="text-brand/40"
                  vectorEffect="non-scaling-stroke"
                />
              </svg>
            </div>
            <div className="border-border/70 bg-card rounded-md border p-3">
              <div className="flex items-center justify-between">
                <GhostLine className="h-2 w-28" />
                <GhostLine className="w-16" />
              </div>
              <div className="mt-3 flex h-24 items-end gap-1.5">
                {GHOST_BARS.map((height, index) => (
                  <span
                    key={index}
                    className="bg-brand/25 block flex-1 rounded-t-sm"
                    style={{ height: `${height}%` }}
                  />
                ))}
              </div>
            </div>
          </div>
          <div className="border-border/70 bg-card flex flex-col gap-2 rounded-md border p-3">
            <GhostLine className="h-2 w-16" />
            {GHOST_AGING.map((share, index) => (
              <span key={index} className="flex items-center gap-3">
                <GhostLine className="w-12 shrink-0" />
                <GhostBar share={share} className="w-full" />
                <GhostLine className="h-2 w-10 shrink-0" />
              </span>
            ))}
          </div>
        </div>
      }
    />
  );
}

function GhostPill() {
  return <span className="border-border/70 block h-4 w-8 rounded-md border" />;
}
