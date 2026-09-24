import { AgentPickerList } from "@/components/assistant/agent-picker";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";

const NO_RECENT: readonly string[] = [];

/** The most agents laid side by side; each is read on its own, so the cap bounds the reads. */
export const MAX_COMPARED_AGENTS = 10;

type AgentPickerProps = {
  picked: readonly string[];
  onAdd: (agent: AgentChoice) => void;
};

/**
 * Adds an agent to the comparison. The list is the organization's, read from
 * the server a page at a time and searched there, so opening it costs the
 * same with three agents as with three hundred; agents already shown are left
 * out of it.
 */
export function AgentPicker({ picked, onAdd }: AgentPickerProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const hidden = useMemo(() => new Set(picked), [picked]);
  const full = picked.length >= MAX_COMPARED_AGENTS;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={full}
        render={<Button type="button" size="sm" variant="outline" />}
        aria-label={full ? t("Compare up to 10 agents at once") : undefined}
      >
        <PlusIcon className="size-3.5" />
        {t("Add an agent")}
      </PopoverTrigger>
      <PopoverContent
        align="end"
        side="bottom"
        sideOffset={6}
        className="ui-lift-float w-80 gap-0 overflow-hidden p-0"
      >
        {open ? (
          <AgentPickerList
            selectedId={null}
            recentIds={NO_RECENT}
            hiddenIds={hidden}
            source="grantable"
            onSelect={(agent) => {
              onAdd(agent);
              setOpen(false);
            }}
            emptyMessage={t("Every agent is already shown.")}
          />
        ) : null}
      </PopoverContent>
    </Popover>
  );
}
