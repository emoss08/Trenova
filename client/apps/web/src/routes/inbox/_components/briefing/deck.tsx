import { senderName } from "@/lib/triage/message-state";
import {
  Attachment01Icon,
  CheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  Link01Icon,
  ShieldTickIcon,
  User01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { memo, useState, type CSSProperties } from "react";
import { USER_FORMATTERS, actionCopy } from "../action-copy";
import { dueSpan, dueText } from "../due";
import type { DeckEntry, DeckGroup, DeckItem } from "./briefing-model";

/** How many cards of the stack are drawn behind the front one. */
const STACK_DEPTH = 3;
/** How many people the front card lists. */
const LISTED = 4;

export type DeckChip = {
  group: DeckGroup;
  label: string;
  /** The message the chip opens again. */
  firstId: string | null;
  busy: boolean;
};

type GroupLook = { icon: IconComponent | null; tile: string };

const LOOK: Record<DeckGroup, GroupLook> = {
  File: { icon: Attachment01Icon, tile: "bg-info-subtle text-info" },
  Draft: { icon: null, tile: "tb-nova-tile" },
  Decide: { icon: User01Icon, tile: "bg-surface-hover text-foreground-muted" },
  Link: { icon: Link01Icon, tile: "bg-warning-subtle text-warning" },
  Held: { icon: ShieldTickIcon, tile: "bg-warning-subtle text-warning" },
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
  /** Ids Nova is drafting now; Nova's card shows it is at work until they have their drafts. */
  drafting: boolean;
  /** The group whose card is leaving the stack after its action. */
  flying: DeckGroup | null;
  onAct: (entry: DeckEntry) => void;
  onOpenMessage: (id: string) => void;
};

/**
 * The deck: one card per kind of decision, stacked, the front one in hand.
 * The front card names the people it is about, how long each has left, and
 * the one thing that clears the card. Nova's card, the drafts it can write,
 * has its own surface; the others are plain.
 */
export function Deck({ entries, now, drafting, flying, onAct, onOpenMessage }: DeckProps) {
  const t = useT();
  const [current, setCurrent] = useState(0);
  const count = entries.length;
  const front = count === 0 ? 0 : Math.min(current, count - 1);
  const step = (delta: number) => setCurrent((front + delta + count) % count);

  return (
    <div
      className="grid w-full max-w-140 pb-9 @max-2xl:pb-0"
      role="region"
      aria-roledescription={t("deck")}
      aria-label={t("What is left, by what it needs")}
    >
      {entries.map((entry, index) => {
        const offset = (index - front + count) % count;
        if (offset >= STACK_DEPTH) {
          return null;
        }
        return (
          <DeckCard
            key={entry.group}
            entry={entry}
            offset={offset}
            position={index + 1}
            total={count}
            now={now}
            working={entry.group === "Draft" && drafting}
            flying={flying === entry.group}
            onBring={() => setCurrent(index)}
            onStep={count > 1 ? step : null}
            onAct={onAct}
            onOpenMessage={onOpenMessage}
          />
        );
      })}
    </div>
  );
}

const DeckCard = memo(function DeckCard({
  entry,
  offset,
  position,
  total,
  now,
  working,
  flying,
  onBring,
  onStep,
  onAct,
  onOpenMessage,
}: {
  entry: DeckEntry;
  offset: number;
  position: number;
  total: number;
  now: number;
  working: boolean;
  flying: boolean;
  onBring: () => void;
  onStep: ((delta: number) => void) | null;
  onAct: (entry: DeckEntry) => void;
  onOpenMessage: (id: string) => void;
}) {
  const t = useT();
  const isFront = offset === 0;
  const nova = entry.group === "Draft";
  const look = LOOK[entry.group];
  const Icon = look.icon;
  const title = deckTitle(t, entry);

  return (
    <section
      aria-label={title}
      aria-hidden={isFront ? undefined : true}
      inert={!isFront || undefined}
      data-front={isFront}
      data-working={working}
      data-flying={flying}
      onClick={isFront ? undefined : onBring}
      style={{ "--tb-offset": offset, zIndex: 10 - offset } as CSSProperties}
      className={cn(
        "tb-card border-border bg-card relative col-start-1 row-start-1 flex min-h-75 flex-col rounded-lg border px-5.5 pt-5 pb-4",
        isFront ? "ui-lift cursor-default" : "cursor-pointer",
        nova && "tb-nova-card",
        "@max-2xl:min-h-70",
        !isFront && "@max-2xl:hidden",
      )}
    >
      {nova && (
        <>
          <span aria-hidden className="tb-nova-aura" />
          <span aria-hidden className="tb-nova-edge" />
        </>
      )}
      <div className="flex items-center justify-between">
        <span
          aria-hidden
          className={cn("flex size-8.5 items-center justify-center rounded-md", look.tile)}
        >
          {Icon === null ? (
            <NovaBadge size="md" spin={working ? "always" : "none"} />
          ) : (
            <Icon className="size-4" />
          )}
        </span>
        <span className="text-2xs text-foreground-subtle font-mono tabular-nums">
          {t("{0} / {1}", position, total)}
        </span>
      </div>
      <h3
        className={cn(
          "mt-4 text-2xl font-semibold transition-opacity duration-200",
          !isFront && "opacity-0",
        )}
      >
        {title}
      </h3>
      <p
        className={cn(
          "text-foreground-subtle mt-1 text-base transition-opacity duration-200",
          !isFront && "opacity-0",
        )}
      >
        {deckWhy(t, entry.group)}
      </p>
      {isFront && (
        <>
          <ul className="mt-3.5 flex flex-1 flex-col">
            {entry.items.slice(0, LISTED).map((item, index) => (
              <li key={item.messageId}>
                <DeckRow
                  item={item}
                  now={now}
                  nova={nova}
                  delay={nova ? 180 + index * 90 : 0}
                  onOpen={onOpenMessage}
                />
              </li>
            ))}
            {entry.count > LISTED && (
              <li className="text-foreground-subtle px-0 pt-1 text-xs">
                {t("{0, plural, one {+ # more} other {+ # more}}", entry.count - LISTED)}
              </li>
            )}
          </ul>
          <div
            className={cn(
              "border-border-subtle mt-3 flex items-center gap-1 border-t pt-3",
              nova && "border-accent-violet-border",
            )}
          >
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={t("Previous card")}
              disabled={onStep === null}
              onClick={() => onStep?.(-1)}
            >
              <ChevronLeftIcon className="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={t("Next card")}
              disabled={onStep === null}
              onClick={() => onStep?.(1)}
            >
              <ChevronRightIcon className="size-3.5" />
            </Button>
            <span className="flex-1" />
            {entry.estimateMinutes > 0 && (
              <span className="text-foreground-subtle mr-2 font-mono text-xs tabular-nums @max-2xl:hidden">
                {t("~{0} min", entry.estimateMinutes)}
              </span>
            )}
            <Button
              size="lg"
              disabled={working}
              onClick={() => onAct(entry)}
              className={cn(
                nova &&
                  "bg-accent-violet text-foreground-on-solid hover:bg-accent-violet-on-subtle active:bg-accent-violet-on-subtle dark:hover:bg-accent-violet-border",
              )}
            >
              {working && <Spinner className="size-3.5" />}
              {working ? t("Drafting…") : deckButton(t, entry.group)}
            </Button>
          </div>
        </>
      )}
    </section>
  );
});

const DeckRow = memo(function DeckRow({
  item,
  now,
  nova,
  delay,
  onOpen,
}: {
  item: DeckItem;
  now: number;
  nova: boolean;
  delay: number;
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
      style={delay > 0 ? { animationDelay: `${delay}ms` } : undefined}
      className={cn(
        "ui-focus-ring hover:bg-surface-hover -mx-2 grid w-[calc(100%+1rem)] grid-cols-[24px_minmax(0,1fr)_auto] items-center gap-2.5 rounded-md px-2 py-1.75 text-left",
        nova ? "tb-nova-item" : "animate-rise",
      )}
    >
      <span
        aria-hidden
        className={cn(
          "text-3xs text-foreground flex size-6 items-center justify-center rounded-full font-semibold",
          nova ? "tb-nova-avatar" : "bg-surface-active",
        )}
      >
        {getNameInitials(name, "?")}
      </span>
      <span className="flex min-w-0 flex-col leading-tight">
        <b className="truncate text-sm font-medium">{name}</b>
        <span className="text-foreground-subtle truncate text-xs">{next}</span>
      </span>
      {due !== null && (
        <span
          className={cn(
            "text-2xs font-mono tabular-nums",
            due.state === "over" ? "text-warning" : "text-foreground-subtle",
          )}
        >
          {dueText(t, due)}
        </span>
      )}
    </button>
  );
});

/** Cards that were acted on, as chips that open their first message again. */
export function DeckChips({
  chips,
  onOpen,
}: {
  chips: readonly DeckChip[];
  onOpen: (chip: DeckChip) => void;
}) {
  if (chips.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap justify-center gap-1.5">
      {chips.map((chip) => (
        <button
          key={chip.group}
          type="button"
          disabled={chip.busy}
          onClick={() => onOpen(chip)}
          className={cn(
            "ui-focus-ring animate-confirm inline-flex h-6.5 items-center gap-1.5 rounded-badge px-2.5 text-sm",
            chip.busy
              ? "bg-surface-hover text-foreground-muted"
              : "bg-success-subtle text-success-subtle-foreground hover:border-success-border border border-transparent",
          )}
        >
          {chip.busy ? (
            <Spinner className="size-3" />
          ) : (
            <CheckIcon aria-hidden className="size-2.75" strokeWidth={2.6} />
          )}
          {chip.label}
        </button>
      ))}
    </div>
  );
}
