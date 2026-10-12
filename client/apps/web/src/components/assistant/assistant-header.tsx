import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import type { AssistantLayout } from "@/lib/assistant-dock";
import type { AssistantThread } from "@/types/assistant";
import { Expand01Icon, Minimize01Icon } from "@trenova/shared/components/icons";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { AssistantIconButton } from "./assistant-icon-button";
import { AssistantPlacementMenu } from "./assistant-placement-menu";

/** What the panel is showing under its header. */
export type AssistantView = "home" | "thread" | "history";

export type AssistantThreadActions = {
  onDelete: (thread: AssistantThread) => void;
  onDownloadTranscript: (thread: AssistantThread) => void;
  /** Continues the conversation at the Desk, with room for what it produced. */
  onOpenInDesk: (thread: AssistantThread) => void;
};

/** One of the header's icon buttons, its name said aloud and on hover. */
function HeaderButton({
  label,
  onClick,
  pressed,
  children,
}: {
  label: string;
  onClick: () => void;
  pressed?: boolean;
  children: ReactNode;
}) {
  return (
    <AssistantIconButton title={label} aria-label={label} aria-pressed={pressed} onClick={onClick}>
      {children}
    </AssistantIconButton>
  );
}

function Icon({ name, size = 15 }: { name: DeskIconName; size?: number }) {
  return <DeskIcon name={name} size={size} />;
}

/** The open conversation's own actions, behind ⋯ beside its title. */
function ThreadMenu({
  thread,
  actions,
}: {
  thread: AssistantThread;
  actions: AssistantThreadActions;
}) {
  const t = useT();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <AssistantIconButton
            title={t("Conversation actions")}
            aria-label={t("Conversation actions")}
          />
        }
      >
        <Icon name="more" size={14} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-52">
        <DropdownMenuItem title={t("Open in Desk")} onClick={() => actions.onOpenInDesk(thread)} />
        <DropdownMenuItem
          title={t("Download transcript")}
          onClick={() => actions.onDownloadTranscript(thread)}
        />
        <DropdownMenuSeparator />
        <DropdownMenuItem
          title={t("Delete conversation")}
          color="danger"
          onClick={() => actions.onDelete(thread)}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export type AssistantHeaderProps = {
  layout: Exclude<AssistantLayout, "full">;
  view: AssistantView;
  thread: AssistantThread | null;
  actions: AssistantThreadActions;
  onBack: () => void;
  onToggleHistory: () => void;
  onNew: () => void;
  onLayout: (layout: AssistantLayout) => void;
  onClose: () => void;
};

/**
 * The compact and side header: back out of a conversation or the history, the
 * conversation's title with its actions behind ⋯, then the panel's controls —
 * the conversations, a new one, docking to the side or floating, full screen
 * and close.
 */
export function AssistantHeader({
  layout,
  view,
  thread,
  actions,
  onBack,
  onToggleHistory,
  onNew,
  onLayout,
  onClose,
}: AssistantHeaderProps) {
  const t = useT();
  const title =
    view === "thread" && thread
      ? thread.title || t("Untitled conversation")
      : view === "history"
        ? t("Conversations")
        : t("Assistant");

  return (
    <div className="as-h">
      <div className="as-h-t">
        {view !== "home" && (
          <span className="as-back">
            <HeaderButton label={t("Back")} onClick={onBack}>
              <Icon name="chevL" />
            </HeaderButton>
          </span>
        )}
        <b key={title}>{title}</b>
        {view === "thread" && thread && <ThreadMenu thread={thread} actions={actions} />}
      </div>
      <HeaderButton
        label={t("Conversations")}
        pressed={view === "history"}
        onClick={onToggleHistory}
      >
        <Icon name="clock" />
      </HeaderButton>
      <HeaderButton label={t("New conversation")} onClick={onNew}>
        <Icon name="plus" />
      </HeaderButton>
      <HeaderButton
        label={layout === "side" ? t("Float") : t("Dock to side")}
        pressed={layout === "side"}
        onClick={() => onLayout(layout === "side" ? "compact" : "side")}
      >
        <Icon name="panel" />
      </HeaderButton>
      <AssistantPlacementMenu />
      <HeaderButton label={t("Full screen")} onClick={() => onLayout("full")}>
        <Expand01Icon className="size-3.5" />
      </HeaderButton>
      <HeaderButton label={t("Close · Esc")} onClick={onClose}>
        <Icon name="x" />
      </HeaderButton>
    </div>
  );
}

export type AssistantFullHeaderProps = {
  sidebarOpen: boolean;
  thread: AssistantThread | null;
  actions: AssistantThreadActions;
  onToggleSidebar: () => void;
  onNew: () => void;
  onShrink: () => void;
  onClose: () => void;
};

/**
 * The full-screen layout's bar over the conversation: the sidebar toggle (and
 * a new conversation while the sidebar is folded away), the title in the
 * middle, then shrink and close.
 */
export function AssistantFullHeader({
  sidebarOpen,
  thread,
  actions,
  onToggleSidebar,
  onNew,
  onShrink,
  onClose,
}: AssistantFullHeaderProps) {
  const t = useT();

  return (
    <div className="as-fh">
      <HeaderButton
        label={sidebarOpen ? t("Hide conversations · ⌘\\") : t("Show conversations · ⌘\\")}
        pressed={!sidebarOpen}
        onClick={onToggleSidebar}
      >
        <Icon name="rail" />
      </HeaderButton>
      {!sidebarOpen && (
        <HeaderButton label={t("New conversation")} onClick={onNew}>
          <Icon name="plus" />
        </HeaderButton>
      )}
      <div className="as-fh-t">
        {thread && (
          <>
            <b key={thread.id}>{thread.title || t("Untitled conversation")}</b>
            <ThreadMenu thread={thread} actions={actions} />
          </>
        )}
      </div>
      <HeaderButton label={t("Shrink")} onClick={onShrink}>
        <Minimize01Icon className="size-3.5" />
      </HeaderButton>
      <HeaderButton label={t("Close · Esc")} onClick={onClose}>
        <Icon name="x" />
      </HeaderButton>
    </div>
  );
}
