import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { Sparkles } from "lucide-react";

/**
 * The workspace as it will look once the period has settlements: the queue
 * down the left with a statement per carrier, and the one being worked open
 * beside it with its cost lines and the net payable at the foot.
 */
const GHOST_QUEUE: readonly { carrier: string; amount: string }[] = [
  { carrier: "w-3/5", amount: "w-10" },
  { carrier: "w-2/5", amount: "w-12" },
  { carrier: "w-1/2", amount: "w-8" },
  { carrier: "w-3/5", amount: "w-10" },
];

const GHOST_LINES: readonly { name: string; amount: string }[] = [
  { name: "w-32", amount: "w-14" },
  { name: "w-20", amount: "w-10" },
  { name: "w-28", amount: "w-12" },
];

type SettlementPeriodEmptyProps = {
  title: string;
  description: string;
  onGenerate?: () => void;
  generating?: boolean;
  className?: string;
};

export function SettlementPeriodEmpty({
  title,
  description,
  onGenerate,
  generating = false,
  className,
}: SettlementPeriodEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      className={cn("flex-1 justify-center", className)}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        onGenerate ? (
          <Button variant="outline" size="sm" disabled={generating} onClick={onGenerate}>
            <Sparkles className="size-3.5" />
            {t("Generate settlements")}
          </Button>
        ) : null
      }
      sketch={
        <div className="grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)] gap-3 text-left">
          <div className="border-border/70 bg-card divide-border/60 flex flex-col divide-y divide-dashed rounded-md border">
            {GHOST_QUEUE.map((row, index) => (
              <div key={index} className="flex flex-col gap-1.5 px-3 py-2.5">
                <div className="flex items-center justify-between gap-3">
                  <GhostLine className={`h-2 ${row.carrier}`} />
                  <GhostLine className={`h-2 ${row.amount}`} />
                </div>
                <div className="flex items-center gap-2">
                  <GhostLine className="w-12" />
                  <GhostPill className="w-12" />
                </div>
              </div>
            ))}
          </div>
          <div className="border-border/70 bg-card flex flex-col rounded-md border">
            <div className="flex flex-col gap-2.5 border-b px-4 py-3">
              <div className="flex items-center gap-2">
                <GhostLine className="h-2.5 w-28" />
                <GhostPill className="w-14" />
              </div>
              <div className="flex items-baseline gap-3">
                <GhostLine className="h-3.5 w-20" />
                <GhostLine className="w-24" />
              </div>
            </div>
            <div className="divide-border/60 flex flex-col divide-y divide-dashed px-4">
              {GHOST_LINES.map((line, index) => (
                <div key={index} className="flex items-center justify-between gap-3 py-2.5">
                  <GhostLine className={`h-2 ${line.name}`} />
                  <GhostLine className={`h-2 ${line.amount}`} />
                </div>
              ))}
            </div>
            <div className="bg-muted/40 mt-auto flex items-center justify-between border-t px-4 py-2.5">
              <GhostLine className="bg-foreground/30 h-2 w-16" />
              <GhostLine className="bg-foreground/30 h-2.5 w-16" />
            </div>
          </div>
        </div>
      }
    />
  );
}

function GhostPill({ className }: { className?: string }) {
  return <span className={cn("border-border/70 block h-4 rounded-full border", className)} />;
}
