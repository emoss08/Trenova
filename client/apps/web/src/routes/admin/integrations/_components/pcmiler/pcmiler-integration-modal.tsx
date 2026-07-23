import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { TableSheetProps } from "@/types/data-table";
import { PCMilerIntegrationForm } from "./pcmiler-integration-form";

export function PCMilerIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <PCMilerIntegrationForm open={open} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
