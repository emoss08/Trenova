import { Dialog, DialogContent } from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { GoogleMapsForm } from "./google-integration-form";

export function GoogleIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <GoogleMapsForm open={open} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
