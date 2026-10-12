import { useUserTimezone } from "@/hooks/use-user-timezone";
import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { downloadEml } from "@/lib/triage/eml";
import { isSnoozed } from "@/lib/triage/message-state";
import { useTriageStore } from "@/stores/triage-store";
import type { ContextMenuEntry } from "@trenova/shared/components/context-menu/menu-spec";
import {
  ArchiveIcon,
  ArrowDownIcon,
  ArrowUpIcon,
  ClockIcon,
  Copy01Icon,
  CornerDownLeftIcon,
  DotsHorizontalIcon,
  Download01Icon,
  LayoutRightIcon,
  Link01Icon,
  User01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ReactElement } from "react";
import { USER_FORMATTERS, snoozeText } from "../action-copy";
import { assigneeEntries, eventSnoozeEntries, snoozeTimeEntries } from "../menus";
import { PersonAvatar } from "../person-avatar";
import { useMessagePresence } from "../use-triage-presence";
import { useTriageActions } from "../use-triage-actions";
import { useTriageAssignees, useTriageDocumentTypes } from "../use-triage-data";
import { openMenuFrom } from "./menu-anchor";

function ToolButton({
  label,
  shortcut,
  children,
}: {
  label: string;
  shortcut?: string;
  children: ReactElement;
}) {
  const t = useT();
  return (
    <Tooltip>
      <TooltipTrigger render={children} />
      <TooltipContent>
        {shortcut === undefined ? label : t("{0} ({1})", label, shortcut)}
      </TooltipContent>
    </Tooltip>
  );
}

/** Who else has the message open, and whether they are writing back. */
function PresenceChip({ messageId }: { messageId: string }) {
  const t = useT();
  const members = useMessagePresence(messageId);
  const first = members[0];
  if (first === undefined) {
    return null;
  }
  const firstName = first.name.split(/\s+/)[0] ?? first.name;
  const text =
    first.activity === "writing"
      ? t("{0} is writing a reply", firstName)
      : t("{0} is viewing", firstName);

  return (
    <span
      role="status"
      title={text}
      className="bg-surface-hover text-foreground-muted animate-rise mr-1 inline-flex h-7 items-center gap-1.5 rounded-full pr-2.5 pl-1 text-sm"
    >
      <PersonAvatar name={first.name} present />
      <span className="truncate @max-[820px]/reader:sr-only">{text}</span>
      {members.length > 1 && (
        <span className="text-foreground-subtle font-mono text-xs tabular-nums">
          {t("+{0}", members.length - 1)}
        </span>
      )}
    </span>
  );
}

export type ReaderToolbarProps = {
  message: InboundMessageDetail;
  position: { index: number; total: number };
  onClose: () => void;
  onStep: (direction: 1 | -1) => void;
  /** Steps still open, shown on the panel toggle. */
  openSteps: number;
  panelShown: boolean;
  onTogglePanel: () => void;
  onReply: () => void;
};

/**
 * The reader's strip: close and step through the folder, who else is here,
 * who owns it, snooze, the actions panel and everything else under "more".
 * Each menu is the one shared context menu opened from its button.
 */
