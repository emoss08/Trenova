import { DeleteConversationCard } from "@/components/assistant/delete-conversation-card";
import { useDeleteConversation } from "@/components/assistant/use-delete-conversation";
import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { DeskTip } from "@/components/desk-chat/desk-tip";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import type { AssistantThread } from "@/types/assistant";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { isMacPlatform } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ReactNode } from "react";
import { useSearchParams } from "react-router";
import { preloadDeskWorkspace } from "../artifacts/desk-workspace-lazy";
import { lensFromParam, lensLabel } from "../attention/lens";
import { DeskCaseMenu } from "../case/desk-case-menu";
import { DeskThreadTitle } from "../desk-thread-title";
import { DeskHandoffMenu } from "../handoff/desk-handoff-menu";
import type { DeskPlace } from "../rail/desk-rail";
import { DeskSituation } from "./desk-situation";
import "./top.css";

/** A key hint, as the prototype spells it: the modifier, a space, the key. */
function modKey(key: string): string {
  // i18n-ignore: key names as printed on the keyboard
  return `${isMacPlatform() ? "⌘" : "Ctrl"} ${key}`;
}

/** Reaching the Workspace button starts reading its code, so a click opens it at once. */
function preloadWorkspace() {
  preloadDeskWorkspace().catch(() => undefined);
}

/** A square icon button of the bar, the prototype's `.ib`. */
function TopIconButton({
  label,
  className,
  onClick,
  children,
}: {
  label: string;
  className?: string;
  onClick?: () => void;
  children: ReactNode;
}) {
  return (
    <DeskTip label={label}>
      <button
        type="button"
        className={cn("dk-tb-ib ui-focus-ring", className)}
        aria-label={label}
        onClick={onClick}
      >
        {children}
      </button>
    </DeskTip>
  );
}

/** Where the person is, in the prototype's crumbs: the place in quiet text, a slash, the page. */
function Crumbs({ lead, page }: { lead?: ReactNode; page: ReactNode }) {
  return (
    <>
      {lead !== undefined && (
        <>
          {lead}
          <span className="dk-tb-sl">/</span>
        </>
      )}
      <b>{page}</b>
    </>
  );
}

function placeName(t: TranslateFn, place: DeskPlace): string {
  switch (place) {
    case "triage":
      return t("Triage");
    case "decisions":
      return t("Decisions");
    case "scheduled":
      return t("Scheduled");
    case "memory":
      return t("Memory");
    default:
      return t("Today");
  }
}

export type DeskTopProps = {
  place: DeskPlace;
  thread: AssistantThread | null;
  /** The conversation's agent, or on an agent's page that agent. */
  agent: AgentChoice | null;
  /** Every agent the person may use, for handing the conversation to one. */
  agents: readonly AgentChoice[];
  onOpenAgent: () => void;
  workspaceOpen: boolean;
  artifactCount: number;
  /** Something landed in the workspace while it was folded away. */
  newArtifact: boolean;
  /** A change in this conversation waits for the person. */
  pending: boolean;
  onToggleWorkspace: () => void;
  onSetPinned: (pinned: boolean) => void;
  onDownload: () => void;
  onRename: (title: string) => void;
  onDelete: (thread: AssistantThread) => void;
  /** Opens the rail over the page on a phone. */
  onOpenRail: () => void;
  /** Opens the rail again beside the page, given only while it is folded away. */
  onUnfoldRail?: () => void;
  /** Whether the person may read the Watchtower, which the live figures count. */
  canWatch: boolean;
  /** Open items waiting on people, for the situation line. */
  needsYou: number;
};

/**
 * The bar across the top of the Desk, as the prototype's `TopBar` draws it:
 * where the person is, the operation now in one joined strip, **Search or
 * do** and **Ask**. In a conversation that fills the room it names the agent
 * and the conversation and holds the conversation's own controls.
 */
