import { useUserTimezone } from "@/hooks/use-user-timezone";
import { useTriageStore } from "@/stores/triage-store";
import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import {
  AlertTriangleIcon,
  ArchiveIcon,
  CheckIcon,
  DotsHorizontalIcon,
  Edit05Icon,
  HandIcon,
  HelpCircleIcon,
  Link01Icon,
  Send01Icon,
  ShieldAlertIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, kbdOnSolidClassName } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { classificationLabel } from "../classification";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { USER_FORMATTERS, actionCopy, ruleActionPhrase } from "../action-copy";
import { tomorrowMorning } from "../snooze-times";
import { useTriageActions } from "../use-triage-actions";
import {
  actionBarMode,
  whyRows,
  type ActionBarMode,
  type PrimaryKind,
  type WhyNote,
  type WhyRow,
} from "./action-bar-mode";
import { openMenuFrom } from "./menu-anchor";
import { useReaderPanel } from "./reader-panel";

function primaryLabel(t: TranslateFn, primary: PrimaryKind): string {
  switch (primary) {
    case "doIt":
      return t("Do it");
    case "novaDoIt":
      return t("Have Nova do it");
    case "reviewReply":
      return t("Review reply");
    case "reviewDraft":
      return t("Review draft");
  }
}

function noteText(t: TranslateFn, note: WhyNote, mailbox: string): string {
  switch (note) {
    case "notRead":
      return t("Nova was off when it arrived, so nothing was read from it.");
    case "alwaysReview":
      return t("Everything on {0} waits for a person.", mailbox);
    case "loadByHand":
      return t("Creating a load is not something Nova does on its own.");
    case "filingWaits":
      return t("Filing a document on a shipment waits for a person on {0}.", mailbox);
    case "replyByHand":
      return t("Nova drafts replies but never sends one on its own.");
    case "moneyByHand":
      return t("A charge is a person's call.");
  }
}

function WhyRowView({ row }: { row: WhyRow }) {
  const t = useT();
  const line = (icon: ReactNode, text: ReactNode) => (
    <div className="text-foreground-muted flex items-center gap-2.5 p-1.5 text-base leading-normal">
      {icon}
      <span className="min-w-0">{text}</span>
    </div>
  );

  switch (row.kind) {
    case "failure":
      return line(
        row.held ? (
          <ShieldAlertIcon aria-hidden className="text-foreground-subtle size-3.25 shrink-0" />
        ) : (
          <AlertTriangleIcon aria-hidden className="text-foreground-subtle size-3.25 shrink-0" />
        ),
        row.text,
      );
    case "confidence": {
      const percent = Math.round(row.confidence * 100);
      const bar = Math.round(row.bar * 100);
      return line(
        <span
          aria-hidden
          className="bg-surface-active relative inline-block h-1 w-18 shrink-0 rounded-full"
        >
          <span
            className={cn(
              "absolute inset-y-0 left-0 rounded-full",
              row.under ? "bg-warning" : "bg-foreground-muted",
            )}
            style={{ width: `${percent}%` }}
          />
          <span
            className="bg-foreground absolute -top-0.75 h-2.5 w-0.5 rounded-full"
            style={{ left: `${bar}%` }}
          />
        </span>,
        <>
          <span className="text-foreground font-medium">{t("{0}% sure", percent)}</span>
          {row.under && t(" — under this mailbox's {0}% bar", bar)}
        </>,
      );
    }
    case "note":
      return line(
        <HandIcon aria-hidden className="text-foreground-subtle size-3.25 shrink-0" />,
        noteText(t, row.note, row.mailbox),
      );
    case "match":
      return line(
        <Link01Icon aria-hidden className="text-foreground-subtle size-3.25 shrink-0" />,
        row.text,
      );
  }
}

