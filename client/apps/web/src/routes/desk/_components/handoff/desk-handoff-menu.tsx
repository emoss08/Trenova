import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useReducer } from "react";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { deskPopoverClass } from "@/components/desk-chat/desk-popover";
import { DeskTip } from "@/components/desk-chat/desk-tip";
import { HANDOFF_MENU_CLOSED, handoffMenuReducer, handoffTargets } from "./handoff-state";
import { useHandOffThread } from "./use-hand-off-thread";
import { useHandoffSuggestion } from "./use-handoff-suggestion";

/**
 * The top bar's Hand off button and its menu. Picking an agent starts a
 * conversation with it, opened with a summary of this one, its pinned facts
 * and its pinned artifacts; the card the server leaves here says so and
 * offers the way across. When the conversation's agent last said another of
 * the person's agents holds what it could not do, those agents lead the menu
 * and are marked, so the person does not have to match a name in the reply.
 */
export function DeskHandoffMenu({
  threadId,
  agent,
  agents,
  facts = [],
}: {
  threadId: string;
  agent: AgentChoice | null;
  agents: readonly AgentChoice[];
  /** The conversation's pinned facts, carried over as the person sees them. */
  facts?: readonly string[];
}) {
  const t = useT();
  const [state, dispatch] = useReducer(handoffMenuReducer, HANDOFF_MENU_CLOSED);

  // The reply being written and the thread's saved history, narrowed to the
  // suggestion: the menu re-renders only when the suggestion changes, not on
  // every message or streamed word. The conversation reads the history and
  // follows the reply; the menu only watches them, never fetches.
  const suggested = useHandoffSuggestion(threadId);
  const targets = useMemo(
    () => handoffTargets(agents, agent, suggested),
    [agent, agents, suggested],
  );
  const suggestedIds = useMemo(() => new Set(suggested), [suggested]);

  const handoff = useHandOffThread(threadId, facts);

  return (
    <Popover
      open={state.open}
      onOpenChange={(open) => dispatch(open ? { type: "toggle" } : { type: "close" })}
    >
      <DeskTip label={t("Hand off to another agent")}>
        <PopoverTrigger
          disabled={state.pendingAgentId !== null}
          render={
            <Button
              variant="quiet"
              size="icon-sm"
              className={cn(
                "size-7.5 rounded-lg text-dsk-subtle transition-colors duration-140 hover:bg-dsk-hover hover:text-dsk-fg",
                state.open && "bg-dsk-hover text-dsk-fg",
              )}
              aria-label={t("Hand off to another agent")}
              aria-haspopup="menu"
            />
          }
        >
          <DeskIcon name="handoff" size={14} />
        </PopoverTrigger>
      </DeskTip>
      <PopoverContent
        align="end"
        sideOffset={6}
        className={cn(deskPopoverClass, "dk-hom-p w-75 p-1")}
      >
        <div className="dk-hom-h">
          <b>{t("Hand off to")}</b>
          <span>{t("Carries over a summary, pinned facts and pinned artifacts")}</span>
        </div>
        {targets.length === 0 ? (
          <p className="dk-hom-empty">{t("There is no other agent you can hand this to.")}</p>
        ) : (
          <ScrollArea viewportClassName="max-h-[min(60vh,420px)]" maskVariant="raised">
            <div role="menu">
              {targets.map((target) => (
                <Button
                  key={target.id}
                  variant="bare"
                  size="bare"
                  role="menuitem"
                  className="flex w-full gap-2.5 rounded-lg px-2 py-1.75 text-left hover:bg-dsk-hover"
                  onClick={() => {
                    dispatch({ type: "pick", agentId: target.id });
                    handoff.mutate(target.id, { onSettled: () => dispatch({ type: "settled" }) });
                  }}
                >
                  <DeskAgentTile agent={target} size="xs" />
                  <span className="flex min-w-0 flex-col">
                    <b className="text-sm font-medium">{target.name}</b>
                    {suggestedIds.has(target.id) && (
                      <span className="flex items-center gap-1 text-xs text-dsk-fg2">
                        <AssistMark className="size-3 shrink-0" aria-hidden />
                        {t("Holds what this conversation needs")}
                      </span>
                    )}
                    <em className="truncate text-xs text-dsk-subtle not-italic">
                      {target.description}
                    </em>
                  </span>
                </Button>
              ))}
            </div>
          </ScrollArea>
        )}
      </PopoverContent>
    </Popover>
  );
}
