import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DeskAgentTile } from "./desk-agent-tile";
import { DeskIcon } from "./desk-icons";
import type { DeskPlace } from "./desk-rail";

/**
 * The strip across the top of the Desk. In a conversation it names the agent
 * and the conversation and holds the conversation's own controls; anywhere
 * else it names the place.
 */
export function DeskTopBar({
  place,
  thread,
  agent,
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
  agent: AgentChoice | null;
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
            <span className="dk-ttl-a">{agent?.name ?? t("Agent unavailable")}</span>
            <span className="dk-ttl-sl">/</span>
            <b>{thread.title || t("Untitled conversation")}</b>
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
