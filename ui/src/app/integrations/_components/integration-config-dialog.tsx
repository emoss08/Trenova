/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