export function DeskTop({
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
  onDelete,
  onOpenRail,
  onUnfoldRail,
  canWatch,
  needsYou,
}: DeskTopProps) {
  const t = useT();
  const [search] = useSearchParams();
  const subject = useDeskMarginStore((state) => state.pageSubject);
  const marginOpen = useDeskMarginStore((state) => state.margin !== null);
  const toggleMargin = useDeskMarginStore((state) => state.toggleMargin);
  const deletion = useDeleteConversation(onDelete);
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const inThread = place === "thread" && thread !== null;

  const crumbs = inThread ? (
    <>
      <span className="dk-tb-agc">
        <DeskAgentTile agent={agent} size="xs" />
        {agent ? (
          <DeskTip label={t("What this agent can do")}>
            <button type="button" className="dk-tb-agn ui-focus-ring" onClick={onOpenAgent}>
              {agent.name}
            </button>
          </DeskTip>
        ) : (
          t("Agent unavailable")
        )}
      </span>
      <span className="dk-tb-sl">/</span>
      <DeskThreadTitle key={thread.id} thread={thread} onRename={onRename} />
    </>
  ) : place === "agent" ? (
    <Crumbs
      lead={
        <span className="dk-tb-agc">
          <DeskAgentTile agent={agent} size="xs" />
          {agent?.name ?? t("Agent")}
        </span>
      }
      page={t("What it can do")}
    />
  ) : place === "needs" ? (
    <Crumbs
      lead={<span className="dk-tb-c">{t("Needs you")}</span>}
      page={lensLabel(t, lensFromParam(search.get("lens")))}
    />
  ) : place === "case" ? (
    <Crumbs lead={<span className="dk-tb-c">{t("Case")}</span>} page={subject?.label ?? ""} />
  ) : place === "handled" ? (
    <Crumbs
      lead={<span className="dk-tb-c">{t("Nova handled")}</span>}
      page={t("What an agent did")}
    />
  ) : (
    <Crumbs page={placeName(t, place)} />
  );

  return (
    <header className={cn("dk-tb", place === "needs" && "dk-ruled")}>
      <TopIconButton label={t("Open the rail")} className="dk-tb-phone" onClick={onOpenRail}>
        <DeskIcon name="panel" size={14} />
      </TopIconButton>
      {onUnfoldRail && (
        <TopIconButton label={t("Show the rail")} className="dk-tb-unfold" onClick={onUnfoldRail}>
          <DeskIcon name="rail" size={14} />
        </TopIconButton>
      )}
      <div className="dk-tb-ttl">{crumbs}</div>

      {!inThread && <DeskSituation canWatch={canWatch} needsYou={needsYou} />}
      <button
        type="button"
        className="dk-tb-kbtn ui-focus-ring"
        aria-label={t("Search or run a command")}
        onClick={() => useCommandPaletteStore.getState().setOpen(true)}
      >
        <DeskIcon name="search" size={14} />
        <span className="dk-tb-lbl">{t("Search or do")}</span>
        <span className="dk-tb-kbd">{modKey("K")}</span>
      </button>
      {!inThread && (
        <button
          type="button"
          className="dk-tb-kbtn ui-focus-ring"
          aria-pressed={marginOpen}
          aria-label={t("Ask Nova")}
          onClick={toggleMargin}
        >
          <NovaBadge size="xs" />
          <span className="dk-tb-lbl">{t("Ask")}</span>
          <span className="dk-tb-kbd">{modKey("J")}</span>
        </button>
      )}

      {inThread && (
        <>
          <DeskCaseMenu thread={thread} timezone={timezone} />
          <DeskHandoffMenu threadId={thread.id} agent={agent} agents={agents} />
          <DropdownMenu>
            <DeskTip label={t("More")}>
              <DropdownMenuTrigger
                render={<button type="button" className="dk-tb-ib ui-focus-ring" />}
                aria-label={t("More actions")}
              >
                <DeskIcon name="more" size={14} />
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
              <DropdownMenuItem
                title={t("Delete conversation")}
                startContent={<DeskIcon name="trash" size={14} />}
                onClick={() => deletion.ask(thread)}
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
            <button
              type="button"
              className="dk-tb-kbtn ui-focus-ring"
              aria-pressed={workspaceOpen}
              aria-label={t("Workspace")}
              onPointerEnter={preloadWorkspace}
              onFocus={preloadWorkspace}
              onClick={onToggleWorkspace}
            >
              <DeskIcon name="panel" size={14} />
              <span className="dk-tb-lbl">{t("Workspace")}</span>
              <span className="dk-tb-n">{artifactCount}</span>
              {!workspaceOpen && (newArtifact || pending) && (
                <span className={cn("dk-tb-dot", pending ? "dk-warn" : "dk-blue")} />
              )}
            </button>
          </DeskTip>
        </>
      )}

      <DeleteConversationCard
        thread={deletion.asking}
        onConfirm={deletion.confirm}
        onClose={deletion.cancel}
      />
    </header>
  );
}