export function ReaderToolbar({
  message,
  position,
  onClose,
  onStep,
  openSteps,
  panelShown,
  onTogglePanel,
  onReply,
}: ReaderToolbarProps) {
  const t = useT();
  const actions = useTriageActions();
  const timeZone = useUserTimezone();
  const meId = useAuthStore((state) => state.user?.id ?? null);
  const assignees = useTriageAssignees().data ?? [];
  const documentTypes = useTriageDocumentTypes().data?.results ?? [];
  const underline = useTriageStore((state) => state.underlineFacts);
  const waiting = message.needsReview;
  const snoozed = isSnoozed(message);
  const ids = [message.id];

  const openOwner = (button: HTMLElement) =>
    openMenuFrom(
      button,
      {
        title: t("Assign to"),
        entries: assigneeEntries({
          t,
          actions,
          ids,
          meId,
          assignees,
          ownerId: message.ownerId,
        }),
      },
      { align: "end" },
    );

  const openSnooze = (button: HTMLElement) => {
    const time = {
      now: Math.floor(Date.now() / 1000),
      timeZone,
      format: USER_FORMATTERS,
    };
    const events = eventSnoozeEntries({ t, actions, message, ids, documentTypes });
    const entries: ContextMenuEntry[] = [
      ...snoozeTimeEntries(t, time, (choice) => actions.snooze(ids, choice)),
      ...(events.length > 0
        ? [
            { type: "separator" } as const,
            { type: "header", label: t("Or until something happens") } as const,
            ...events,
          ]
        : []),
      ...(snoozed
        ? [
            { type: "separator" } as const,
            {
              id: "wake",
              label: t("Wake it now"),
              icon: XCloseIcon,
              onSelect: () => actions.wake(message.id),
            },
          ]
        : []),
    ];
    openMenuFrom(
      button,
      {
        title:
          snoozed && message.snooze !== null
            ? t("Snoozed {0}", snoozeText(t, message.snooze, USER_FORMATTERS))
            : t("Snooze until"),
        entries,
      },
      { align: "end" },
    );
  };

  const openMore = (button: HTMLElement) =>
    openMenuFrom(
      button,
      {
        label: t("More"),
        entries: [
          {
            id: "reply",
            label: t("Reply"),
            icon: CornerDownLeftIcon,
            shortcut: "R",
            onSelect: onReply,
          },
          {
            id: "link",
            label: t("Link by hand"),
            icon: Link01Icon,
            shortcut: "L",
            onSelect: () =>
              useTriageStore.getState().openOverlay({ kind: "link", messageId: message.id }),
          },
          waiting && {
            id: "ignore",
            label: t("Ignore"),
            icon: ArchiveIcon,
            shortcut: "#",
            onSelect: () => actions.ignore(ids),
          },
          { type: "separator" },
          {
            id: "copy-link",
            label: t("Copy link"),
            icon: Copy01Icon,
            onSelect: () => actions.copyLink(message.id),
          },
          {
            id: "download",
            label: t("Download as .eml"),
            icon: Download01Icon,
            onSelect: () => downloadEml(message),
          },
          { type: "separator" },
          {
            id: "underline",
            label: underline ? t("Stop underlining what Nova read") : t("Underline what Nova read"),
            onSelect: () => useTriageStore.getState().setUnderlineFacts(!underline),
          },
        ],
      },
      { align: "end" },
    );

  const owner = message.owner;

  return (
    <div className="flex h-11 shrink-0 items-center gap-0.5 pr-2.5 pl-3">
      <ToolButton label={t("Close")} shortcut="Esc">
        <Button size="icon-sm" variant="quiet" aria-label={t("Close")} onClick={onClose}>
          <XCloseIcon aria-hidden className="size-3.5" />
        </Button>
      </ToolButton>
      <ToolButton label={t("Previous")} shortcut="K">
        <Button
          size="icon-sm"
          variant="quiet"
          aria-label={t("Previous message")}
          disabled={position.index <= 1}
          onClick={() => onStep(-1)}
        >
          <ArrowUpIcon aria-hidden className="size-3.5" />
        </Button>
      </ToolButton>
      <ToolButton label={t("Next")} shortcut="J">
        <Button
          size="icon-sm"
          variant="quiet"
          aria-label={t("Next message")}
          disabled={position.index < 1 || position.index >= position.total}
          onClick={() => onStep(1)}
        >
          <ArrowDownIcon aria-hidden className="size-3.5" />
        </Button>
      </ToolButton>
      {position.index > 0 && position.total > 0 && (
        <span className="text-foreground-subtle mx-1 font-mono text-xs whitespace-nowrap tabular-nums">
          {t("{0} / {1}", position.index, position.total)}
        </span>
      )}
      <span className="flex-1" />
      <PresenceChip messageId={message.id} />
      {waiting && (
        <Button
          size="bare"
          variant="bare"
          aria-haspopup="menu"
          aria-label={owner === null ? t("Assign") : t("Assigned to {0}", owner.name)}
          className="text-foreground-muted hover:bg-surface-hover hover:text-foreground ui-press mr-0.5 inline-flex h-7 items-center gap-1.5 rounded-full pr-2.5 pl-1 text-sm"
          onClick={(event) => openOwner(event.currentTarget)}
        >
          {owner === null ? (
            <>
              <span className="border-border-strong text-foreground-subtle flex size-5 items-center justify-center rounded-full border border-dashed">
                <User01Icon aria-hidden className="size-3" />
              </span>
              <span className="@max-[520px]/reader:sr-only">{t("Assign")}</span>
            </>
          ) : (
            <>
              <PersonAvatar name={owner.name} />
              <span className="@max-[520px]/reader:sr-only">
                {owner.id === meId ? t("You") : (owner.name.split(/\s+/)[0] ?? owner.name)}
              </span>
            </>
          )}
        </Button>
      )}
      {waiting && (
        <ToolButton label={snoozed ? t("Snoozed") : t("Snooze")} shortcut="S">
          <Button
            size="icon-sm"
            variant="quiet"
            aria-haspopup="menu"
            aria-label={t("Snooze")}
            aria-pressed={snoozed}
            className={cn(snoozed && "bg-surface-hover text-foreground")}
            onClick={(event) => openSnooze(event.currentTarget)}
          >
            <ClockIcon aria-hidden className="size-3.5" />
          </Button>
        </ToolButton>
      )}
      <ToolButton label={t("Actions and information")} shortcut="I">
        <Button
          size="bare"
          variant="bare"
          aria-label={t("Actions and information")}
          aria-pressed={panelShown}
          className={cn(
            "text-foreground-subtle hover:bg-surface-hover hover:text-foreground ui-press inline-flex h-7 items-center gap-1.5 rounded-md px-2 text-xs",
            panelShown && "bg-surface-hover text-foreground",
          )}
          onClick={onTogglePanel}
        >
          <LayoutRightIcon aria-hidden className="size-3.5" />
          {waiting && openSteps > 0 && <span className="font-mono tabular-nums">{openSteps}</span>}
        </Button>
      </ToolButton>
      <Button
        size="icon-sm"
        variant="quiet"
        aria-haspopup="menu"
        aria-label={t("More")}
        onClick={(event) => openMore(event.currentTarget)}
      >
        <DotsHorizontalIcon aria-hidden className="size-4" />
      </Button>
    </div>
  );
}
