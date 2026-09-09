import { handleMutationError } from "@/hooks/use-api-mutation";
import { purchaseQuarterLabel } from "@/lib/fuel-purchase";
import { deleteFuelPurchase, type FuelPurchaseRow } from "@/lib/graphql/fuel-purchase";
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
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { Trash2Icon } from "lucide-react";
import { toast } from "sonner";

type DeleteFuelPurchaseDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  purchase: FuelPurchaseRow | null;
  onDeleted: () => Promise<void> | void;
};

export function DeleteFuelPurchaseDialog({
  open,
  onOpenChange,
  purchase,
  onDeleted,
}: DeleteFuelPurchaseDialogProps) {
  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!purchase) throw new Error("No fuel purchase selected");
      return deleteFuelPurchase(purchase.id, purchase.version);
    },
    onSuccess: async () => {
      toast.success("Purchase deleted", {
        description: "Recompute the quarter's return to take it out of the figures.",
      });
      await onDeleted();
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "Fuel Purchase" }),
  });

  const quarter = purchase ? purchaseQuarterLabel(purchase.purchasedAt) : null;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-destructive/10 text-destructive">
            <Trash2Icon />
          </AlertDialogMedia>
          <AlertDialogTitle>Delete this fuel purchase?</AlertDialogTitle>
          <AlertDialogDescription>
            {purchase ? (
              <span className="block">
                {purchase.gallons} gallons of {purchase.fuelType.toLowerCase()} bought in{" "}
                {purchase.jurisdiction.code} on {formatUnixDateTime(purchase.purchasedAt)}
                {purchase.tractor?.code ? ` for tractor ${purchase.tractor.code}` : ""} will be
                removed outright.
              </span>
            ) : null}
            <span className="mt-2 block">
              A return already generated for {quarter ?? "its quarter"} keeps its figures until it
              is recomputed, so recompute the return after deleting if the quarter is still open.
              {purchase?.source === "CardImport"
                ? " Importing the same statement again will record this row afresh."
                : ""}
            </span>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>Keep purchase</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => mutate()}
            disabled={isPending || !purchase}
          >
            {isPending ? "Deleting..." : "Delete purchase"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
