import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { TableSheetProps } from "@/types/data-table";
import { OpenAIIntegrationForm } from "./openai-integration-form";

export function OpenAIIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <OpenAIIntegrationForm open={open} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}
