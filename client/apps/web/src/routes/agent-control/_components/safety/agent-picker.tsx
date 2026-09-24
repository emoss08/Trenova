import type { AgentSafety } from "@/lib/graphql/agent-safety";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { fieldTriggerClass } from "@trenova/shared/lib/variants/field";
import { CheckIcon, ChevronDownIcon } from "lucide-react";
import { useState } from "react";

type AgentPickerProps = {
  agents: readonly AgentSafety[];
  picked: readonly string[];
  onChange: (picked: string[]) => void;
};

/** Chooses which agents the matrix lays side by side. */
export function AgentPicker({ agents, picked, onChange }: AgentPickerProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const pickedSet = new Set(picked);

  const toggle = (agentId: string) => {
    onChange(pickedSet.has(agentId) ? picked.filter((id) => id !== agentId) : [...picked, agentId]);
  };

  const summary =
    picked.length === 0
      ? t("Pick agents")
      : picked.length === 1
        ? (agents.find((safety) => safety.agentId === picked[0])?.agent.name ?? t("1 agent"))
        : t("{0, plural, one {# agent} other {# agents}}", picked.length);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            role="combobox"
            aria-expanded={open}
            aria-label={t("Agents to compare")}
            className={cn(fieldTriggerClass, "w-48 justify-between font-normal")}
          >
            <span className="truncate">{summary}</span>
            <ChevronDownIcon className="text-muted-foreground size-3 shrink-0" />
          </Button>
        }
      />
      <PopoverContent align="end" className="w-64 p-0">
        <Command>
          <CommandInput placeholder={t("Search agents")} />
          <CommandList className="max-h-64 overflow-y-auto">
            <CommandEmpty>{t("No agent matches.")}</CommandEmpty>
            <CommandGroup>
              {agents.map((safety) => (
                <CommandItem
                  key={safety.agentId}
                  value={`${safety.agent.name} ${safety.agentId}`}
                  onSelect={() => toggle(safety.agentId)}
                >
                  <span className="truncate">{safety.agent.name}</span>
                  <CheckIcon
                    className={cn(
                      "ml-auto size-3",
                      pickedSet.has(safety.agentId) ? "opacity-100" : "opacity-0",
                    )}
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
