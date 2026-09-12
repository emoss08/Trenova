import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon } from "lucide-react";
import { Link } from "react-router";

/**
 * The page as it will look once a batch has been imported: four figures
 * across the top, the exception aging table on the left and the work items
 * on the right. The first figure is drawn a little fuller than the rest, the
 * way "imported" always leads "matched".
 */
const GHOST_FIGURES: readonly { label: string; value: string }[] = [
  { label: "w-12", value: "w-10" },
  { label: "w-10", value: "w-8" },
  { label: "w-14", value: "w-6" },
  { label: "w-14", value: "w-8" },
];
const GHOST_AGING_ROWS = ["w-12", "w-14", "w-14", "w-12"] as const;
const GHOST_WORK_ROWS = ["w-12", "w-14", "w-16"] as const;

type ReconciliationSummaryEmptyProps = {
  title: string;
  description: string;
  className?: string;
};

export function ReconciliationSummaryEmpty({
  title,
  description,
  className,
}: ReconciliationSummaryEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      sketchClassName="max-w-2xl"
      title={title}
      description={description}
      action={
        <Link to="/accounting/reconciliation/import-batches">
          <Button variant="outline" size="sm">
            {t("Import a batch")}
            <ArrowRightIcon className="size-3.5" />
          </Button>
        </Link>
      }
      sketch={
        <div className="flex flex-col gap-3">
          <div className="grid grid-cols-4 gap-2.5">
            {GHOST_FIGURES.map((figure, index) => (
              <div
                key={index}
                className="border-border/70 bg-card flex flex-col gap-2 rounded-md border px-3 py-2.5"
              >
                <GhostLine className={cn("h-1", figure.label)} />
                <span
                  className={cn(
                    "block h-3 rounded-sm",
                    index === 0 ? "bg-brand/20" : "bg-muted",
                    figure.value,
                  )}
                />
                <GhostLine className="h-1 w-3/4" />
              </div>
            ))}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="border-border/70 bg-card flex flex-col gap-2 rounded-md border p-3">
              <GhostLine className="h-1.5 w-20" />
              <div className="border-border/70 divide-border/60 divide-y divide-dashed rounded-md border">
                {GHOST_AGING_ROWS.map((width, index) => (
                  <div key={index} className="flex items-center justify-between px-2 py-1.5">
                    <GhostLine className={cn("h-1", width)} />
                    <GhostLine className="h-1 w-4" />
                  </div>
                ))}
              </div>
            </div>
            <div className="border-border/70 bg-card flex flex-col gap-2 rounded-md border p-3">
              <GhostLine className="h-1.5 w-16" />
              <div className="flex flex-col gap-2.5 pt-1">
                {GHOST_WORK_ROWS.map((width, index) => (
                  <div key={index} className="flex items-center justify-between">
                    <GhostLine className={cn("h-1", width)} />
                    <GhostLine className="h-1 w-4" />
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      }
    />
  );
}
