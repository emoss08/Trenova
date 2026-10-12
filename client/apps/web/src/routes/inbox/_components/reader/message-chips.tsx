import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { isAwaitingReply, isSnoozed } from "@/lib/triage/message-state";
import { useTriageStore } from "@/stores/triage-store";
import {
  CheckIcon,
  ChevronDownIcon,
  ClockIcon,
  Link01Icon,
  Send01Icon,
} from "@trenova/shared/components/icons";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ReactNode } from "react";
import { USER_FORMATTERS, snoozeText } from "../action-copy";
import {
  CLASSIFICATION_DOT,
  INBOUND_CLASSIFICATIONS,
  classificationLabel,
} from "../classification";
import { PersonAvatar } from "../person-avatar";
import { useTriageActions } from "../use-triage-actions";
import { KindTag } from "./kind-tag";
import { openMenuFrom } from "./menu-anchor";

const CHIP =
  "inline-flex h-6 items-center gap-1.5 rounded-full pr-2.5 pl-2 text-sm whitespace-nowrap transition-colors";

function Chip({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <span className={cn(CHIP, "bg-surface-hover text-foreground-muted", className)}>
      {children}
    </span>
  );
}

/**
 * What the message is and where it stands, as chips under the sender: its
 * kind (which opens "it's actually…"), no record linked, who has it, how long
 * it sleeps, and that it waits on the sender.
 */
export function MessageChips({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const actions = useTriageActions();
  const meId = useAuthStore((state) => state.user?.id ?? null);
  const kind = message.classification;

  const openCorrection = (button: HTMLElement) =>
    openMenuFrom(button, {
      title:
        kind === null
          ? t("What kind of message is this?")
          : t(
              "Nova read this as {0}. It's actually…",
              toSentenceFragment(classificationLabel(t, kind)),
            ),
      entries: INBOUND_CLASSIFICATIONS.filter((other) => other !== kind).map((other) => ({
        id: other,
        label: classificationLabel(t, other),
        leading: (
          <span aria-hidden className={cn("size-2 rounded-sm", CLASSIFICATION_DOT[other])} />
        ),
        onSelect: () => actions.correctKind(message.id, other),
      })),
    });

  const unlinked =
    message.matchedShipment === null &&
    message.matchedCustomer === null &&
    message.status !== "Quarantined";

  return (
    <div className="-mt-1 flex flex-wrap items-center gap-1.5">
      {message.novaRead ? (
        <button
          type="button"
          aria-haspopup="menu"
          title={t("Read wrong? Correct it")}
          onClick={(event) => openCorrection(event.currentTarget)}
          className={cn(
            CHIP,
            "bg-surface-hover text-foreground-muted hover:bg-surface-active hover:text-foreground ui-focus-ring group pr-2",
          )}
        >
          <KindTag kind={kind} />
          <ChevronDownIcon
            aria-hidden
            className="text-foreground-subtle group-hover:text-foreground size-2.5"
          />
        </button>
      ) : (
        <Chip>
          <KindTag kind={kind} />
        </Chip>
      )}
      {message.classificationCorrected && (
        <Chip className="bg-success-subtle text-success-subtle-foreground animate-confirm">
          <CheckIcon aria-hidden className="size-3" strokeWidth={2.4} />
          {t("Corrected — Nova will remember")}
        </Chip>
      )}
      {unlinked && (
        <button
          type="button"
          onClick={() =>
            useTriageStore.getState().openOverlay({ kind: "link", messageId: message.id })
          }
          className={cn(
            CHIP,
            "border-border-strong text-foreground-muted hover:bg-surface-hover hover:text-foreground ui-focus-ring border border-dashed",
          )}
        >
          <Link01Icon aria-hidden className="size-3" />
          {t("Not linked")}
        </button>
      )}
      {message.owner !== null && message.needsReview && (
        <Chip>
          <PersonAvatar name={message.owner.name} size="xs" />
          {message.owner.id === meId ? t("Assigned to you") : message.owner.name}
        </Chip>
      )}
      {message.snooze !== null && isSnoozed(message) && (
        <Chip className="bg-warning-subtle text-warning-subtle-foreground">
          <ClockIcon aria-hidden className="size-3" />
          {snoozeText(t, message.snooze, USER_FORMATTERS)}
        </Chip>
      )}
      {isAwaitingReply(message) && (
        <Chip className="bg-brand-subtle text-brand-subtle-foreground">
          <Send01Icon aria-hidden className="size-3" />
          {t("Waiting on them")}
        </Chip>
      )}
    </div>
  );
}

/** "Handled by you · Oct 7, 1:39 PM — note", with Reopen, on a settled message. */
export function SettledReceipt({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const rt = useRichT();
  const actions = useTriageActions();
  const meId = useAuthStore((state) => state.user?.id ?? null);
  if (message.needsReview || (message.status !== "Actioned" && message.status !== "Ignored")) {
    return null;
  }

  const reviewer = message.reviewer;
  const by =
    reviewer !== null
      ? reviewer.id === meId
        ? t("you")
        : reviewer.name
      : message.handledByNova
        ? t("Nova")
        : t("a teammate");
  const line = message.status === "Actioned" ? t("Handled by {0}", by) : t("Ignored by {0}", by);
  const note = message.reviewNote.trim();
  const stamped =
    message.reviewedAt === null
      ? line
      : t("{0} · {1}", line, formatUnixDateTimeShort(message.reviewedAt));

  return (
    <div className="text-success flex items-start gap-2 text-base leading-normal">
      <CheckIcon aria-hidden className="mt-1 size-3 shrink-0" strokeWidth={2.4} />
      <span className="min-w-0 flex-1">
        {note === ""
          ? stamped
          : rt(
              "{0}<n> — {1}</n>",
              { n: (chunks) => <span className="text-foreground-subtle">{chunks}</span> },
              stamped,
              note,
            )}
      </span>
      <button
        type="button"
        onClick={() => actions.reopen([message.id])}
        className="text-foreground-subtle hover:text-foreground ui-focus-ring shrink-0 rounded-sm text-sm"
      >
        {t("Reopen")}
      </button>
    </div>
  );
}
