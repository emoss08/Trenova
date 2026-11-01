import { Dialog } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { PCMilerIntegrationForm } from "./pcmiler-integration-form";
import { IntegrationDialogContent } from "../shared/integration-dialog-content";

export function PCMilerIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <IntegrationDialogContent>
        <PCMilerIntegrationForm open={open} onClose={() => onOpenChange(false)} />
      </IntegrationDialogContent>
    </Dialog>
  );
}
