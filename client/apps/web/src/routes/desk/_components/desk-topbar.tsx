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
import { DeskSituation } from "./desk-situation";
import { lensFromParam, lensLabel } from "./attention/lens";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { useSearchParams } from "react-router";

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
  onSetPinned,
  onDownload,
  onRename,
  onOpenRail,
  onUnfoldRail,
  canWatch,
  needsYou,
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
  /** Pins or unpins the open conversation. */
  onSetPinned: (pinned: boolean) => void;
  onDownload: () => void;
  /** Renames the open conversation; the name in the bar is where it is edited. */
  onRename: (title: string) => void;
  /** Opens the rail over the page, where it is folded away on a phone. */
  onOpenRail?: () => void;
  /** Opens the rail again beside the page, given only while it is folded away. */
  onUnfoldRail?: () => void;
  /** Whether the person may read the Watchtower, which the live figures count. */
  canWatch: boolean;
  /** Open items waiting on people, for the situation line. */
  needsYou: number;
}) {
  const t = useT();
  const [search] = useSearchParams();
  const subject = useDeskMarginStore((state) => state.pageSubject);
  const marginOpen = useDeskMarginStore((state) => state.margin !== null);
  const toggleMargin = useDeskMarginStore((state) => state.toggleMargin);
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
      {onUnfoldRail && (
        <DeskTip label={t("Show sidebar")}>
          <Button
            variant="quiet"
            size="icon-sm"
            className={cn(deskIconClass, "-ml-1.5 max-[720px]:hidden")}
            aria-label={t("Show sidebar")}
            onClick={onUnfoldRail}
          >
            <DeskIcon name="rail" size={15} />
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
        ) : place === "needs" ? (
          <>
            <span className="dk-ttl-a">{t("Needs you")}</span>
            <span className="dk-ttl-sl">/</span>
            <b>{lensLabel(t, lensFromParam(search.get("lens")))}</b>
          </>
        ) : place === "case" && subject ? (
          <>
            <span className="dk-ttl-a">{t("Case")}</span>
            <span className="dk-ttl-sl">/</span>
            <b>{subject.label}</b>
          </>
        ) : (
          <b>
            {place === "decisions"
              ? t("Decisions")
              : place === "case"
                ? t("Case")
                : place === "triage"
                    ? t("Triage")
                    : place === "watchtower"
                      ? t("Watchtower")
                      : place === "scheduled"
                        ? t("Scheduled")
                        : place === "memory"
                          ? t("Memory")
                          : t("Today")}
          </b>
        )}
      </div>
      <span className="dk-sp" />
      {!inThread && <DeskSituation canWatch={canWatch} needsYou={needsYou} />}
      <button
        type="button"
        className="dk-w-kbtn ui-focus-ring"
        aria-label={t("Search or run a command")}
        onClick={() => useCommandPaletteStore.getState().setOpen(true)}
      >
        <DeskIcon name="search" size={14} />
        <span className="dk-w-kbtn-l">{t("Search or do")}</span>
        <span className="dk-w-kbd">{formatShortcut("K")}</span>
      </button>
      {!inThread && (
        <button
          type="button"
          className="dk-w-kbtn ui-focus-ring"
          aria-pressed={marginOpen}
          aria-label={t("Ask Nova")}
          onClick={toggleMargin}
        >
          <NovaBadge size="xs" />
          <span className="dk-w-kbtn-l">{t("Ask")}</span>
          <span className="dk-w-kbd">{formatShortcut("J")}</span>
        </button>
      )}
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
                onClick={() => onSetPinned(!thread.pinned)}
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
