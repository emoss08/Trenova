import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import type { FuelSurchargeChange } from "@/hooks/use-shipment-totals-preview";
import { FuelIcon } from "lucide-react";

function money(value: number) {
  return `$${value.toFixed(2)}`;
}

type FuelSurchargeChangeDialogProps = {
  change: FuelSurchargeChange | null;
  onResolve: (action: "replace" | "keep" | "dismiss") => void;
};

export function FuelSurchargeChangeDialog({ change, onResolve }: FuelSurchargeChangeDialogProps) {
  return (
    <AlertDialog
      open={!!change}
      onOpenChange={(open) => {
        if (!open) onResolve("dismiss");
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <FuelIcon className="size-4 text-primary" />
            Fuel Surcharge Re-Rated
          </AlertDialogTitle>
          <AlertDialogDescription className="space-y-2">
            <span className="block">
              A change to this shipment (like an updated stop or distance) re-rated the automatic
              fuel surcharge from{" "}
              <span className="font-medium text-foreground tabular-nums">
                {change ? money(change.previousAmount) : ""}
              </span>{" "}
              to{" "}
              <span className="font-medium text-foreground tabular-nums">
                {change ? money(change.nextAmount) : ""}
              </span>
              . Only one fuel surcharge line is kept — choose which amount to bill.
            </span>
            <span className="block">
              Keeping the original locks the fuel surcharge so future changes won&apos;t re-rate it.
              You can unlock it from the charge list at any time.
            </span>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => onResolve("keep")}>
            Keep Original{change ? ` (${money(change.previousAmount)})` : ""}
          </AlertDialogCancel>
          <AlertDialogAction onClick={() => onResolve("replace")}>
            Use New Amount{change ? ` (${money(change.nextAmount)})` : ""}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
