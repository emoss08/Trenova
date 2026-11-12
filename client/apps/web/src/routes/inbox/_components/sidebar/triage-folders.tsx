import { HoverGlideGroup } from "@/components/hover-glide";
import type { InboundClassification, InboundMailbox, TriageView } from "@/lib/graphql/inbox";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { useContextMenuTrigger } from "@trenova/shared/components/context-menu/context-menu-host";
import type { ContextMenuSpec } from "@trenova/shared/components/context-menu/menu-spec";
import {
  Edit05Icon,
  Inbox01Icon,
  PlusIcon,
  SearchLgIcon,
  Send01Icon,
  Settings01Icon,
  Trash01Icon,
  XCloseIcon,
  ZapIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { useNavigate } from "react-router";
import { useTriageStore } from "@/stores/triage-store";
import { USER_FORMATTERS } from "../action-copy";
import { CLASSIFICATION_DOT, classificationLabel } from "../classification";
import { LANE_ICON, laneLabel, mailboxName } from "../folder-labels";
import {
  BRIEFING,
  RULES,
  folderFilter,
  isSameFolder,
  type QueueFolder,
  type TriageFolder,
} from "../folders";
import { laneCount, visibleLanes, type LaneKey } from "../lanes";
import { folderMenu } from "../menus";
import {
  useTriageCounts,
  useTriageMailboxes,
  useTriageRules,
  useTriageViews,
} from "../use-triage-data";
import { useTriageActions } from "../use-triage-actions";

/** Where a source's settings live: the mailbox administration page. */
const SOURCES_PATH = "/admin/inbound-mailboxes";

export type TriageFoldersProps = {
  folder: TriageFolder;
  onSelect: (folder: TriageFolder) => void;
};

/**
 * Triage's places, drawn in the Desk's rail while Triage is in front: the
 * Briefing, the lanes, the kinds still waiting, saved views, the sources mail
 * arrives on, and the rules Nova follows. Only numbers someone can act on are
 * shown, and every count opens what is behind it. Kinds, views and sources
 * open a menu on right-click or a long press.
 */
export function TriageFolders({ folder, onSelect }: TriageFoldersProps) {
  const t = useT();
  const navigate = useNavigate();
  const actions = useTriageActions();
  const timeZone = useUserTimezone();
  const now = useNowSeconds();
  const counts = useTriageCounts().data;
  const mailboxes = useTriageMailboxes().data ?? [];
  const views = useTriageViews().data ?? [];
  const rules = useTriageRules().data ?? [];
  const [renaming, setRenaming] = useState<string | null>(null);
  const openConnect = () =>
    useTriageStore.getState().openOverlay({ kind: "connect", mailboxId: null });

  const kinds = (counts?.byClassification ?? []).filter((row) => row.waiting > 0);
  const waitingByMailbox = new Map(
    (counts?.byMailbox ?? []).map((row) => [row.mailboxId, row.waiting] as const),
  );
  const enabledRules = rules.filter((rule) => rule.enabled).length;
  const time = { now, timeZone, format: USER_FORMATTERS };

  const menuFor = (
    target: QueueFolder,
    title: string,
    waiting: number,
    extra: ContextMenuSpec["entries"],
  ) =>
    folderMenu({
      t,
      actions,
      title,
      waiting,
      filter: folderFilter(target, { search: "", sort: "Due", views }),
      time,
      open: () => onSelect(target),
      extra,
    });

  return (
    <HoverGlideGroup className="bg-transparent" glideClassName="rounded-[8px] bg-dsk-hover">
      <nav aria-label={t("Triage folders")} className="flex flex-col gap-1">
      <RailHeading>{t("Triage")}</RailHeading>
      <RailRow
        active={folder.kind === "briefing"}
        leading={<NovaBadge size="sm" spin="hover" />}
        label={t("Briefing")}
        onClick={() => onSelect(BRIEFING)}
      />
      {visibleLanes(counts).map((lane) => (
        <LaneRow
          key={lane}
          lane={lane}
          count={laneCount(counts, lane)}
          active={isSameFolder(folder, { kind: "lane", lane })}
          onSelect={onSelect}
        />
      ))}

      {kinds.length > 0 && <RailHeading>{t("Waiting")}</RailHeading>}
      {kinds.map((row) => {
        const target: QueueFolder = {
          kind: "classification",
          classification: row.classification,
        };
        const title = classificationLabel(t, row.classification);
        return (
          <RailRow
            key={row.classification}
            active={isSameFolder(folder, target)}
            leading={<KindDot classification={row.classification} />}
            label={title}
            count={row.waiting}
            onClick={() => onSelect(target)}
            menu={() =>
              menuFor(target, title, row.waiting, [
                {
                  id: "rule",
                  label: t("Make a rule for {0}", toSentenceFragment(title)),
                  icon: ZapIcon,
                  onSelect: () => {
                    useTriageStore.getState().seedRule({ classification: row.classification });
                    onSelect(RULES);
                  },
                },
              ])
            }
          />
        );
      })}

      {views.length > 0 && <RailHeading>{t("Views")}</RailHeading>}
      {views.map((view) => {
        const target: QueueFolder = { kind: "view", viewId: view.id };
        return (
          <ViewRow
            key={view.id}
            view={view}
            active={isSameFolder(folder, target)}
            renaming={renaming === view.id}
            onRenamed={(name) => {
              setRenaming(null);
              actions.renameView(view, name);
            }}
            onCancelRename={() => setRenaming(null)}
            onClick={() => onSelect(target)}
            onDelete={() => {
              if (isSameFolder(folder, target)) {
                onSelect(BRIEFING);
              }
              actions.deleteView(view);
            }}
            menu={() =>
              menuFor(target, view.name, view.waiting, [
                {
                  id: "rename",
                  label: t("Rename"),
                  icon: Edit05Icon,
                  onSelect: () => setRenaming(view.id),
                },
                {
                  id: "delete",
                  label: t("Delete view"),
                  icon: Trash01Icon,
                  destructive: true,
                  onSelect: () => {
                    if (isSameFolder(folder, target)) {
                      onSelect(BRIEFING);
                    }
                    actions.deleteView(view);
                  },
                },
              ])
            }
          />
        );
      })}

      <RailHeading
        action={
          mailboxes.length > 0 ? (
            <button
              type="button"
              onClick={openConnect}
              className="rounded-sm ui-focus-ring text-xs text-dsk-subtle hover:text-dsk-fg"
            >
              {t("Add")}
            </button>
          ) : undefined
        }
      >
        {t("Sources")}
      </RailHeading>
      {mailboxes.length === 0 ? (
        <button
          type="button"
          onClick={openConnect}
          data-glide-target
          className="flex h-7.5 items-center gap-2.5 rounded-[8px] border border-dashed ui-focus-ring border-dsk-b px-2.5 text-dsk-base text-dsk-fg2 hover:text-dsk-fg"
        >
          <PlusIcon aria-hidden className="size-3.5" />
          <span>{t("Add a source")}</span>
        </button>
      ) : (
        mailboxes.map((mailbox) => (
          <SourceRow
            key={mailbox.id}
            mailbox={mailbox}
            active={isSameFolder(folder, { kind: "mailbox", mailboxId: mailbox.id })}
            onSelect={onSelect}
            menu={(target) =>
              menuFor(target, mailboxName(mailbox), waitingByMailbox.get(mailbox.id) ?? 0, [
                {
                  id: "test",
                  label: t("Send a test email"),
                  icon: Send01Icon,
                  hint: mailbox.address,
                  onSelect: () => actions.sendTestEmail(mailbox),
                },
                {
                  id: "settings",
                  label: t("Source settings"),
                  icon: Settings01Icon,
                  onSelect: () => navigate(SOURCES_PATH),
                },
              ])
            }
          />
        ))
      )}

      <RailHeading>{t("Automation")}</RailHeading>
      <RailRow
        active={folder.kind === "rules"}
        icon={ZapIcon}
        label={t("Rules")}
        count={enabledRules}
        onClick={() => onSelect(RULES)}
      />
    </nav>
    </HoverGlideGroup>
  );
}

function RailHeading({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex items-center justify-between px-2.5 pt-4.5 pb-2 text-xs leading-none text-dsk-faint">
      <span>{children}</span>
      {action}
    </div>
  );
}

function KindDot({ classification }: { classification: InboundClassification }) {
  return (
    <span aria-hidden className="flex size-3.5 items-center justify-center">
      <span className={cn("size-2 rounded-full", CLASSIFICATION_DOT[classification])} />
    </span>
  );
}

type RailRowProps = {
  active: boolean;
  icon?: IconComponent;
  leading?: ReactNode;
  label: string;
  count?: number;
  /** "loud" draws the count as a warning chip: what is waiting on you. */
  tone?: "quiet" | "loud";
  trailing?: ReactNode;
  title?: string;
  onClick: () => void;
  menu?: () => ContextMenuSpec | null;
};

const NO_MENU = () => null;

/** A row of the Desk's rail, lit the way the rail lights the place in front. */
// i18n-ignore: class names, never shown to a person
const ROW_CLASS = [
  "ui-focus-ring relative flex h-7.5 w-full shrink-0 items-center gap-2.5 rounded-[8px] px-2.5",
  "text-left text-dsk-base text-dsk-fg2 hover:text-dsk-fg",
  "aria-[current=page]:bg-dsk-canvas aria-[current=page]:text-dsk-fg",
  "aria-[current=page]:shadow-[0_0_0_1px_var(--dsk-b)]",
].join(" ");

function RailRow({
  active,
  icon: Icon,
  leading,
  label,
  count,
  tone = "quiet",
  trailing,
  title,
  onClick,
  menu = NO_MENU,
}: RailRowProps) {
  const trigger = useContextMenuTrigger(menu);

  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      title={title}
      {...trigger}
      data-glide-target
      className={ROW_CLASS}
    >
      {leading ??
        (Icon === undefined ? null : (
          <Icon aria-hidden className="size-3.5 shrink-0 text-dsk-subtle" />
        ))}
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {trailing}
      {count !== undefined && count > 0 && (
        <span
          className={cn(
            "text-xs tabular-nums",
            tone === "loud"
              ? "flex h-4.5 min-w-5 items-center justify-center rounded-badge bg-warning px-1.5 font-semibold text-warning-on-solid"
              : "text-dsk-subtle",
          )}
        >
          {count}
        </span>
      )}
    </button>
  );
}

