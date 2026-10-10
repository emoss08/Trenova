import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { DeskTip } from "@/components/desk-chat/desk-tip";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import type { DeskPlace } from "./desk-rail";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { DeskCaseMenu } from "./case/desk-case-menu";
import { DeskHandoffMenu } from "./handoff/desk-handoff-menu";
import { DeskThreadTitle } from "./desk-thread-title";
import { preloadDeskWorkspace } from "./artifacts/desk-workspace-lazy";

/** Reaching the Workspace button starts reading its code, so a click opens it at once. */
function preloadWorkspace() {
  preloadDeskWorkspace().catch(() => undefined);
}
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";

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
  onRename,
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
  /** Renames the open conversation; the name in the bar is where it is edited. */
  onRename: (title: string) => void;
  /** Opens the rail over the page, where it is folded away on a phone. */
  onOpenRail?: () => void;
}) {
  const t = useT();
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const inThread = place === "thread" && thread !== null;

  return (
    <header className="dk-top">
      {onOpenRail && (
        <DeskTip label={t("Menu")}>
          <Button
            variant="quiet"
            size="icon-sm"
            className={cn(deskIconClass, "hidden max-[720px]:mr-1 max-[720px]:inline-flex")}
            aria-label={t("Menu")}
            onClick={onOpenRail}
          >
            <DeskIcon name="menu" size={16} />
          </Button>
        </DeskTip>
      )}
      <div className="dk-ttl">
        {inThread ? (
          <>
            <DeskAgentTile agent={agent} size="xs" className="dk-ttl-at" />
            {agent ? (
              <DeskTip label={t("What this agent can do")}>
                <Button
                  variant="bare"
                  size="bare"
                  className="-mx-1 rounded-md px-1 py-0.5 whitespace-nowrap text-dsk-subtle transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg max-[720px]:hidden"
                  onClick={onOpenAgent}
                >
                  {agent.name}
                </Button>
              </DeskTip>
            ) : (
              <span className="dk-ttl-a">{t("Agent unavailable")}</span>
            )}
            <span className="dk-ttl-sl">/</span>
            <DeskThreadTitle key={thread.id} thread={thread} onRename={onRename} />
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
          <DeskCaseMenu thread={thread} timezone={timezone} />
          <DeskHandoffMenu threadId={thread.id} agent={agent} agents={agents} />
          <DropdownMenu>
            <DeskTip label={t("More")}>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="quiet"
                    size="icon-sm"
                    className={deskIconClass}
                    aria-label={t("More actions")}
                  />
                }
              >
                <DeskIcon name="more" size={15} />
              </DropdownMenuTrigger>
            </DeskTip>
            <DropdownMenuContent align="end" sideOffset={6} className="w-56">
              <DropdownMenuItem
                title={thread.pinned ? t("Unpin conversation") : t("Pin conversation")}
                startContent={<DeskIcon name="pin" size={14} />}
                onClick={onTogglePin}
              />
              <DropdownMenuItem
                title={t("Download transcript")}
                startContent={<DeskIcon name="download" size={14} />}
                onClick={onDownload}
              />
              {agent && (
                <DropdownMenuItem
                  title={t("What this agent can do")}
                  startContent={<DeskIcon name="shield" size={14} />}
                  onClick={onOpenAgent}
                />
              )}
            </DropdownMenuContent>
          </DropdownMenu>
          <DeskTip label={workspaceOpen ? t("Hide the workspace") : t("Show the workspace")}>
          <Button
            variant="bare"
            size="bare"
            className={cn(
              "dk-wsb flex h-8 gap-2 rounded-lg pr-3 pl-2.5 text-sm text-dsk-muted transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg max-[720px]:px-2",
              workspaceOpen && "bg-dsk-hover text-dsk-fg",
            )}
            aria-pressed={workspaceOpen}
            aria-label={t("Workspace")}
            onPointerEnter={preloadWorkspace}
            onFocus={preloadWorkspace}
            onClick={onToggleWorkspace}
          >
            <DeskIcon name="panel" size={15} />
            <span className="max-[720px]:hidden">{t("Workspace")}</span>
            <span className="dk-n">{artifactCount}</span>
            {!workspaceOpen && (newArtifact || pending) && (
              <span className={cn("dk-dot", pending && "dk-dot-wait")} />
            )}
          </Button>
          </DeskTip>
        </>
      )}
    </header>
  );
}
