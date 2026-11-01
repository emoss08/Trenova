import { Dialog } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { PostmarkIntegrationForm } from "./postmark-integration-form";
import { IntegrationDialogContent } from "../shared/integration-dialog-content";

export function PostmarkIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <IntegrationDialogContent>
        <PostmarkIntegrationForm open={open} onClose={() => onOpenChange(false)} />
      </IntegrationDialogContent>
    </Dialog>
  );
}
