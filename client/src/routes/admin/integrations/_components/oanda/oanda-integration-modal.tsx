import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { TableSheetProps } from "@/types/data-table";
import { OANDAExchangeRatesForm } from "./oanda-integration-form";

export function OANDAExchangeRatesIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <OANDAExchangeRatesForm open={open} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
