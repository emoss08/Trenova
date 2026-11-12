import { senderName } from "@/lib/triage/message-state";
import {
  Attachment01Icon,
  ChevronDownIcon,
  Link01Icon,
  ShieldTickIcon,
  User01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { Activity, memo, useId, useState } from "react";
import { USER_FORMATTERS, actionCopy } from "../action-copy";
import { dueSpan, dueText } from "../due";
import type { DeckEntry, DeckGroup, DeckItem } from "./briefing-model";

/** How many people a group lists before it says how many more. */
const LISTED = 4;

export type DeckChip = {
  group: DeckGroup;
  label: string;
  /** The message the chip opens again. */
  firstId: string | null;
};

/** A group's mark; Nova's own group, the drafts it can write, wears Nova's. */
const ICON: Record<DeckGroup, IconComponent | null> = {
  File: Attachment01Icon,
  Draft: null,
  Decide: User01Icon,
  Link: Link01Icon,
  Held: ShieldTickIcon,
};

function deckTitle(t: TranslateFn, entry: DeckEntry): string {
  switch (entry.group) {
    case "File":
      return t("{0, plural, one {# document to file} other {# documents to file}}", entry.count);
    case "Draft":
      return t(
        "{0, plural, one {# draft Nova can write} other {# drafts Nova can write}}",
        entry.count,
      );
    case "Decide":
      return t("{0, plural, one {# needs your call} other {# need your call}}", entry.count);
    case "Link":
      return t("{0, plural, one {# without a record} other {# without a record}}", entry.count);
    case "Held":
      return t("{0, plural, one {# held back} other {# held back}}", entry.count);
  }
}

function deckWhy(t: TranslateFn, group: DeckGroup): string {
  switch (group) {
    case "File":
      return t("Each one names its PRO. One tap files them all.");
    case "Draft":
      return t("Loads and replies. Nothing is sent or created until you look.");
    case "Decide":
      return t("Disputes, exceptions — anything with money on it.");
    case "Link":
      return t("No PRO, PO or known sender to match on.");
    case "Held":
      return t("Looks like phishing. Ignore unless you know them.");
  }
}

function deckButton(t: TranslateFn, group: DeckGroup): string {
  switch (group) {
    case "File":
      return t("File all");
    case "Draft":
      return t("Draft all");
    case "Decide":
      return t("Start");
    case "Link":
      return t("Link");
    case "Held":
      return t("Review");
  }
}

type DeckProps = {
  entries: readonly DeckEntry[];
  now: number;
  /** Nova is drafting now; its group shows it is at work until the drafts are in. */
  drafting: boolean;
  /** Whether the reader may work messages; without it a group has no button. */
  canAct: boolean;
  onAct: (entry: DeckEntry) => void;
  onOpenMessage: (id: string) => void;
};

/**
 * What is left, by what it needs: one group per kind of decision, quickest
 * first, every group in sight. A group names the people it is about, how long
 * each has left, and the one thing that clears it.
 */
export function Deck({ entries, now, drafting, canAct, onAct, onOpenMessage }: DeckProps) {
  return (
    <div className="flex flex-col gap-2.5">
      {entries.map((entry, index) => (
        <DeckGroupCard
          key={entry.group}
          entry={entry}
          lead={index === 0}
          now={now}
          working={entry.group === "Draft" && drafting}
          canAct={canAct}
          onAct={onAct}
          onOpenMessage={onOpenMessage}
        />
      ))}
    </div>
  );
}

const DeckGroupCard = memo(function DeckGroupCard({
  entry,
  lead,
  now,
  working,
  canAct,
  onAct,
  onOpenMessage,
}: {
  entry: DeckEntry;
  /** The first group is the one to start with, and its button says so. */
  lead: boolean;
  now: number;
  working: boolean;
  canAct: boolean;
  onAct: (entry: DeckEntry) => void;
  onOpenMessage: (id: string) => void;
}) {
  const t = useT();
  const listId = useId();
  // The group to start with is open; the rest say what they are and how many,
  // and name their people when asked, so the page reads at a glance.
  const [open, setOpen] = useState(lead);
  const Icon = ICON[entry.group];
  const title = deckTitle(t, entry);

  return (
    <section
      aria-label={title}
      className="animate-rise overflow-hidden rounded-[8px] border border-dsk-b bg-dsk-card"
    >
      <div className="flex items-center gap-3 px-3 py-2.5">
        <button
          type="button"
          aria-expanded={open}
          aria-controls={listId}
          onClick={() => setOpen((shown) => !shown)}
          className="group/head flex min-w-0 flex-1 items-center gap-3 rounded-[6px] ui-focus-ring text-left"
        >
          <span
            aria-hidden
            className="grid size-7 flex-none place-items-center rounded-[6px] border border-dsk-b-sub bg-dsk-sunken text-dsk-fg2"
          >
            {Icon === null ? (
              <NovaBadge size="sm" spin={working ? "always" : "none"} />
            ) : (
              <Icon className="size-3.5" />
            )}
          </span>
          <span className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="flex items-center gap-1.5">
              <span className="truncate text-dsk-base font-medium text-dsk-fg">{title}</span>
              <ChevronDownIcon
                aria-hidden
                className={cn(
                  "size-3 flex-none text-dsk-faint transition-transform duration-200 group-hover/head:text-dsk-fg2",
                  open && "rotate-180",
                )}
              />
            </span>
            <span className="truncate text-dsk-sm text-dsk-subtle">{deckWhy(t, entry.group)}</span>
          </span>
        </button>
        {entry.estimateMinutes > 0 && (
          <span className="flex-none text-xs text-dsk-faint tabular-nums @max-2xl:hidden">
            {t("~{0} min", entry.estimateMinutes)}
          </span>
        )}
        {canAct && (
          <Button
            size="sm"
            variant={lead ? "default" : "secondary"}
            isLoading={working}
            loadingText={t("Drafting…")}
            onClick={() => onAct(entry)}
          >
            {deckButton(t, entry.group)}
          </Button>
        )}
      </div>
      <Activity mode={open ? "visible" : "hidden"}>
        <ul id={listId} className="flex flex-col border-t border-dsk-b-sub p-1">
          {entry.items.slice(0, LISTED).map((item) => (
            <li key={item.messageId}>
              <DeckRow item={item} now={now} onOpen={onOpenMessage} />
            </li>
          ))}
          {entry.count > LISTED && (
            <li className="px-2 pt-1 pb-1.5 text-xs text-dsk-subtle">
              {t("{0, plural, one {+ # more} other {+ # more}}", entry.count - LISTED)}
            </li>
          )}
        </ul>
      </Activity>
    </section>
  );
});

const DeckRow = memo(function DeckRow({
  item,
  now,
  onOpen,
}: {
  item: DeckItem;
  now: number;
  onOpen: (id: string) => void;
}) {
  const t = useT();
  const name = senderName(item);
  const next =
    item.action === null ? item.subject : actionCopy(t, item.action, item, USER_FORMATTERS).title;
  const due = item.dueAt === null ? null : dueSpan(item.dueAt, now);

  return (
    <button
      type="button"
      onClick={() => onOpen(item.messageId)}
      className="grid w-full grid-cols-[22px_minmax(0,11rem)_minmax(0,1fr)_auto] items-center gap-2.5 rounded-[6px] ui-focus-ring px-2 py-1.5 text-left hover:bg-dsk-hover @max-2xl:grid-cols-[22px_minmax(0,1fr)_auto]"
    >
      <span
        aria-hidden
        className="grid size-5.5 place-items-center rounded-full border border-dsk-b-sub bg-dsk-sunken text-3xs font-medium text-dsk-muted"
      >
        {getNameInitials(name, "?")}
      </span>
      <b className="truncate text-dsk-base font-medium text-dsk-fg">{name}</b>
      <span className="truncate text-dsk-sm text-dsk-subtle @max-2xl:hidden">{next}</span>
      {due !== null && (
        <span
          className={cn(
            "text-xs tabular-nums",
            due.state === "over" ? "text-dsk-warn-fg" : "text-dsk-subtle",
          )}
        >
          {dueText(t, due)}
        </span>
      )}
    </button>
  );
});

/** Groups that were acted on, as badges that open their first message again. */
export function DeckChips({
  chips,
  onOpen,
}: {
  chips: readonly DeckChip[];
  onOpen: (chip: DeckChip) => void;
}) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {chips.map((chip) => (
        <button
          key={chip.group}
          type="button"
          onClick={() => onOpen(chip)}
          className="rounded-badge ui-focus-ring"
        >
          <Badge variant="success" dot>
            {chip.label}
          </Badge>
        </button>
      ))}
    </div>
  );
}
