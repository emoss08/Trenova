import { useT } from "@trenova/shared/i18n/use-t";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type { RecurringShipmentRow } from "@/lib/graphql/recurring-shipment-table";
import { queries } from "@/lib/queries";
import { cn } from "@trenova/shared/lib/utils";
import type { RecurringShipmentRun, RecurringShipmentRunStatus } from "@/types/recurring-shipment";
import { useQuery } from "@tanstack/react-query";

const runStatusStyles: Record<RecurringShipmentRunStatus, string> = {
  Generated: "border-green-600/30 bg-green-600/10 text-green-700 dark:text-green-400",
  Skipped: "border-amber-600/30 bg-amber-600/10 text-amber-700 dark:text-amber-400",
  Failed: "border-red-600/30 bg-red-600/10 text-red-700 dark:text-red-400",
};

function RunRow({ run }: { run: RecurringShipmentRun }) {
  const t = useT();

  return (
    <div className="border-border flex flex-col gap-1 rounded-md border p-2.5">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Badge variant="outline" className={cn("font-medium", runStatusStyles[run.status])}>
            {run.status}
          </Badge>
          <span className="text-2xs text-muted-foreground">{run.trigger}</span>
        </div>
        <HoverCardTimestamp timestamp={run.occurrenceAt} />
      </div>
      {run.generatedShipment?.proNumber && (
        <p className="text-sm">
          {t("Generated shipment")} <span className="font-medium">{run.generatedShipment.proNumber}</span>
        </p>
      )}
      {run.originalOccurrenceAt && run.originalOccurrenceAt !== run.occurrenceAt && (
        <p className="text-2xs text-muted-foreground">
          {t("Shifted from its original slot by the exception policy")}
        </p>
      )}
      {run.detail && <p className="text-2xs text-muted-foreground">{run.detail}</p>}
    </div>
  );
}

export function RecurringShipmentRunsDialog({
  series,
  open,
  onOpenChange,
}: {
  series: RecurringShipmentRow | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const { data, isLoading } = useQuery({
    ...queries.recurringShipment.listRuns(series?.id ?? "", { limit: 50 }),
    enabled: open && !!series?.id,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Generation History")}</DialogTitle>
          <DialogDescription>
            {t("{0} — every generated, skipped, and failed occurrence.", series ? t("Runs for \"{0}\"", series.name) : t("Runs"))}
          </DialogDescription>
        </DialogHeader>
        <div className="flex max-h-96 flex-col gap-2 overflow-y-auto pr-1">
          {isLoading && (
            <>
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-16 w-full" />
            </>
          )}
          {!isLoading && (data?.results?.length ?? 0) === 0 && (
            <p className="text-muted-foreground py-6 text-center text-sm">
              {t("Nothing generated yet. Runs appear here as the schedule fires.")}
            </p>
          )}
          {data?.results?.map((run) => (
            <RunRow key={run.id} run={run} />
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
