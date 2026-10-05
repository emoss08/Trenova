import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import type { DeskPlace } from "./desk-rail";
import { DeskHandoffMenu } from "./handoff/desk-handoff-menu";

/**
 * The strip across the top of the Desk. In a conversation it names the agent
 * and the conversation and holds the conversation's own controls; anywhere
 * else it names the place.
 */
export function DeskTopBar({
  place,
  thread,
  agent,
  agents,
  onOpenAgent,
  workspaceOpen,
  artifactCount,
  newArtifact,
  pending,
  onToggleWorkspace,
  onTogglePin,
  onDownload,
  onOpenRail,
}: {
  place: DeskPlace;
  thread: AssistantThread | null;
  /** The conversation's agent, or on an agent's page that agent. */
  agent: AgentChoice | null;
  /** Every agent the person may use, for handing the conversation to one. */
  agents: readonly AgentChoice[];
  /** Opens the page saying what the conversation's agent can do. */
  onOpenAgent: () => void;
  workspaceOpen: boolean;
  artifactCount: number;
  /** Something landed in the workspace while it was folded away. */
  newArtifact: boolean;
  /** A change in this conversation waits for the person. */
  pending: boolean;
  onToggleWorkspace: () => void;
  onTogglePin: () => void;
  onDownload: () => void;
  /** Opens the rail over the page, where it is folded away on a phone. */
  onOpenRail?: () => void;
}) {
  const t = useT();
  const inThread = place === "thread" && thread !== null;

  return (
    <header className="dk-top">
      {onOpenRail && (
        <button
          type="button"
          className="dk-ib dk-top-menu"
          title={t("Menu")}
          aria-label={t("Menu")}
          onClick={onOpenRail}
        >
          <DeskIcon name="menu" size={16} />
        </button>
      )}
      <div className="dk-ttl">
        {inThread ? (
          <>
            <DeskAgentTile agent={agent} size="xs" className="dk-ttl-at" />
            {agent ? (
              <button
                type="button"
                className="dk-ttl-a dk-ttl-ag"
                title={t("What this agent can do")}
                onClick={onOpenAgent}
              >
                {agent.name}
              </button>
            ) : (
              <span className="dk-ttl-a">{t("Agent unavailable")}</span>
            )}
            <span className="dk-ttl-sl">/</span>
            <b>{thread.title || t("Untitled conversation")}</b>
          </>
        ) : place === "agent" ? (
          <>
            <DeskAgentTile agent={agent} size="xs" className="dk-ttl-at" />
            <b>{agent?.name ?? t("Agent")}</b>
            <span className="dk-ttl-sl">/</span>
            <span className="dk-ttl-a">{t("What it can do")}</span>
          </>
        ) : (
          <b>
            {place === "decisions"
              ? t("Decisions")
              : place === "watchtower"
                ? t("Watchtower")
                : place === "memory"
                  ? t("Memory")
                  : t("Today")}
          </b>
        )}
      </div>
      <span className="dk-sp" />
      {inThread && (
        <>
          <button
            type="button"
            className="dk-ib"
            title={t("Download transcript")}
            aria-label={t("Download transcript")}
            onClick={onDownload}
          >
            <DeskIcon name="download" size={14} />
          </button>
          <button
            type="button"
            className={cn("dk-ib", thread.pinned && "dk-on")}
            title={thread.pinned ? t("Unpin") : t("Pin")}
            aria-label={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
            aria-pressed={thread.pinned}
            onClick={onTogglePin}
          >
            <DeskIcon name="pin" size={14} />
          </button>
          <DeskHandoffMenu threadId={thread.id} agent={agent} agents={agents} />
          {agent && (
            <button
              type="button"
              className="dk-ib"
              title={t("What this agent can do")}
              aria-label={t("What this agent can do")}
              onClick={onOpenAgent}
            >
              <DeskIcon name="shield" size={14} />
            </button>
          )}
          <button
            type="button"
            className={cn("dk-wsb", workspaceOpen && "dk-on")}
            aria-pressed={workspaceOpen}
            aria-label={t("Workspace")}
            onClick={onToggleWorkspace}
          >
            <DeskIcon name="panel" size={15} />
            <span className="dk-wsb-l">{t("Workspace")}</span>
            <span className="dk-n">{artifactCount}</span>
            {!workspaceOpen && (newArtifact || pending) && (
              <span className={cn("dk-dot", pending && "dk-dot-wait")} />
            )}
          </button>
        </>
      )}
    </header>
  );
}
