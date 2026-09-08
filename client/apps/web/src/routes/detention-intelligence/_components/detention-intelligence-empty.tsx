import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";

const LEDGER_SHARES = [
  { width: "w-1/2", className: "bg-emerald-500/30" },
  { width: "w-1/3", className: "bg-blue-500/30" },
  { width: "w-1/6", className: "bg-amber-500/30" },
] as const;
const LEDGER_STATS = 4;

const GHOST_FACILITIES: readonly { name: string; dwell: number; breach: number; money: string }[] =
  [
    { name: "w-2/5", dwell: 90, breach: 70, money: "w-14" },
    { name: "w-1/2", dwell: 65, breach: 45, money: "w-12" },
    { name: "w-1/3", dwell: 40, breach: 25, money: "w-10" },
  ];

const GHOST_PANEL_ROWS: readonly { name: string; figure: string }[] = [
  { name: "w-3/5", figure: "w-12" },
  { name: "w-2/5", figure: "w-10" },
  { name: "w-1/2", figure: "w-14" },
];

type DetentionIntelligenceEmptyProps = {
  title: string;
  description: string;
  /** Offered while the window can still be widened. */
  onWiden?: () => void;
  widenLabel?: string;
  className?: string;
};

/**
 * The page as it will look once stops have settled detention in the window:
 * the ledger strip with its kept, driver-pay and forgiven shares, the ranked
 * facilities with their dwell spread and breach rate, and the customer and
 * waiver panels side by side beneath.
 */
export function DetentionIntelligenceEmpty({
  title,
  description,
  onWiden,
  widenLabel = "Look back further",
  className,
}: DetentionIntelligenceEmptyProps) {
  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onWiden ? (
          <Button variant="outline" size="sm" onClick={onWiden}>
            {widenLabel}
          </Button>
        ) : null
      }
      sketch={
        <div className="flex flex-col gap-3 text-left">
          <div className="border-border/70 bg-card rounded-lg border">
            <div className="flex flex-col gap-2 border-b px-3.5 py-3">
              <GhostLine className="h-2 w-28" />
              <span className="bg-muted flex h-2 w-full overflow-hidden rounded-full">
                {LEDGER_SHARES.map((share, index) => (
                  <span key={index} className={cn("h-full", share.width, share.className)} />
                ))}
              </span>
            </div>
            <div className="divide-border/60 grid grid-cols-4 divide-x">
              {Array.from({ length: LEDGER_STATS }, (_, index) => (
                <div key={index} className="flex flex-col gap-1.5 px-3.5 py-2.5">
                  <GhostLine className="w-14" />
                  <GhostLine className="h-2.5 w-16" />
                  <GhostLine className="w-20" />
                </div>
              ))}
            </div>
          </div>
          <div className="border-border/70 bg-card divide-border/60 flex flex-col divide-y divide-dashed rounded-lg border">
            {GHOST_FACILITIES.map((row, index) => (
              <div key={index} className="flex items-center gap-3 px-3 py-2.5">
                <span className="bg-muted size-3 shrink-0 rounded-sm" />
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <GhostLine className={`h-2 ${row.name}`} />
                  <GhostLine className="w-24" />
                </div>
                <GhostBar share={row.dwell} className="w-24 shrink-0" />
                <GhostBar share={row.breach} className="w-14 shrink-0" />
                <div className="flex w-20 shrink-0 flex-col items-end gap-1.5">
                  <GhostLine className={`h-2 ${row.money}`} />
                  <GhostLine className="w-12" />
                </div>
              </div>
            ))}
          </div>
          <div className="grid grid-cols-2 gap-3">
            {Array.from({ length: 2 }, (_, panel) => (
              <div
                key={panel}
                className="border-border/70 bg-card divide-border/60 flex flex-col divide-y divide-dashed rounded-lg border"
              >
                <div className="flex items-center gap-2 px-3 py-2.5">
                  <span className="bg-muted size-5 shrink-0 rounded-md" />
                  <GhostLine className="h-2 w-24" />
                </div>
                {GHOST_PANEL_ROWS.map((row, index) => (
                  <div key={index} className="flex items-center justify-between gap-3 px-3 py-2.5">
                    <GhostLine className={row.name} />
                    <GhostLine className={`h-2 ${row.figure}`} />
                  </div>
                ))}
              </div>
            ))}
          </div>
        </div>
      }
    />
  );
}
