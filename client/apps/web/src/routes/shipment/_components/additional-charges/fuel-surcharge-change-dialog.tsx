import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

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
            <FuelIcon className="text-primary size-4" />
            {t("Fuel Surcharge Re-Rated")}
          </AlertDialogTitle>
          <AlertDialogDescription className="space-y-2">
            <span className="block">
              {t(
                "A change to this shipment (like an updated stop or distance) re-rated the automatic fuel surcharge from",
              )}{" "}
              <span className="text-foreground font-medium tabular-nums">
                {change ? money(change.previousAmount) : ""}
              </span>{" "}
              to{" "}
              <span className="text-foreground font-medium tabular-nums">
                {change ? money(change.nextAmount) : ""}
              </span>
              {t(". Only one fuel surcharge line is kept — choose which amount to bill.")}
            </span>
            <span className="block">
              {t(
                "Keeping the original locks the fuel surcharge so future changes won't re-rate it. You can unlock it from the charge list at any time.",
              )}
            </span>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => onResolve("keep")}>
            {t("Keep Original{0}", change ? ` (${money(change.previousAmount)})` : "")}
          </AlertDialogCancel>
          <AlertDialogAction onClick={() => onResolve("replace")}>
            {t("Use New Amount{0}", change ? ` (${money(change.nextAmount)})` : "")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
