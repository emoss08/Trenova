import type { InboundMessageDetail, TriageAction } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { isSnoozed, senderName } from "@/lib/triage/message-state";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import {
  ArchiveIcon,
  ArrowDownIcon,
  ArrowUpIcon,
  CheckIcon,
  ClockIcon,
  PaperclipIcon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, kbdOnSolidClassName } from "@trenova/shared/components/ui/kbd";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixTime } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useEffect, useMemo, useState } from "react";
import { USER_FORMATTERS, actionCopy, nextOpenAction, snoozeText } from "../action-copy";
import { CLASSIFICATION_VARIANT, classificationLabel } from "../classification";
import { splitQuotedBody } from "../quoted-body";
import { useAnnouncePresence } from "../use-triage-presence";
import { useTriageActions } from "../use-triage-actions";

/**
 * The reader's contract with the page.
 *
 * The page owns which message is open and the folder's order; the reader owns
 * everything inside its column. It performs every action through
 * `useTriageActions()` and registers what ↵ and R do with
 * `useRegisterReaderCommands`, so the keys reach whatever the reader draws.
 */
export type ReaderProps = {
  messageId: string;
  /** Where the open message sits in the folder: 1-based, and how many the folder holds. */
  position: { index: number; total: number };
  onClose: () => void;
  /** J/K from the toolbar's arrows: the next (1) or previous (-1) message. */
  onStep: (direction: 1 | -1) => void;
};

/**
 * Registers what the open reader does for ↵ (do the next step) and R (reply)
 * while the calling component is mounted.
 */
export function useRegisterReaderCommands(commands: {
  doNextStep?: () => void;
  reply?: () => void;
}): void {
  const { doNextStep, reply } = commands;
  useEffect(() => {
    useTriageStore.getState().setReaderCommands({ doNextStep, reply });
    return () => useTriageStore.getState().setReaderCommands({});
  }, [doNextStep, reply]);
}

function ReaderSkeleton() {
  return (
    <div aria-busy="true" className="mx-auto flex w-full max-w-170 flex-col gap-4 px-8 pt-7">
      <Skeleton className="h-6 w-2/3" />
      <div className="flex items-center gap-3">
        <Skeleton className="size-7 rounded-full" />
        <Skeleton className="h-3.5 w-48" />
      </div>
      <Skeleton className="h-24 w-full" />
    </div>
  );
}

function ActionLine({
  message,
  action,
  number,
}: {
  message: InboundMessageDetail;
  action: TriageAction;
  number: number;
}) {
  const t = useT();
  const copy = actionCopy(t, action, message, USER_FORMATTERS);

  return (
    <li className="flex items-start gap-3 py-2">
      <span
        className={cn(
          "mt-0.5 flex size-4.5 shrink-0 items-center justify-center rounded-full border font-mono text-2xs",
          action.done
            ? "bg-success border-success text-foreground-on-solid"
            : "border-border-strong text-foreground-subtle",
        )}
      >
        {action.done ? <CheckIcon className="size-2.5" strokeWidth={3} /> : number}
      </span>
      <span className="flex min-w-0 flex-col">
        <span className={cn("text-sm font-medium", action.done && "text-foreground-muted")}>
          {copy.title}
        </span>
        <span className="text-foreground-subtle truncate text-xs">{copy.detail}</span>
      </span>
    </li>
  );
}

/**
 * One message, open: its subject, who sent it and when, what Nova read it as,
 * the steps it asks for, the words with the quoted thread folded away, and
 * its files. The actions panel, the floating action bar and the composer are
 * built on this column.
 */
