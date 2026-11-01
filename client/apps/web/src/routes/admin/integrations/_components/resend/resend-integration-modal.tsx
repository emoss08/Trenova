import { Dialog } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { ResendIntegrationForm } from "./resend-integration-form";
import { IntegrationDialogContent } from "../shared/integration-dialog-content";

export function ResendIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <IntegrationDialogContent>
        <ResendIntegrationForm open={open} onClose={() => onOpenChange(false)} />
      </IntegrationDialogContent>
    </Dialog>
  );
}
