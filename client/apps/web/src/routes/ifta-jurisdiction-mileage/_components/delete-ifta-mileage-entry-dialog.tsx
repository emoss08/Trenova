import { useT } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";
import {
  deleteIftaMileageEntry,
  type IftaMileageEntryRow,
} from "@/lib/graphql/ifta-jurisdiction-mileage";
import { quarterLabel } from "@/lib/ifta-return";
import { useMutation } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatDecimalString } from "@trenova/shared/types/decimal";
import { IFTA_MILEAGE_SOURCE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { IFTA_MILES_SCALE } from "@trenova/shared/types/ifta-jurisdiction-mileage";
import { Trash2Icon } from "lucide-react";
import { toast } from "sonner";

type DeleteIftaMileageEntryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  entry: IftaMileageEntryRow | null;
  onDeleted: () => Promise<void> | void;
};

export function DeleteIftaMileageEntryDialog({
  open,
  onOpenChange,
  entry,
  onDeleted,
}: DeleteIftaMileageEntryDialogProps) {
  const t = useT();

  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!entry) throw new Error("No mileage entry selected");
      return deleteIftaMileageEntry(entry.id, entry.version);
    },
    onSuccess: async () => {
      toast.success(t("Entry deleted"), {
        description: t("Recompute the quarter's return to take these miles out of the figures."),
      });
      await onDeleted();
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "Jurisdiction Mileage" }),
  });

  const period = entry ? quarterLabel({ year: entry.year, quarter: entry.quarter }) : null;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-destructive/10 text-destructive">
            <Trash2Icon />
          </AlertDialogMedia>
          <AlertDialogTitle>{t("Delete this entry?")}</AlertDialogTitle>
          <AlertDialogDescription>
            {entry ? (
              <span className="block">
                {t(
                  "{0} miles in {1} on {2}{3} will be removed outright.",
                  formatDecimalString(entry.miles, IFTA_MILES_SCALE),
                  entry.jurisdiction.code,
                  formatUnixDate(entry.traveledAt),
                  entry.tractor?.code ? ` ${t("for tractor {0}", entry.tractor.code)}` : "",
                )}
              </span>
            ) : null}
            <span className="mt-2 block">
              {t(
                "The quarter's return drops these miles on its next recompute. A return already generated for {0} keeps its figures until it is recomputed, so recompute it after deleting while the quarter is still open.",
                period ?? t("the quarter"),
              )}
            </span>
            {entry && entry.source !== "Manual" ? (
              <span className="mt-2 block">
                {t(
                  "These miles were written by {0} , not keyed by hand, so deleting removes the system's own record of the travel.",
                  IFTA_MILEAGE_SOURCE_LABELS[entry.source].toLowerCase(),
                )}
              </span>
            ) : null}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>{t("Keep entry")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => mutate()}
            disabled={isPending || !entry}
          >
            {isPending ? t("Deleting...") : t("Delete entry")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