function WhyButton({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const next = message.actions.find((step) => !step.done) ?? null;
  const rows = whyRows(message, next);
  if (rows.length === 0) {
    return null;
  }

  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type="button"
            aria-label={t("Why it's waiting")}
            className="text-foreground-subtle hover:bg-surface-hover hover:text-foreground data-popup-open:bg-surface-hover data-popup-open:text-foreground ui-focus-ring flex size-5.5 shrink-0 items-center justify-center rounded-full"
          />
        }
      >
        <HelpCircleIcon aria-hidden className="size-3.25" />
      </PopoverTrigger>
      <PopoverContent side="top" align="start" sideOffset={12} className="w-90 gap-0.5 p-2">
        {rows.map((row, index) => (
          <WhyRowView key={`${row.kind}:${index}`} row={row} />
        ))}
      </PopoverContent>
    </Popover>
  );
}

function SheetButton({ done, total, onOpen }: { done: number; total: number; onOpen: () => void }) {
  const t = useT();
  return (
    <button
      type="button"
      onClick={onOpen}
      className="bg-surface-hover ui-press ui-focus-ring inline-flex h-7.5 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium"
    >
      <span className="text-foreground-subtle font-mono tabular-nums">{t("{0}/{1}", done, total)}</span>
      {t("Actions")}
    </button>
  );
}

function teachTitle(
  t: TranslateFn,
  message: InboundMessageDetail,
  mode: Extract<ActionBarMode, { kind: "done" }>,
): string {
  if (mode.teach === null || message.classification === null) {
    return "";
  }

  return t(
    "Rule: {0} from {1} → {2}",
    toSentenceFragment(classificationLabel(t, message.classification)),
    message.senderDomain,
    ruleActionPhrase(t, mode.teach),
  );
}

/**
 * The bar floating at the foot of the reader: the next step and the button
 * that does it, or what is left once every step is done — handle it (and
 * teach Nova to do it alone next time), wait for their reply, ignore it, or
 * handle it with a note. A held message offers "Not phishing" or Ignore; one
 * waiting on the sender says until when.
 */