function LaneRow({
  lane,
  count,
  active,
  onSelect,
}: {
  lane: LaneKey;
  count: number;
  active: boolean;
  onSelect: (folder: TriageFolder) => void;
}) {
  const t = useT();
  const counted = lane === "waiting" || lane === "them" || lane === "mine" || lane === "snoozed";

  return (
    <RailRow
      active={active}
      icon={LANE_ICON[lane]}
      label={laneLabel(t, lane)}
      count={counted ? count : undefined}
      tone={lane === "waiting" ? "loud" : "quiet"}
      onClick={() => onSelect({ kind: "lane", lane })}
    />
  );
}

function ViewRow({
  view,
  active,
  renaming,
  onRenamed,
  onCancelRename,
  onClick,
  onDelete,
  menu,
}: {
  view: TriageView;
  active: boolean;
  renaming: boolean;
  onRenamed: (name: string) => void;
  onCancelRename: () => void;
  onClick: () => void;
  onDelete: () => void;
  menu: () => ContextMenuSpec;
}) {
  const t = useT();
  const trigger = useContextMenuTrigger(menu);

  if (renaming) {
    return (
      <div className="flex h-7.5 items-center gap-2.5 rounded-[8px] bg-dsk-canvas px-2.5 shadow-[0_0_0_1px_var(--dsk-b)]">
        <SearchLgIcon aria-hidden className="size-3.5 shrink-0 text-dsk-subtle" />
        <input
          autoFocus
          defaultValue={view.name}
          aria-label={t("View name")}
          onBlur={(event) => onRenamed(event.currentTarget.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.currentTarget.blur();
            }
            if (event.key === "Escape") {
              event.preventDefault();
              onCancelRename();
            }
          }}
          className="min-w-0 flex-1 bg-transparent text-dsk-base text-dsk-fg outline-none"
        />
      </div>
    );
  }

  return (
    <div className="group/view relative">
      <button
        type="button"
        onClick={onClick}
        aria-current={active ? "page" : undefined}
        {...trigger}
        data-glide-target
        className={ROW_CLASS}
      >
        <SearchLgIcon aria-hidden className="size-3.5 shrink-0 text-dsk-subtle" />
        <span className="min-w-0 flex-1 truncate">{view.name}</span>
        {view.waiting > 0 && (
          <span className="text-xs text-dsk-subtle tabular-nums group-hover/view:invisible">
            {view.waiting}
          </span>
        )}
      </button>
      <button
        type="button"
        aria-label={t("Remove view")}
        onClick={onDelete}
        className="absolute top-1/2 right-2 hidden size-4.5 -translate-y-1/2 items-center justify-center rounded-sm ui-focus-ring text-dsk-subtle group-hover/view:flex hover:text-dsk-fg focus-visible:flex"
      >
        <XCloseIcon aria-hidden className="size-2.5" />
      </button>
    </div>
  );
}

function SourceRow({
  mailbox,
  active,
  onSelect,
  menu,
}: {
  mailbox: InboundMailbox;
  active: boolean;
  onSelect: (folder: TriageFolder) => void;
  menu: (target: QueueFolder) => ContextMenuSpec;
}) {
  const t = useT();
  const target: QueueFolder = { kind: "mailbox", mailboxId: mailbox.id };
  const unhealthy = mailbox.health === "Quiet" || mailbox.health === "NeverReceived";

  return (
    <RailRow
      active={active}
      icon={Inbox01Icon}
      label={mailboxName(mailbox)}
      title={mailbox.address}
      onClick={() => onSelect(target)}
      menu={() => menu(target)}
      trailing={
        unhealthy ? (
          <span
            role="img"
            aria-label={t("Mail has stopped arriving")}
            className="size-1.5 shrink-0 rounded-full bg-warning"
          />
        ) : undefined
      }
    />
  );
}
