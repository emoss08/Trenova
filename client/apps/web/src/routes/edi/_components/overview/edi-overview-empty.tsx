import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";

const GHOST_TILES = 4;
const GHOST_BARS = [30, 55, 45, 70, 60, 85, 65, 50, 75, 40] as const;
const GHOST_PARTNERS: readonly { name: string; rate: number }[] = [
  { name: "w-32", rate: 96 },
  { name: "w-24", rate: 88 },
  { name: "w-28", rate: 72 },
];
const GHOST_FAILURES: readonly { reference: string; error: string }[] = [
  { reference: "w-24", error: "w-3/5" },
  { reference: "w-20", error: "w-2/5" },
];

type EDIOverviewEmptyProps = {
  title: string;
  description: string;
  /** Offered while a narrower window can still be widened. */
  onWiden?: () => void;
  widenLabel?: string;
  className?: string;
};

/**
 * The overview as it will look once documents move: the attention tiles
 * along the top, the volume chart, the partner scorecards with their
 * delivery rates, and the failures list, all faded out where the figures
 * would go. With the widest window already showing nothing, the way in is
 * a trading partner.
 */
export function EDIOverviewEmpty({
  title,
  description,
  onWiden,
  widenLabel = "Look at everything",
  className,
}: EDIOverviewEmptyProps) {
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
        ) : (
          <Button variant="outline" size="sm" render={<Link to="/edi/partners" />}>
            Set up a trading partner
          </Button>
        )
      }
      sketch={
        <div className="flex flex-col gap-3 text-left">
          <div className="grid grid-cols-4 gap-3">
            {Array.from({ length: GHOST_TILES }, (_, index) => (
              <div
                key={index}
                className="border-border/70 bg-card flex flex-col gap-2 rounded-md border p-3"
              >
                <GhostLine className="w-4/5" />
                <GhostLine className="h-3.5 w-8" />
                <GhostLine className="w-3/5" />
              </div>
            ))}
          </div>
          <div className="grid grid-cols-[minmax(0,3fr)_minmax(0,2fr)] gap-3">
            <div className="border-border/70 bg-card rounded-md border p-3">
              <GhostLine className="h-2 w-24" />
              <div className="mt-3 flex h-20 items-end gap-1">
                {GHOST_BARS.map((height, index) => (
                  <span
                    key={index}
                    className={cn(
                      "block flex-1 rounded-t-sm",
                      index % 3 === 2 ? "bg-amber-500/30" : "bg-brand/25",
                    )}
                    style={{ height: `${height}%` }}
                  />
                ))}
              </div>
            </div>
            <div className="border-border/70 bg-card divide-border/60 flex flex-col divide-y divide-dashed rounded-md border">
              {GHOST_PARTNERS.map((partner, index) => (
                <div key={index} className="flex flex-col gap-1.5 px-3 py-2.5">
                  <div className="flex items-center justify-between gap-3">
                    <GhostLine className={`h-2 ${partner.name}`} />
                    <GhostLine className="h-2 w-8" />
                  </div>
                  <GhostBar share={partner.rate} className="w-full" />
                </div>
              ))}
            </div>
          </div>
          <div className="border-border/70 bg-card divide-border/60 flex flex-col divide-y divide-dashed rounded-md border">
            {GHOST_FAILURES.map((failure, index) => (
              <div key={index} className="flex items-start gap-3 px-3 py-2.5">
                <span className="bg-muted mt-0.5 size-3.5 shrink-0 rounded-sm" />
                <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                  <span className="flex items-center gap-2">
                    <span className="border-border/70 block h-4 w-24 rounded-md border" />
                    <GhostLine className={`h-2 ${failure.reference}`} />
                  </span>
                  <GhostLine className={failure.error} />
                </div>
                <GhostLine className="w-12 shrink-0" />
              </div>
            ))}
          </div>
        </div>
      }
    />
  );
}