export function ActionBar({
  message,
  onPrimary,
  onNext,
}: {
  message: InboundMessageDetail;
  /** ↵ and the primary button: submit the open step, or start the next one. */
  onPrimary: () => void;
}) {
  const t = useT();
  const actions = useTriageActions();
  const panel = useReaderPanel();
  const timeZone = useUserTimezone();
  const [noting, setNoting] = useState(false);
  const [note, setNote] = useState("");
  const [teach, setTeach] = useState(false);
  const mode = actionBarMode(message, panel.state.drafting);
  if (mode.kind === "none") {
    return null;
  }

  const ids = [message.id];
  const total = message.actions.length;
  const doneCount = message.actions.filter((step) => step.done).length;
  const teachRule = mode.kind === "done" && teach ? mode.teach : null;
  const handle = () =>
    actions.handle(ids, {
      note: note.trim() === "" ? undefined : note.trim(),
      teach: teachRule ?? undefined,
    });
  const waitForReply = () =>
    actions.waitForReply(message.id, tomorrowMorning(Math.floor(Date.now() / 1000), timeZone));
  const canWait =
    message.awaitingReply === null &&
    message.replies.some((reply) => reply.status === "Sent");

  const openMore = (button: HTMLElement) =>
    openMenuFrom(
      button,
      {
        label: t("More"),
        entries: [
          {
            id: "note",
            label: t("Handled, with a note"),
            icon: Edit05Icon,
            onSelect: () => setNoting(true),
          },
          {
            id: "ignore",
            label: t("Ignore"),
            icon: ArchiveIcon,
            shortcut: "#",
            onSelect: () => actions.ignore(ids),
          },
          canWait && {
            id: "wait",
            label: t("Wait for their reply"),
            icon: Send01Icon,
            onSelect: waitForReply,
          },
        ],
      },
      { align: "end", side: "above" },
    );

  let lead: ReactNode = null;
  let main: ReactNode;
  if (noting) {
    main = (
      <>
        <input
          autoFocus
          value={note}
          aria-label={t("What you did")}
          placeholder={t("What you did, so nobody does it twice")}
          onChange={(event) => setNote(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              event.stopPropagation();
              handle();
              setNoting(false);
            } else if (event.key === "Escape") {
              event.stopPropagation();
              setNoting(false);
            }
          }}
          className="placeholder:text-foreground-subtle h-8.5 min-w-0 flex-1 bg-transparent px-2 text-base outline-none"
        />
        <Button size="sm" variant="ghost" onClick={() => setNoting(false)}>
          {t("Cancel")}
        </Button>
        <Button
          size="sm"
          onClick={() => {
            handle();
            setNoting(false);
          }}
        >
          {t("Mark handled")}
        </Button>
      </>
    );
  } else if (mode.kind === "waiting") {
    lead = (
      <>
        <span className="bg-brand-subtle text-brand-subtle-foreground flex size-5 shrink-0 items-center justify-center rounded-md">
          <Send01Icon aria-hidden className="size-2.75" />
        </span>
        <span className="truncate text-base">
          {t(
            "Waiting on {0} · back {1} if no reply",
            mode.firstName,
            USER_FORMATTERS.when(mode.until),
          )}
        </span>
      </>
    );
    main = (
      <>
        <Button size="sm" variant="ghost" onClick={() => actions.stopWaiting(message.id)}>
          {t("Stop waiting")}
        </Button>
        <Button className="h-8.5 rounded-lg" onClick={handle}>
          {t("Mark handled")}
          <Kbd className={kbdOnSolidClassName}>E</Kbd>
        </Button>
      </>
    );
  } else if (mode.kind === "held") {
    lead = (
      <>
        <span className="bg-danger-subtle text-danger-subtle-foreground flex size-5 shrink-0 items-center justify-center rounded-md">
          <ShieldAlertIcon aria-hidden className="size-2.75" />
        </span>
        <span className="truncate text-base">{t("Held back — looks like phishing")}</span>
        <WhyButton message={message} />
      </>
    );
    main = (
      <>
        <Button
          size="sm"
          variant="ghost"
          onClick={() =>
            useTriageStore.getState().openOverlay({ kind: "link", messageId: message.id })
          }
        >
          {t("Not phishing")}
        </Button>
        <Button
          className="h-8.5 rounded-lg"
          onClick={() =>
            actions.ignore(ids, { note: t("Phishing — sender not on the allow list.") })
          }
        >
          {t("Ignore")}
          <Kbd className={kbdOnSolidClassName}>↵</Kbd>
        </Button>
      </>
    );
  } else if (mode.kind === "next") {
    const copy = actionCopy(t, mode.action, message, USER_FORMATTERS);
    const nova = mode.primary === "novaDoIt";
    lead = (
      <>
        <span
          className={cn(
            "bg-surface-hover text-foreground-muted inline-flex h-5.5 shrink-0 items-baseline rounded-full px-1.75 font-mono text-xs leading-5.5 tabular-nums",
            mode.busy && "animate-breathe",
          )}
        >
          {mode.position}
          <span className="mx-px opacity-50">/</span>
          {mode.total}
        </span>
        {mode.busy ? (
          <TextShimmer as="span" className="truncate text-base">
            {t("Nova is drafting…")}
          </TextShimmer>
        ) : (
          <span className="truncate text-base">{copy.title}</span>
        )}
        {mode.explain && <WhyButton message={message} />}
      </>
    );
    main = (
      <>
        <Button
          variant={nova ? "nova" : "default"}
          disabled={mode.busy}
          className="h-8.5 rounded-lg"
          onClick={onPrimary}
        >
          {primaryLabel(t, mode.primary)}
          <Kbd className={cn("@max-[720px]/reader:hidden", !nova && kbdOnSolidClassName)}>↵</Kbd>
        </Button>
        <Button variant="ghost" className="h-8.5 rounded-lg" aria-label={t("Mark handled (E)")} onClick={handle}>
          <CheckIcon aria-hidden className="size-3.25" strokeWidth={2} />
          <span className="@max-[720px]/reader:hidden">{t("Done")}</span>
        </Button>
      </>
    );
  } else {
    const summary =
      mode.total === 0
        ? t("Nothing to do")
        : mode.only !== null
          ? actionCopy(t, mode.only, message, USER_FORMATTERS).detail
          : t("All {0} done", mode.total);
    lead = (
      <>
        <span className="bg-success text-foreground-on-solid animate-confirm flex size-5 shrink-0 items-center justify-center rounded-full">
          <CheckIcon aria-hidden className="size-3" strokeWidth={2.6} />
        </span>
        <span className="truncate text-base">{summary}</span>
        {mode.teach !== null && (
          <button
            type="button"
            role="checkbox"
            aria-checked={teach}
            title={teachTitle(t, message, mode)}
            onClick={() => setTeach((value) => !value)}
            className={cn(
              "hover:bg-surface-hover hover:text-foreground ui-focus-ring ml-1.5 inline-flex shrink-0 items-center gap-1.75 rounded-md py-1 pr-2 pl-1.5 text-sm whitespace-nowrap",
              teach ? "text-accent-violet" : "text-foreground-subtle",
            )}
          >
            <span
              aria-hidden
              className={cn(
                "flex size-3.5 shrink-0 items-center justify-center rounded-xs border-[1.5px] transition-colors",
                teach
                  ? "border-accent-violet bg-accent-violet text-canvas"
                  : "border-border-strong",
              )}
            >
              {teach && <CheckIcon className="size-2.25" strokeWidth={3} />}
            </span>
            <span className="@max-[760px]/reader:sr-only">{t("Do this on its own next time")}</span>
          </button>
        )}
      </>
    );
    main = (
      <>
        {mode.waitForReply && canWait && (
          <Button variant="ghost" className="h-8.5 rounded-lg" onClick={waitForReply}>
            <Send01Icon aria-hidden className="size-3" />
            <span className="@max-[720px]/reader:hidden">{t("Wait for reply")}</span>
          </Button>
        )}
        <Button className="h-8.5 rounded-lg" onClick={handle}>
          {t("Mark handled")}
          <Kbd className={cn("@max-[720px]/reader:hidden", kbdOnSolidClassName)}>E</Kbd>
        </Button>
      </>
    );
  }

  const key = noting
    ? "noting"
    : mode.kind === "next"
      ? mode.action.id
      : mode.kind;

  return (
    <div
      key={key}
      role="toolbar"
      aria-label={t("Next step")}
      className={cn(
        "bg-raised ui-lift animate-rise pointer-events-auto flex min-h-13 w-full max-w-155 items-center gap-3.5 rounded-xl py-2 pr-2 pl-3.5",
        noting && "pl-2",
      )}
    >
      {panel.compact && !noting && total > 0 && (
        <SheetButton
          done={doneCount}
          total={total}
          onOpen={() => panel.dispatch({ type: "sheet", open: true })}
        />
      )}
      {lead !== null && !(panel.compact && total > 0) && (
        <div className="flex min-w-22 flex-1 items-center gap-2.25">{lead}</div>
      )}
      {panel.compact && total > 0 && <span className="flex-1" />}
      <div className={cn("flex shrink-0 items-center gap-1", noting && "min-w-0 flex-1")}>
        {main}
        {!noting && (
          <Button
            size="icon-sm"
            variant="quiet"
            aria-haspopup="menu"
            aria-label={t("More")}
            onClick={(event) => openMore(event.currentTarget)}
          >
            <DotsHorizontalIcon aria-hidden className="size-4" />
          </Button>
        )}
      </div>
    </div>
  );
}
