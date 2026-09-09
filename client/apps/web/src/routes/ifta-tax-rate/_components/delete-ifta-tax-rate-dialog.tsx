import { jurisdictionLabel } from "@/components/fields/ifta-jurisdiction-select-field";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { deleteIftaTaxRate, type IftaTaxRateRow } from "@/lib/graphql/ifta-tax-rate";
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
import { IFTA_FUEL_TYPE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { Trash2Icon } from "lucide-react";
import { toast } from "sonner";
import { periodLabel } from "./ifta-tax-rate-columns";

type DeleteIftaTaxRateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rate: IftaTaxRateRow | null;
  onDeleted: () => Promise<void> | void;
};

export function DeleteIftaTaxRateDialog({
  open,
  onOpenChange,
  rate,
  onDeleted,
}: DeleteIftaTaxRateDialogProps) {
  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!rate) throw new Error("No tax rate selected");
      return deleteIftaTaxRate(rate.id, rate.version);
    },
    onSuccess: async () => {
      toast.success("Rate deleted", {
        description: "Returns for the quarter will flag the missing rate when recomputed.",
      });
      await onDeleted();
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "IFTA Tax Rate" }),
  });

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-destructive/10 text-destructive">
            <Trash2Icon />
          </AlertDialogMedia>
          <AlertDialogTitle>Delete this rate?</AlertDialogTitle>
          <AlertDialogDescription>
            {rate ? (
              <span className="block">
                The {IFTA_FUEL_TYPE_LABELS[rate.fuelType].toLowerCase()} rate for{" "}
                {jurisdictionLabel(rate.jurisdiction)} in {periodLabel(rate.year, rate.quarter)} (
                {rate.ratePerGallon} per gallon) will be removed.
              </span>
            ) : null}
            <span className="mt-2 block">
              Rates are global. Every organization&apos;s return for that quarter will report a
              missing rate on this line the next time it is recomputed, and none of them can be
              finalized until a rate is published again.
            </span>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>Keep rate</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => mutate()}
            disabled={isPending || !rate}
          >
            {isPending ? "Deleting..." : "Delete rate"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
