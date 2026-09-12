import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  FUEL_FEED_RUN_LIST_KEY,
  FUEL_PURCHASE_IMPORT_ROWS_KEY,
  resolveFuelPurchaseImportRows,
  type FuelPurchaseImportBatch,
  type FuelPurchaseImportResolveResult,
} from "@/lib/graphql/fuel-purchase-import";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { RefreshCwIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import type { ImportRowFilter } from "@/lib/fuel-purchase-import";
import { ImportReviewTable } from "../../fuel-purchase/_components/import/import-review-table";

type ResolveVariables = {
  batchId: string;
  version: number;
};

/**
 * What one run read, and the rows it could not place.
 *
 * The action here is the point of the whole screen: a feed cannot know which
 * truck a newly discovered card belongs to, so its rows wait. Once somebody has
 * assigned the card or added the missing unit, working the rows out again posts
 * them without the statement being fetched a second time.
 */
export function FeedRunDetailDialog({
  open,
  onOpenChange,
  batch,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  batch: FuelPurchaseImportBatch | null;
}) {
  const t = useT();

  const queryClient = useQueryClient();
  // A run is opened to deal with what it could not place, so the held rows are
  // what it opens on. Everything else is still one click away.
  const [filter, setFilter] = useState<ImportRowFilter>("errors");

  useEffect(() => {
    if (open) {
      setFilter(batch && batch.errorCount > 0 ? "errors" : "all");
    }
  }, [open, batch]);

  // The run being worked out is passed in rather than read from the closure:
  // it is what the mutation acts on, and the version it carries is what the
  // server checks, so it has to be the one the button was pressed on.
  const { mutate, isPending } = useApiMutation<FuelPurchaseImportResolveResult, ResolveVariables>({
    resourceName: "Import",
    mutationFn: ({ batchId, version }) => resolveFuelPurchaseImportRows(batchId, version),
    onSuccess: async (result) => {
      toast.success(describeResolve(result), {
        description:
          result.queued > 0
            ? "The rows still waiting need their card assigned, or the unit they name added."
            : undefined,
      });
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: [FUEL_FEED_RUN_LIST_KEY],
          refetchType: "all",
        }),
        queryClient.invalidateQueries({
          queryKey: [FUEL_PURCHASE_IMPORT_ROWS_KEY],
          refetchType: "all",
        }),
      ]);
    },
  });

  const held = batch?.errorCount ?? 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>
            {batch ? t("{0} run", batch.provider) : t("Run")}
            <span className="text-muted-foreground ml-2 text-sm font-normal">
              {formatUnixDateTimeOrDash(batch?.createdAt)}
            </span>
          </DialogTitle>
          <DialogDescription>
            {t(
              "{0} {1} posted, {2} waiting.",
              batch?.feedReference
                ? t("Read {0}.", batch.feedReference)
                : t("Read from the provider's API."),
              batch?.committedCount ?? 0,
              held,
            )}
          </DialogDescription>
        </DialogHeader>

        {batch ? (
          <ImportReviewTable batch={batch} filter={filter} onFilterChange={setFilter} showFilters />
        ) : null}

        <DialogFooter className="flex flex-row items-center sm:justify-between">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Close")}
          </Button>
          <Button
            type="button"
            onClick={() => {
              if (batch) {
                mutate({ batchId: batch.id, version: batch.version });
              }
            }}
            isLoading={isPending}
            loadingText={t("Working them out...")}
            disabled={!batch || held === 0}
            title={held === 0 ? "This run has nothing waiting" : undefined}
          >
            <RefreshCwIcon className="size-4" />
            {t("Work rows out again")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function describeResolve(result: FuelPurchaseImportResolveResult): string {
  if (result.committed > 0) {
    return `Posted ${result.committed} of ${result.reviewed} held rows`;
  }
  if (result.resolved > 0) {
    return `${result.resolved} of ${result.reviewed} rows are ready to commit`;
  }

  return `None of the ${result.reviewed} held rows could be worked out yet`;
}
