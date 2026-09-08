import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon } from "lucide-react";

/**
 * The page as it will look with pools in it: a row per pool with its name
 * and rates, the year's rounds as a strip of quarterly slots, and how the
 * collections stand. The first ghost pool has a couple of slots drawn in,
 * so the strip reads as a calendar rather than a row of boxes.
 */
const GHOST_POOLS: readonly { name: string; detail: string; drawn: number }[] = [
  { name: "w-32", detail: "w-4/5", drawn: 2 },
  { name: "w-24", detail: "w-3/5", drawn: 0 },
];

const SLOTS_PER_YEAR = 4;

type RandomTestingEmptyProps = {
  title: string;
  description: string;
  onNewPool?: () => void;
  className?: string;
};

export function RandomTestingEmpty({
  title,
  description,
  onNewPool,
  className,
}: RandomTestingEmptyProps) {
  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onNewPool ? (
          <Button variant="outline" size="sm" onClick={onNewPool}>
            <PlusIcon className="size-3.5" />
            New pool
          </Button>
        ) : null
      }
      sketch={
        <div className="border-border/70 bg-card divide-border/60 divide-y divide-dashed rounded-md border">
          {GHOST_POOLS.map((pool, index) => (
            <div
              key={index}
              className="grid grid-cols-[minmax(0,1.2fr)_auto_minmax(0,1fr)_auto] items-center gap-x-6 px-3 py-2.5"
            >
              <div className="flex min-w-0 flex-col gap-2">
                <span className="flex items-center gap-1.5">
                  <GhostLine className={cn("h-2", pool.name)} />
                  <span className="border-border/70 block h-4 w-10 rounded-md border" />
                </span>
                <GhostLine className={pool.detail} />
              </div>
              <span className="flex gap-1">
                {Array.from({ length: SLOTS_PER_YEAR }, (_, slot) => (
                  <span
                    key={slot}
                    className={cn(
                      "block h-7 w-9 rounded-md border",
                      slot < pool.drawn
                        ? "border-brand/50 bg-brand/10"
                        : "border-border/70 border-dashed",
                    )}
                  />
                ))}
              </span>
              <GhostLine className="w-3/4" />
              <span className="border-border/70 block h-5 w-16 rounded-md border" />
            </div>
          ))}
        </div>
      }
    />
  );
}
