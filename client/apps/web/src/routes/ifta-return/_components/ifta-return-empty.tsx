import { useT } from "@trenova/shared/i18n/use-t";
import { quarterLabel, type IftaPeriodKey } from "@/lib/ifta-return";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { PlayIcon } from "lucide-react";

/**
 * The workspace as it will look once the quarter has been generated: the five
 * figures across the top, then the jurisdiction lines grouped by fuel type
 * with their subtotals. The first figure is drawn a little fuller, the way
 * total miles always leads.
 */
const GHOST_TILES = ["w-14", "w-16", "w-12", "w-12", "w-16"] as const;
const GHOST_LINE_ROWS = ["w-6", "w-8", "w-6", "w-8"] as const;

type IftaReturnEmptyProps = {
  period: IftaPeriodKey;
  canCreate: boolean;
  onGenerate: () => void;
  isGenerating: boolean;
};

export function IftaReturnEmpty({
  period,
  canCreate,
  onGenerate,
  isGenerating,
}: IftaReturnEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      sketchClassName="max-w-2xl"
      title={`No return for ${quarterLabel(period)} yet`}
      description={
        canCreate
          ? "Generating reads every completed move's jurisdiction miles and every fuel purchase placed in the quarter, then applies the rates published for it. The draft can be recomputed as late miles and receipts land, and nothing is locked until you finalize it."
          : "Generating reads every completed move's jurisdiction miles and every fuel purchase placed in the quarter, then applies the rates published for it. You do not have permission to generate returns for this organization; ask an administrator to grant it, or ask whoever prepares the filing to generate the quarter."
      }
      action={
        canCreate ? (
          <Button variant="outline" size="sm" onClick={onGenerate} disabled={isGenerating}>
            <PlayIcon className="size-3.5" />
            {isGenerating ? t("Generating...") : t("Generate the return")}
          </Button>
        ) : null
      }
      sketch={
        <div className="flex flex-col gap-3">
          <div className="grid grid-cols-5 gap-2">
            {GHOST_TILES.map((width, index) => (
              <div
                key={index}
                className="border-border/70 bg-card flex flex-col gap-2 rounded-md border px-2.5 py-2"
              >
                <GhostLine className={cn("h-1", width)} />
                <span
                  className={cn(
                    "block h-3 w-10 rounded-sm",
                    index === 0 ? "bg-brand/20" : "bg-muted",
                  )}
                />
                <GhostLine className="h-1 w-3/4" />
              </div>
            ))}
          </div>
          <div className="border-border/70 bg-card flex flex-col gap-2 rounded-md border p-3">
            <GhostLine className="h-1.5 w-24" />
            <div className="border-border/70 divide-border/60 divide-y divide-dashed rounded-md border">
              {GHOST_LINE_ROWS.map((width, index) => (
                <div key={index} className="flex items-center justify-between gap-3 px-2 py-1.5">
                  <GhostLine className={cn("h-1", width)} />
                  <div className="flex items-center gap-3">
                    <GhostLine className="h-1 w-10" />
                    <GhostLine className="h-1 w-8" />
                    <GhostLine className="h-1 w-10" />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      }
    />
  );
}
