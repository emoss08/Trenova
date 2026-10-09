import { useUserTimezone } from "@/hooks/use-user-timezone";
import { DeskDropOverlay } from "@/components/desk-chat/composer/desk-uploads";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import {
  DeskGreeting,
  DeskHomeComposer,
  useDeskHomeAsk,
  type DeskStartExtras,
} from "@/components/desk-chat/desk-home-ask";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo, useState } from "react";
import { threadAge } from "./assistant-history";

/** How many recent conversations the compact home lists. */
const RECENT_LIMIT = 3;

const nowInSeconds = () => Math.floor(Date.now() / 1000);

function touchedAt(thread: AssistantThread): number {
  return thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
}

export type AssistantHomeProps = {
  /** The full-screen layout draws the Desk's front page; the others a short list and the box. */
  full: boolean;
  agents: readonly AgentChoice[];
  agentsById: ReadonlyMap<string, AgentChoice>;
  threads: readonly AssistantThread[];
  isLoading: boolean;
  isStarting: boolean;
  onStart: (agentId: string, question?: string, extras?: DeskStartExtras) => void;
  onOpen: (thread: AssistantThread) => void;
  onShowAll: () => void;
};

/**
 * Where the assistant opens when no conversation is: what it does, in a line,
 * and the box a conversation starts from. The box is the Desk's home
 * composer, with its slow ring and the chosen agent's starter questions typed
 * out; the agent is chosen in it, not from a grid. In the corner and docked to
 * the side, the last three conversations sit above the box with a way to all
 * of them. Full screen, it is the Desk's front page: the date, the heading,
 * the line and the box, centred.
 */
export function AssistantHome({
  full,
  agents,
  agentsById,
  threads,
  isLoading,
  isStarting,
  onStart,
  onOpen,
  onShowAll,
}: AssistantHomeProps) {
  const t = useT();
  const timezone = useUserTimezone();
  const [now] = useState(nowInSeconds);
  const ask = useDeskHomeAsk({ agents, threads, isLoading, pageSource: "screen", onStart });
  const { drag } = ask;
  const recent = useMemo(
    () => [...threads].sort((a, b) => touchedAt(b) - touchedAt(a)).slice(0, RECENT_LIMIT),
    [threads],
  );
  const title = t("Ask about anything you can see");
  const line = t("Reads what's on your screen. Changes wait for your approval.");

  if (full) {
    const date = new Intl.DateTimeFormat(undefined, {
      weekday: "long",
      month: "long",
      day: "numeric",
      timeZone: timezone,
    }).format(new Date(now * 1000));

    return (
      <>
        <DeskDropOverlay show={drag.on} hot={drag.hot} count={drag.count} />
        <div className="dk-home">
          <div className="dk-home-in">
            <DeskGreeting date={date} title={title} line={line} pulse />
            <DeskHomeComposer ask={ask} isStarting={isStarting} />
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <DeskDropOverlay show={drag.on} hot={drag.hot} count={drag.count} />
      <div className="as-scroll">
        <div className="as-col">
          <div className="as-hm-h">
            <h2>{title}</h2>
            <p>{line}</p>
          </div>
          {recent.length > 0 && (
            <>
              <div className="as-rc-h">
                <span>{t("Recent")}</span>
                <Button
                  variant="bare"
                  size="bare"
                  className="text-dsk-subtle hover:text-dsk-fg"
                  onClick={onShowAll}
                >
                  {t("All conversations")}
                </Button>
              </div>
              {recent.map((thread, index) => (
                <Button
                  key={thread.id}
                  variant="bare"
                  size="bare"
                  className="as-rc-i group/rc -mx-2 flex w-[calc(100%+16px)] gap-2.5 rounded-lg px-2 py-1.75 text-left transition-colors duration-150 hover:bg-dsk-hover"
                  style={{ animationDelay: `${60 + index * 40}ms` }}
                  onClick={() => onOpen(thread)}
                >
                  <DeskAgentTile agent={agentsById.get(thread.agentDefinitionId)} size="xs" />
                  <span className="min-w-0 flex-1 truncate text-base text-dsk-fg2 group-hover/rc:text-dsk-fg">
                    {thread.title || t("Untitled conversation")}
                  </span>
                  <em className="font-mono text-xs text-dsk-faint not-italic">
                    {threadAge(thread, now)}
                  </em>
                </Button>
              ))}
            </>
          )}
        </div>
      </div>
      <div className="as-dock dk-dense">
        <DeskHomeComposer ask={ask} isStarting={isStarting} />
      </div>
    </>
  );
}
