import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import type { AgentDefinition } from "@/types/assistant";
import { BotIcon } from "lucide-react";

type AgentPickerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agents: AgentDefinition[];
  isPending: boolean;
  onSelect: (agentId: string) => void;
};

export function AgentPicker({ open, onOpenChange, agents, isPending, onSelect }: AgentPickerProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Choose an agent")}</DialogTitle>
          <DialogDescription>
            {t("Each agent covers a different part of the system and has its own tools.")}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          {agents.map((agent) => (
            <button
              key={agent.id}
              type="button"
              disabled={isPending}
              onClick={() => onSelect(agent.id)}
              className="border-border hover:bg-muted/60 flex flex-col items-start gap-1 rounded-md border p-3 text-left disabled:opacity-60"
            >
              <span className="flex items-center gap-2 font-medium">
                <BotIcon className="size-4" />
                {agent.name}
                {agent.toolNames.length === 0 && (
                  <Badge variant="secondary">{t("Read-only")}</Badge>
                )}
              </span>
              {agent.description && (
                <span className="text-muted-foreground text-sm">{agent.description}</span>
              )}
            </button>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
