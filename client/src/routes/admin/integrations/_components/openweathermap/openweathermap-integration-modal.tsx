import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { TableSheetProps } from "@/types/data-table";
import { OpenWeatherMapForm } from "./openweathermap-integration-form";

export function OpenWeatherMapIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <OpenWeatherMapForm open={open} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
