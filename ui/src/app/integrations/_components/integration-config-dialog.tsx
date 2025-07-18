import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import type { IntegrationSchema } from "@/lib/schemas/integration-schema";
import type { TableSheetProps } from "@/types/data-table";
import { IntegrationConfigForm } from "./integration-config-form";

type IntegrationConfigDialogProps = {
  integration: IntegrationSchema;
} & TableSheetProps;

export function IntegrationConfigDialog({
  integration,
  open,
  onOpenChange,
}: IntegrationConfigDialogProps) {
  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-md">
          <VisuallyHidden>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                {integration.name}
              </DialogTitle>
              <DialogDescription>{integration.description}</DialogDescription>
            </DialogHeader>
          </VisuallyHidden>

          <IntegrationConfigForm
            integration={integration}
            onOpenChange={onOpenChange}
          />
        </DialogContent>
      </Dialog>
    </>
  );
}
