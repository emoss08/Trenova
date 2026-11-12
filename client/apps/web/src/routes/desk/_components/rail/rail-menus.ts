import { conversationPath } from "@/lib/conversation-path";
import { casePath } from "../attention/attention-record";
import { caseOfThread } from "../record-case/case-subject";
import type { AssistantThread } from "@/types/assistant";
import type { ContextMenuSpec } from "@trenova/shared/components/context-menu/menu-spec";
import {
  ArrowRightIcon,
  Columns01Icon,
  Download01Icon,
  File06Icon,
  Link01Icon,
  LinkExternal01Icon,
  Pencil01Icon,
  Pin01Icon,
  PinOffIcon,
  Trash01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type { RailCaseRow } from "./open-work";

/** What the rail can do to a conversation from its row. */
export type RailThreadActions = {
  open: (path: string) => void;
  /** Shows it in Nova's margin; left out where it would not open beside the page. */
  openBeside?: (thread: AssistantThread) => void;
  openInNewTab: (path: string) => void;
  copyLink: (path: string) => void;
  setPinned: (thread: AssistantThread, pinned: boolean) => void;
  rename: (thread: AssistantThread) => void;
  download: (thread: AssistantThread) => void;
  remove: (thread: AssistantThread) => void;
};

/** What the rail can do to a case or a view in Open work from its row. */
export type RailWorkActions = {
  open: (path: string) => void;
  openInNewTab: (path: string) => void;
  copyLink: (path: string) => void;
  /** The record's own page in the TMS, when it has one. */
  recordPath: (row: RailCaseRow) => string | null;
  /** Takes a row opened this session off Open work. */
  forget: (row: RailCaseRow) => void;
};

/** A conversation's right-click menu on the rail. */
export function threadMenu(
  t: TranslateFn,
  thread: AssistantThread,
  actions: RailThreadActions,
): ContextMenuSpec {
  const path = conversationPath(thread.id);
  const record = caseOfThread(thread);
  const casePage = record ? casePath(record.kind, record.id) : null;

  return {
    // No heading: the menu opens on the row it is about. The name is kept for
    // assistive technology.
    label: thread.title || t("Untitled conversation"),
    entries: [
      casePage !== null && {
        label: t("Open the case"),
        icon: ArrowRightIcon,
        onSelect: () => actions.open(casePage),
      },
      {
        label: casePage !== null ? t("Open the conversation") : t("Open"),
        icon: casePage !== null ? File06Icon : ArrowRightIcon,
        onSelect: () => actions.open(path),
      },
      actions.openBeside && {
        label: t("Open beside the page"),
        icon: Columns01Icon,
        onSelect: () => actions.openBeside?.(thread),
      },
      {
        label: t("Open in a new tab"),
        icon: LinkExternal01Icon,
        onSelect: () => actions.openInNewTab(path),
      },
      { type: "separator" },
      {
        label: thread.pinned ? t("Unpin conversation") : t("Pin conversation"),
        icon: thread.pinned ? PinOffIcon : Pin01Icon,
        onSelect: () => actions.setPinned(thread, !thread.pinned),
      },
      { label: t("Rename"), icon: Pencil01Icon, onSelect: () => actions.rename(thread) },
      {
        label: t("Download transcript"),
        icon: Download01Icon,
        onSelect: () => actions.download(thread),
      },
      { label: t("Copy link"), icon: Link01Icon, onSelect: () => actions.copyLink(path) },
      {
        label: t("Delete conversation"),
        icon: Trash01Icon,
        destructive: true,
        onSelect: () => actions.remove(thread),
      },
    ],
  };
}

/** A case's or a view's right-click menu in Open work. */
export function workMenu(
  t: TranslateFn,
  row: RailCaseRow,
  actions: RailWorkActions,
): ContextMenuSpec {
  const record = actions.recordPath(row);

  return {
    label: row.title,
    entries: [
      { label: t("Open"), icon: ArrowRightIcon, onSelect: () => actions.open(row.to) },
      {
        label: t("Open in a new tab"),
        icon: LinkExternal01Icon,
        onSelect: () => actions.openInNewTab(row.to),
      },
      record !== null && {
        label: t("Open the record"),
        icon: File06Icon,
        onSelect: () => actions.open(record),
      },
      { type: "separator" },
      { label: t("Copy link"), icon: Link01Icon, onSelect: () => actions.copyLink(row.to) },
      // What waits on the person is listed by the server until it is decided,
      // so only a row the session put here can be taken off.
      !row.waiting && {
        label: t("Remove from Cases"),
        icon: XCloseIcon,
        onSelect: () => actions.forget(row),
      },
    ],
  };
}
