import { useT } from "@trenova/shared/i18n/use-t";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import type { AgentDefinition, AgentTemplate } from "@/types/assistant";
import { AgentForm } from "./agent-form";

type AgentDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agent: AgentDefinition | null;
  templates: AgentTemplate[];
  onSaved: () => Promise<void> | void;
};

export function AgentDialog({ open, onOpenChange, agent, templates, onSaved }: AgentDialogProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{agent ? t("Edit agent") : t("Add agent")}</DialogTitle>
          <DialogDescription>
            {t(
              "Pick a template, then narrow it. A template sets the outer limit of what the agent can reach; your choices can only restrict it further.",
            )}
          </DialogDescription>
        </DialogHeader>
        <AgentForm
          key={agent?.id ?? "new"}
          agent={agent}
          templates={templates}
          onClose={() => onOpenChange(false)}
          onSaved={onSaved}
        />
      </DialogContent>
    </Dialog>
  );
}
