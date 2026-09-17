import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { queries } from "@/lib/queries";
import type { AgentDefinition } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { ChevronRightIcon } from "lucide-react";
import { useState } from "react";
import { AgentAvatar } from "./message-items";

type AgentPickerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  agents: AgentDefinition[];
  isPending: boolean;
  onSelect: (agentId: string) => void;
};

/**
 * Choosing who to talk to. Each agent is shown with its template and reach,
 * because "can it change anything" is the question people have before they
 * ask one anything.
 */
export function AgentPicker({ open, onOpenChange, agents, isPending, onSelect }: AgentPickerProps) {
  const t = useT();
  const [pendingId, setPendingId] = useState<string | null>(null);

  const templatesQuery = useQuery({ ...queries.assistant.agentTemplates(), enabled: open });
  const templates = templatesQuery.data?.templates ?? [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Who do you want to ask?")}</DialogTitle>
          <DialogDescription>
            {t(
              "Each agent covers a different part of the system and has its own tools. Nothing an agent proposes runs until you approve it.",
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          {agents.map((agent) => {
            const templateLabel = templates.find((item) => item.template === agent.template)?.label;
            const busy = isPending && pendingId === agent.id;

            return (
              <button
                key={agent.id}
                type="button"
                disabled={isPending}
                onClick={() => {
                  setPendingId(agent.id);
                  onSelect(agent.id);
                }}
                className="border-border hover:bg-muted/60 focus-visible:ring-brand/20 flex items-start gap-3 rounded-md border p-3 text-left transition-colors focus-visible:ring-4 focus-visible:outline-none disabled:opacity-60"
              >
                <AgentAvatar className="mt-0.5 size-8" />
                <span className="flex min-w-0 flex-1 flex-col gap-1">
                  <span className="flex flex-wrap items-center gap-x-2 gap-y-1">
                    <span className="text-sm font-medium">{agent.name}</span>
                    {templateLabel && <Badge variant="outline">{templateLabel}</Badge>}
                    <Badge variant="secondary">
                      {agent.toolNames.length === 0
                        ? t("Answers only")
                        : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
                    </Badge>
                  </span>
                  {agent.description && (
                    <span className="text-muted-foreground text-xs">{agent.description}</span>
                  )}
                </span>
                {busy ? (
                  <Spinner className="text-muted-foreground mt-2 size-4 shrink-0" />
                ) : (
                  <ChevronRightIcon className="text-muted-foreground mt-2 size-4 shrink-0" />
                )}
              </button>
            );
          })}
        </div>
      </DialogContent>
    </Dialog>
  );
}
