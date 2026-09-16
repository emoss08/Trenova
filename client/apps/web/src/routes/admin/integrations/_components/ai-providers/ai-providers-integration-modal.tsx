import { useT } from "@trenova/shared/i18n/use-t";
import { AIProviderList } from "@/routes/admin/ai-providers/_components/ai-provider-list";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { Link } from "react-router";

/**
 * The provider manager, opened from its marketplace card.
 *
 * It is the same list the AI Providers settings page shows rather than a
 * cut-down form, because a provider is not one credential: it is an endpoint,
 * a model, and the work it is allowed to take, and an administrator connecting
 * their first one needs to see all of that to know it will do anything.
 */
export function AIProvidersIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{t("AI Providers")}</DialogTitle>
          <DialogDescription>
            {t(
              "Model endpoints for the assistant, agents and operational insights. Work is offered to providers in priority order, so a cheap local model can take the high-volume tasks and a stronger one can sit behind it.",
            )}{" "}
            <Link to="/admin/agent-control" className="text-foreground underline">
              {t("Agent Control")}
            </Link>{" "}
            {t("shows which features these providers make ready.")}
          </DialogDescription>
        </DialogHeader>
        {open && <AIProviderList />}
      </DialogContent>
    </Dialog>
  );
}
