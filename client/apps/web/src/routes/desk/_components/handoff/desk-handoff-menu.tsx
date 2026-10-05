import { appendToHistory, type ThreadHistory } from "@/components/assistant/thread-history";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useMemo, useReducer, useRef } from "react";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { useOutsideDismiss } from "@/components/desk-chat/use-outside-dismiss";
import { HANDOFF_MENU_CLOSED, handoffMenuReducer, handoffTargets } from "./handoff-state";

/**
 * The top bar's Hand off button and its menu. Picking an agent starts a
 * conversation with it, opened with a summary of this one, its pinned facts
 * and its pinned artifacts; the card the server leaves here says so and
 * offers the way across.
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
  const queryClient = useQueryClient();
  const root = useRef<HTMLSpanElement>(null);
  const [state, dispatch] = useReducer(handoffMenuReducer, HANDOFF_MENU_CLOSED);
  const close = useCallback(() => dispatch({ type: "close" }), []);
  useOutsideDismiss(root, state.open, close);

  const targets = useMemo(() => handoffTargets(agents, agent), [agent, agents]);

  const handoff = useApiMutation({
    mutationFn: (agentId: string) =>
      apiService.assistantService.handoffThread(threadId, agentId, facts),
    onSuccess: async (result) => {
      const card = result.message;
      if (card) {
        queryClient.setQueryData<ThreadHistory>(
          queries.assistant.messages(threadId).queryKey,
          (history) => appendToHistory(history, [card]),
        );
      }
      await queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey });
    },
    onSettled: () => dispatch({ type: "settled" }),
    resourceName: "Conversation",
  });

  return (
    <span className="dk-hom" ref={root}>
      <button
        type="button"
        className={cn("dk-ib", state.open && "dk-on")}
        title={t("Hand off to another agent")}
        aria-label={t("Hand off to another agent")}
        aria-expanded={state.open}
        aria-haspopup="menu"
        disabled={state.pendingAgentId !== null}
        onClick={() => dispatch({ type: "toggle" })}
      >
        <DeskIcon name="handoff" size={14} />
      </button>
      {state.open && (
        <div className="dk-hom-p" role="menu">
          <div className="dk-hom-h">
            <b>{t("Hand off to")}</b>
            <span>{t("Carries over a summary, pinned facts and pinned artifacts")}</span>
          </div>
          {targets.length === 0 ? (
            <p className="dk-hom-empty">{t("There is no other agent you can hand this to.")}</p>
          ) : (
            targets.map((target) => (
              <button
                key={target.id}
                type="button"
                role="menuitem"
                onClick={() => {
                  dispatch({ type: "pick", agentId: target.id });
                  handoff.mutate(target.id);
                }}
              >
                <DeskAgentTile agent={target} size="xs" />
                <span>
                  <b>{target.name}</b>
                  <em>{target.description}</em>
                </span>
              </button>
            ))
          )}
        </div>
      )}
    </span>
  );
}
