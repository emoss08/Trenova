import type {
  InboundMessageFilter,
  InboundMessageRow,
  TriageAssignee,
  TriageRuleAction,
} from "@/lib/graphql/inbox";
import type { SelectOption } from "@/lib/graphql/select-options";
import { isAwaitingReply, isSnoozed, senderName } from "@/lib/triage/message-state";
import type {
  ContextMenuEntries,
  ContextMenuEntry,
  ContextMenuSpec,
} from "@trenova/shared/components/context-menu/menu-spec";
import {
  ArchiveIcon,
  ArrowRightIcon,
  CheckIcon,
  ClockIcon,
  Copy01Icon,
  Edit05Icon,
  LinkExternal01Icon,
  Link01Icon,
  MessageChatCircleIcon,
  RefreshCw01Icon,
  Send01Icon,
  User01Icon,
  ZapIcon,
} from "@trenova/shared/components/icons";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { ruleActionLabel, type CopyFormatters } from "./action-copy";
import { CLASSIFICATION_DOT, INBOUND_CLASSIFICATIONS, classificationLabel } from "./classification";
import { eventSnoozeOptions, snoozePresets, type SnoozePresetKey } from "./snooze-times";
import type { SnoozeChoice, TriageActions } from "./use-triage-actions";

/**
 * The context menus of Triage, declared as rows for the one shared menu.
 * Each builder runs when its menu opens, with the state of that moment.
 */

function presetLabel(t: TranslateFn, key: SnoozePresetKey): string {
  switch (key) {
    case "laterToday":
      return t("Later today");
    case "tomorrow":
      return t("Tomorrow morning");
    case "monday":
      return t("Monday");
  }
}

function KindDot({ className }: { className: string }) {
  return <span aria-hidden className={cn("size-2 rounded-full", className)} />;
}

function Initials({ name }: { name: string }) {
  return (
    <span
      aria-hidden
      className="bg-muted text-foreground-muted flex size-4 items-center justify-center rounded-full text-3xs font-medium"
    >
      {getNameInitials(name, "?")}
    </span>
  );
}

type TimeContext = { now: number; timeZone: string; format: CopyFormatters };

/** The three times, each with when it lands. */
export function snoozeTimeEntries(
  t: TranslateFn,
  time: TimeContext,
  pick: (choice: SnoozeChoice) => void,
): ContextMenuEntry[] {
  return snoozePresets(time.now, time.timeZone).map((preset) => ({
    id: preset.key,
    label: presetLabel(t, preset.key),
    hint: time.format.when(preset.until),
    onSelect: () => pick({ kind: "until", until: preset.until }),
  }));
}

const RULE_ACTIONS: readonly TriageRuleAction[] = [
  "FileOnShipment",
  "FileAndQueuePay",
  "DraftReply",
  "DraftLoad",
  "LinkBySender",
  "Ignore",
];

export type RowMenuInput = {
  t: TranslateFn;
  actions: TriageActions;
  /** The row the menu opened on. */
  message: InboundMessageRow;
  /** Every selected message, when the row is one of two or more selected; else empty. */
  selection: readonly InboundMessageRow[];
  meId: string | null;
  assignees: readonly TriageAssignee[];
  documentTypes: readonly SelectOption[];
  time: TimeContext;
  open: (id: string) => void;
  link: (id: string) => void;
};

