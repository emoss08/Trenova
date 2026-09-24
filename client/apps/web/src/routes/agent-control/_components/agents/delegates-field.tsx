import { AgentTile } from "@/components/agent-identity/agent-tile";
import { AgentPickerList } from "@/components/assistant/agent-picker";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { MAX_DELEGATES } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon, XIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useController, useFormContext } from "react-hook-form";
import type { AgentFormValues } from "./agent-form-schema";
import type { DelegateSummary } from "./delegates";

const NO_RECENT: readonly string[] = [];

/**
 * Who this agent may hand a task to: at most eight agents people talk to,
 * each drawn with its mark and name, in the order they were chosen.
 *
 * The list is the ids the form saves. Names and marks come from the agents
 * as saved and from the picker as they are added, so an agent disabled since
 * it was added still reads as itself, flagged, rather than as an id. The
 * agent being edited is never offered, and neither is one already chosen.
 * Each row registers its own path, so the server's `delegateIds[i]` error
 * lands on the agent it is about.
 */
export function DelegatesField({
  agentId,
  saved,
}: {
  /** The agent being edited; empty while it is being created. */
  agentId: string;
  /** The allowlist as it was loaded, with names and marks. */
  saved: readonly DelegateSummary[];
}) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const { field, fieldState } = useController({ control, name: "delegateIds" });
  const ids = useMemo(() => field.value ?? [], [field.value]);
  const [open, setOpen] = useState(false);
  // What was picked in this session, so a new row draws its agent at once.
  const [picked, setPicked] = useState<ReadonlyMap<string, DelegateSummary>>(() => new Map());

  const known = useMemo(() => {
    const byId = new Map<string, DelegateSummary>();
    for (const delegate of saved) {
      byId.set(delegate.id, delegate);
    }
    for (const [id, delegate] of picked) {
      byId.set(id, delegate);
    }
    return byId;
  }, [picked, saved]);

  const hidden = useMemo(() => {
    const set = new Set(ids);
    if (agentId !== "") {
      set.add(agentId);
    }
    return set;
  }, [agentId, ids]);

  const full = ids.length >= MAX_DELEGATES;
  const { onChange } = field;

  const add = useCallback(
    (agent: AgentChoice) => {
      setOpen(false);
      if (hidden.has(agent.id) || ids.length >= MAX_DELEGATES) {
        return;
      }
      setPicked((current) => {
        const next = new Map(current);
        next.set(agent.id, {
          id: agent.id,
          name: agent.name,
          icon: agent.icon,
          accent: agent.accent,
          enabled: true,
          triggerMode: "Chat",
        });
        return next;
      });
      onChange([...ids, agent.id]);
    },
    [hidden, ids, onChange],
  );

  const remove = useCallback(
    (id: string) => onChange(ids.filter((current) => current !== id)),
    [ids, onChange],
  );

  const summary =
    ids.length === 0
      ? t("Asks nobody. It works only with its own tools.")
      : t("{0} of {1} agents", ids.length, MAX_DELEGATES);

  return (
    <div className="flex flex-col gap-1">
      <div
        className={cn(
          "border-border bg-card flex flex-col rounded-lg border",
          fieldState.error && "border-danger-border",
        )}
      >
        <div className="flex items-center justify-between gap-3 px-3 py-2">
          <p className="text-muted-foreground min-w-0 truncate text-xs">{summary}</p>
          <Popover open={open} onOpenChange={setOpen}>
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="inline-flex">
                    <PopoverTrigger
                      disabled={full}
                      render={<Button type="button" size="xs" variant="outline" />}
                    >
                      <PlusIcon className="size-3" />
                      {t("Add an agent")}
                    </PopoverTrigger>
                  </span>
                }
              />
              <TooltipContent>
                {full
                  ? t("An agent can ask at most {0} others", MAX_DELEGATES)
                  : t("Only agents people talk to can be asked")}
              </TooltipContent>
            </Tooltip>
            <PopoverContent
              align="end"
              side="bottom"
              sideOffset={6}
              className="ui-lift-float w-80 gap-0 overflow-hidden p-0"
            >
              {open && (
                <AgentPickerList
                  selectedId={null}
                  recentIds={NO_RECENT}
                  hiddenIds={hidden}
                  onSelect={add}
                  source="organization"
                  emptyMessage={t("No other agents people talk to are left to add.")}
                />
              )}
            </PopoverContent>
          </Popover>
        </div>

        {ids.length > 0 && (
          <ol aria-label={t("Agents it can ask")} className="border-border border-t">
            {ids.map((id, index) => (
              <DelegateRow
                key={id}
                index={index}
                delegate={known.get(id) ?? null}
                onRemove={() => remove(id)}
              />
            ))}
          </ol>
        )}
      </div>
      {fieldState.error?.message && (
        <p role="alert" className="text-danger text-xs">
          {fieldState.error.message}
        </p>
      )}
    </div>
  );
}

/**
 * One agent on the list. It registers `delegateIds.{index}` so an error the
 * server returns for that agent is shown on its row; the value itself is
 * owned by the list above.
 */
function DelegateRow({
  index,
  delegate,
  onRemove,
}: {
  index: number;
  delegate: DelegateSummary | null;
  onRemove: () => void;
}) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const { fieldState } = useController({ control, name: `delegateIds.${index}` });
  const name = delegate?.name ?? t("An agent that could not be found");
  const error = fieldState.error?.message;

  return (
    <li className="border-border flex min-w-0 flex-col gap-0.5 border-b px-3 py-2 last:border-b-0">
      <div className="flex min-w-0 items-center gap-2.5">
        <AgentTile
          agent={delegate}
          size="sm"
          className={cn(delegate && !delegate.enabled && "opacity-60 grayscale")}
        />
        <span
          className={cn(
            "min-w-0 flex-1 truncate text-sm",
            delegate?.enabled === false ? "text-foreground-muted" : "text-foreground",
          )}
        >
          {name}
        </span>
        {delegate && !delegate.enabled && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Badge variant="neutral" appearance="outline" className="shrink-0">
                  {t("Disabled")}
                </Badge>
              }
            />
            <TooltipContent>
              {t("It stays on the list, and is refused when asked until it is enabled again")}
            </TooltipContent>
          </Tooltip>
        )}
        <Button
          type="button"
          size="icon-xs"
          variant="ghost"
          aria-label={t("Remove {0}", name)}
          className="text-muted-foreground hover:text-foreground shrink-0"
          onClick={onRemove}
        >
          <XIcon className="size-3" />
        </Button>
      </div>
      {error && (
        <p role="alert" className="text-danger pl-8.5 text-xs">
          {error}
        </p>
      )}
    </li>
  );
}
