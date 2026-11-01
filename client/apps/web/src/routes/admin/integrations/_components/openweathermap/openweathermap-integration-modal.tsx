import { Dialog } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { OpenWeatherMapForm } from "./openweathermap-integration-form";
import { IntegrationDialogContent } from "../shared/integration-dialog-content";

export function OpenWeatherMapIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <IntegrationDialogContent>
        <OpenWeatherMapForm open={open} onClose={() => onOpenChange(false)} />
      </IntegrationDialogContent>
    </Dialog>
  );
}