/** A row's menu, or the selection's when the row is part of one. */
export function rowMenu({
  t,
  actions,
  message,
  selection,
  meId,
  assignees,
  documentTypes,
  time,
  open,
  link,
}: RowMenuInput): ContextMenuSpec {
  const many = selection.length > 1;
  const targets = many ? selection : [message];
  const waiting = targets.filter((row) => row.needsReview);
  const waitingIds = waiting.map((row) => row.id);
  const settledIds = targets.filter((row) => !row.needsReview).map((row) => row.id);
  const need = waiting.length > 0;
  const single = !many;
  const sentReply = message.actions.some(
    (action) => action.kind === "Reply" && action.reply?.status === "Sent",
  );

  const events = single
    ? eventSnoozeOptions(message).flatMap((option): ContextMenuEntry[] => {
        switch (option.event) {
          case "ShipmentDelivered":
            return [
              {
                id: "delivered",
                label: t("Until {0} delivers", option.proNumber),
                icon: ZapIcon,
                onSelect: () =>
                  actions.snooze(waitingIds, { kind: "event", event: "ShipmentDelivered" }),
              },
            ];
          case "InvoicePaid":
            return [
              {
                id: "paid",
                label: t("Until the invoice is paid"),
                icon: ZapIcon,
                onSelect: () => actions.snooze(waitingIds, { kind: "event", event: "InvoicePaid" }),
              },
            ];
          case "DocumentAttached":
            return documentTypes.length === 0
              ? []
              : [
                  {
                    id: "document",
                    label: t("Until a document is attached"),
                    icon: ZapIcon,
                    submenu: documentTypes.map((type) => ({
                      id: type.id,
                      label: type.label,
                      onSelect: () =>
                        actions.snooze(waitingIds, {
                          kind: "event",
                          event: "DocumentAttached",
                          documentTypeId: type.id,
                          documentTypeName: type.label,
                        }),
                    })),
                  },
                ];
        }
      })
    : [];

  const assigneeRows: ContextMenuEntry[] = [
    ...(meId === null
      ? []
      : [
          {
            id: "me",
            label: t("Me"),
            icon: User01Icon,
            onSelect: () => actions.assignToMe(waitingIds),
          },
        ]),
    ...assignees
      .filter((assignee) => assignee.user.id !== meId)
      .map((assignee) => ({
        id: assignee.user.id,
        label: assignee.user.name,
        leading: <Initials name={assignee.user.name} />,
        hint:
          assignee.waiting > 0
            ? t("{0, plural, one {# waiting} other {# waiting}}", assignee.waiting)
            : undefined,
        onSelect: () => actions.assign(waitingIds, assignee.user),
      })),
    { type: "separator" },
    { id: "nobody", label: t("Nobody"), onSelect: () => actions.assign(waitingIds, null) },
  ];

  return {
    title: many
      ? t("{0, plural, one {# selected} other {# selected}}", selection.length)
      : senderName(message),
    subtitle: many ? undefined : message.subject === "" ? t("(no subject)") : message.subject,
    entries: [
      single && {
        id: "open",
        label: t("Open"),
        icon: ArrowRightIcon,
        shortcut: "↵",
        onSelect: () => open(message.id),
      },
      single && {
        id: "new-tab",
        label: t("Open in a new tab"),
        icon: LinkExternal01Icon,
        onSelect: () => actions.openInNewTab(message.id),
      },
      { type: "separator" },
      need && {
        id: "handle",
        label: t("Mark handled"),
        icon: CheckIcon,
        shortcut: "E",
        onSelect: () => actions.handle(waitingIds),
      },
      need && {
        id: "ignore",
        label: t("Ignore"),
        icon: ArchiveIcon,
        shortcut: "#",
        onSelect: () => actions.ignore(waitingIds),
      },
      need && {
        id: "snooze",
        label: t("Snooze"),
        icon: ClockIcon,
        shortcut: "S",
        submenu: [
          ...snoozeTimeEntries(t, time, (choice) => actions.snooze(waitingIds, choice)),
          { type: "separator" },
          { type: "header", label: t("Until something happens") },
          ...events,
        ],
      },
      need &&
        single &&
        isSnoozed(message) && {
          id: "wake",
          label: t("Wake it now"),
          icon: ClockIcon,
          onSelect: () => actions.wake(message.id),
        },
      need &&
        single &&
        sentReply &&
        !isAwaitingReply(message) && {
          id: "wait",
          label: t("Wait on them"),
          icon: Send01Icon,
          submenu: [
            { type: "header", label: t("Back in Waiting on you if they haven't replied by") },
            ...snoozeTimeEntries(t, time, (choice) => {
              if (choice.kind === "until") {
                actions.waitForReply(message.id, choice.until);
              }
            }),
          ],
        },
      need &&
        single &&
        isAwaitingReply(message) && {
          id: "stop-waiting",
          label: t("Stop waiting"),
          icon: Send01Icon,
          onSelect: () => actions.stopWaiting(message.id),
        },
      settledIds.length > 0 && {
        id: "reopen",
        label: t("Reopen"),
        icon: RefreshCw01Icon,
        onSelect: () => actions.reopen(settledIds),
      },
      { type: "separator" },
      need && {
        id: "assign",
        label: t("Assign"),
        icon: User01Icon,
        shortcut: "M",
        submenu: assigneeRows,
      },
      single && {
        id: "kind",
        label: t("Correct the kind"),
        icon: Edit05Icon,
        submenu: INBOUND_CLASSIFICATIONS.filter((kind) => kind !== message.classification).map(
          (kind) => ({
            id: kind,
            label: classificationLabel(t, kind),
            leading: <KindDot className={CLASSIFICATION_DOT[kind]} />,
            onSelect: () => actions.correctKind(message.id, kind),
          }),
        ),
      },
      single && {
        id: "link",
        label: t("Link to a record…"),
        icon: Link01Icon,
        shortcut: "L",
        onSelect: () => link(message.id),
      },
      single && {
        id: "desk",
        label: t("Ask an agent in Desk"),
        icon: MessageChatCircleIcon,
        shortcut: "A",
        onSelect: () => actions.askInDesk(message),
      },
      { type: "separator" },
      single && {
        id: "copy-link",
        label: t("Copy link"),
        icon: Copy01Icon,
        onSelect: () => actions.copyLink(message.id),
      },
      single && {
        id: "copy-address",
        label: t("Copy sender address"),
        icon: Copy01Icon,
        hint: message.fromAddress,
        onSelect: () => actions.copyText(message.fromAddress),
      },
      single &&
        message.classification !== null && {
          id: "rule",
          label: t("Make a rule from this sender"),
          icon: ZapIcon,
          submenu: [
            { type: "header", label: t("Nova should…") },
            ...RULE_ACTIONS.map((action) => ({
              id: action,
              label: ruleActionLabel(t, action),
              onSelect: () =>
                actions.draftRule({
                  classification: message.classification ?? "Other",
                  senderScope: "Domain",
                  sender: message.senderDomain,
                  mailboxId: message.mailboxId,
                  action,
                }),
            })),
          ],
        },
    ],
  };
}

