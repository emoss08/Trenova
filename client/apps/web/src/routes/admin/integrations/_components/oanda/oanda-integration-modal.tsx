import { Dialog } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { OANDAExchangeRatesForm } from "./oanda-integration-form";
import { IntegrationDialogContent } from "../shared/integration-dialog-content";

export function OANDAExchangeRatesIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <IntegrationDialogContent>
        <OANDAExchangeRatesForm open={open} onClose={() => onOpenChange(false)} />
      </IntegrationDialogContent>
    </Dialog>
  );
}
