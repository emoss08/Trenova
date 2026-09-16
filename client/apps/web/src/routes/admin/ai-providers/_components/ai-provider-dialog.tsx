import { useT } from "@trenova/shared/i18n/use-t";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import type { AIProvider } from "@/types/ai-provider";
import { AIProviderForm } from "./ai-provider-form";

type AIProviderDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  provider: AIProvider | null;
  onSaved: () => Promise<void> | void;
};

export function AIProviderDialog({ open, onOpenChange, provider, onSaved }: AIProviderDialogProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{provider ? t("Edit AI provider") : t("Add AI provider")}</DialogTitle>
          <DialogDescription>
            {t(
              "Point Trenova at a model endpoint and choose which work it handles. Start from a preset, or configure any OpenAI-compatible server directly.",
            )}
          </DialogDescription>
        </DialogHeader>
        <AIProviderForm
          key={provider?.id ?? "new"}
          provider={provider}
          onClose={() => onOpenChange(false)}
          onSaved={onSaved}
        />
      </DialogContent>
    </Dialog>
  );
}