export type FolderMenuInput = {
  t: TranslateFn;
  actions: TriageActions;
  title: string;
  waiting: number;
  /** The folder as the server is asked for it; its waiting messages are what the menu acts on. */
  filter: InboundMessageFilter | null;
  time: TimeContext;
  open: () => void;
  /** Extra rows for this kind of folder: rename and delete a view, test a source, make a rule. */
  extra: ContextMenuEntries;
};

/** The menu on a kind, a view or a source in the rail. */
export function folderMenu({
  t,
  actions,
  title,
  waiting,
  filter,
  time,
  open,
  extra,
}: FolderMenuInput): ContextMenuSpec {
  const none = waiting === 0 || filter === null;

  return {
    title,
    subtitle: t("{0, plural, one {# waiting} other {# waiting}}", waiting),
    entries: [
      { id: "open", label: t("Open"), icon: ArrowRightIcon, onSelect: open },
      { type: "separator" },
      {
        id: "handle-all",
        label: t("Mark all handled"),
        icon: CheckIcon,
        disabled: none,
        hint: none ? undefined : String(waiting),
        onSelect: () => {
          if (filter !== null) {
            actions.handleFolder(filter);
          }
        },
      },
      {
        id: "snooze-all",
        label: t("Snooze all"),
        icon: ClockIcon,
        disabled: none,
        submenu: snoozeTimeEntries(t, time, (choice) => {
          if (filter !== null) {
            actions.snoozeFolder(filter, choice);
          }
        }),
      },
      { type: "separator" },
      ...extra,
    ],
  };
}