export function Reader({ messageId, position, onClose, onStep }: ReaderProps) {
  const t = useT();
  const actions = useTriageActions();
  const openOverlay = useTriageStore((state) => state.openOverlay);
  const query = useQuery(queries.inbox.message(messageId));
  const message = query.data;
  const [showQuoted, setShowQuoted] = useState(false);
  useAnnouncePresence(messageId, "viewing");

  const body = useMemo(() => splitQuotedBody(message?.textBody ?? ""), [message?.textBody]);
  const next = message === undefined ? null : nextOpenAction(message.actions);

  useRegisterReaderCommands({
    doNextStep: useMemo(() => {
      if (message === undefined || next === null) {
        return undefined;
      }
      return () => {
        switch (next.kind) {
          case "Link":
            openOverlay({ kind: "link", messageId: message.id });
            return;
          case "BlockSender":
            actions.blockSender(message.id);
            return;
          case "File":
            if (next.attachment !== null && next.shipment !== null) {
              void actions
                .file(message.id, { attachmentId: next.attachment.id, shipmentId: next.shipment.id })
                .catch(() => undefined);
            }
            return;
          case "BuildLoad":
            void actions.draftLoad(message.id).catch(() => undefined);
            return;
          case "Reply":
            void actions.draftReply(message.id).catch(() => undefined);
            return;
          case "CheckStops":
          case "LogException":
            return;
        }
      };
    }, [message, next, actions, openOverlay]),
    reply: useMemo(
      () =>
        message === undefined || message.replyDraft !== null
          ? undefined
          : () => void actions.draftReply(message.id).catch(() => undefined),
      [message, actions],
    ),
  });

  if (query.isLoading || message === undefined) {
    return <ReaderSkeleton />;
  }

  const classification = message.classification ?? "Other";
  const waiting = message.needsReview;

  return (
    <article aria-label={message.subject} className="flex min-h-0 flex-1 flex-col">
      <div className="flex h-11 shrink-0 items-center gap-0.5 px-3">
        <Button size="icon-sm" variant="quiet" aria-label={t("Close")} onClick={onClose}>
          <XCloseIcon aria-hidden className="size-3.5" />
        </Button>
        <Button
          size="icon-sm"
          variant="quiet"
          aria-label={t("Previous message")}
          disabled={position.index <= 1}
          onClick={() => onStep(-1)}
        >
          <ArrowUpIcon aria-hidden className="size-3.5" />
        </Button>
        <Button
          size="icon-sm"
          variant="quiet"
          aria-label={t("Next message")}
          disabled={position.index >= position.total}
          onClick={() => onStep(1)}
        >
          <ArrowDownIcon aria-hidden className="size-3.5" />
        </Button>
        {position.total > 0 && position.index > 0 && (
          <span className="text-foreground-subtle mx-1 font-mono text-xs tabular-nums">
            {t("{0} / {1}", position.index, position.total)}
          </span>
        )}
      </div>

      <div className="scrollbar-overlay min-h-0 flex-1 overflow-y-auto">
        <div className="animate-rise mx-auto flex w-full max-w-170 flex-col gap-5 px-8 pt-5 pb-32">
          <h1 className="text-2xl font-semibold">
            {message.subject === "" ? t("(no subject)") : message.subject}
          </h1>
          <div className="flex items-center gap-3">
            <span className="bg-muted text-foreground-muted flex size-7 shrink-0 items-center justify-center rounded-full text-2xs font-semibold">
              {getNameInitials(senderName(message), "?")}
            </span>
            <span className="flex min-w-0 flex-1 items-baseline gap-2">
              <span className="truncate text-base font-medium">{senderName(message)}</span>
              {message.fromName.trim() !== "" && (
                <span className="text-foreground-subtle truncate text-sm">{message.fromAddress}</span>
              )}
            </span>
            <span className="text-foreground-subtle shrink-0 font-mono text-xs tabular-nums">
              {t(
                "{0} · {1}",
                formatUnixTime(message.receivedAt),
                formatSecondsAgo(Math.floor(Date.now() / 1000) - message.receivedAt),
              )}
            </span>
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            <Badge variant={CLASSIFICATION_VARIANT[classification]}>
              {classificationLabel(t, classification)}
            </Badge>
            {message.matchedShipment === null && message.matchedCustomer === null && (
              <Badge variant="neutral" appearance="outline">
                {t("Not linked")}
              </Badge>
            )}
            {message.owner !== null && (
              <Badge variant="neutral" appearance="outline">
                {message.owner.name}
              </Badge>
            )}
            {message.snooze !== null && isSnoozed(message) && (
              <Badge variant="warning">
                <ClockIcon aria-hidden className="size-3" />
                {snoozeText(t, message.snooze, USER_FORMATTERS)}
              </Badge>
            )}
          </div>

          {message.actions.length > 0 && (
            <section
              aria-label={t("Actions")}
              className="border-border-subtle bg-card rounded-lg border px-4 py-3"
            >
              <div className="text-foreground-subtle flex items-center justify-between text-xs font-medium">
                <span>{t("Actions")}</span>
                <span className="font-mono tabular-nums">
                  {t(
                    "{0}/{1}",
                    message.actions.filter((action) => action.done).length,
                    message.actions.length,
                  )}
                </span>
              </div>
              <ol className="mt-1">
                {message.actions.map((action, index) => (
                  <ActionLine key={action.id} message={message} action={action} number={index + 1} />
                ))}
              </ol>
            </section>
          )}

          <div className="text-base leading-relaxed wrap-anywhere whitespace-pre-wrap">{body.own}</div>
          {body.quoted !== "" && (
            <div>
              <Button size="xs" variant="ghost" onClick={() => setShowQuoted((value) => !value)}>
                {showQuoted ? t("Hide the quoted thread") : t("Show the quoted thread")}
              </Button>
              {showQuoted && (
                <div className="text-foreground-muted border-border mt-2 border-l-2 pl-3 text-sm whitespace-pre-wrap">
                  {body.quoted}
                </div>
              )}
            </div>
          )}

          {message.attachments.length > 0 && (
            <ul className="flex flex-wrap gap-1.5">
              {message.attachments.map((attachment) => (
                <li
                  key={attachment.id}
                  className="border-border text-foreground-muted flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs"
                >
                  <PaperclipIcon aria-hidden className="size-3" />
                  {attachment.fileName}
                </li>
              ))}
            </ul>
          )}

          {message.replyDraft !== null && (
            <section className="border-border-subtle rounded-lg border px-4 py-3">
              <div className="text-foreground-subtle mb-2 flex items-center gap-1.5 text-xs font-medium">
                {message.replyDraft.draftedByNova ? (
                  <>
                    <NovaMark className="size-3" />
                    {t("Drafted by Nova")}
                  </>
                ) : (
                  t("Reply draft")
                )}
              </div>
              <p className="text-sm whitespace-pre-wrap">{message.replyDraft.body}</p>
            </section>
          )}
        </div>
      </div>

      {waiting && (
        <div className="flex shrink-0 items-center justify-end gap-1.5 px-6 pb-4">
          <Button size="sm" variant="ghost" onClick={() => actions.ignore([message.id])}>
            <ArchiveIcon aria-hidden className="size-3.5" />
            {t("Ignore")}
            <Kbd>#</Kbd>
          </Button>
          <Button size="sm" onClick={() => actions.handle([message.id])}>
            <CheckIcon aria-hidden className="size-3.5" />
            {t("Mark handled")}
            <Kbd className={kbdOnSolidClassName}>E</Kbd>
          </Button>
        </div>
      )}
    </article>
  );
}
