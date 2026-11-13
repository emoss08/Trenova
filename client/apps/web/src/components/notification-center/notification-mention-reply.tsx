import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { useReplyToMention } from "@trenova/shared/hooks/use-notifications";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Notification } from "@trenova/shared/types/notification";
import { useState, type FormEvent, type MouseEvent } from "react";
import { notificationDataString, notificationRelatedId } from "./notification-registry";

function stopPropagation(event: MouseEvent) {
  event.stopPropagation();
}

/** What someone said when they mentioned the person, quoted under the row's title. */
export function MentionQuote({ notification }: { notification: Notification }) {
  if (!notification.message) return null;

  return (
    <p className="mt-1.5 rounded-md bg-surface-hover px-2 py-1.5 text-xs text-foreground">
      {notification.message}
    </p>
  );
}

/**
 * The reply to a mention, written where it is read. It goes to the shipment's thread and
 * mentions the author back, and the row is done once it is sent: answering is what the
 * notification asked for.
 */
export function MentionReply({
  notification,
  onReplied,
}: {
  notification: Notification;
  onReplied: () => void;
}) {
  const t = useT();

  const shipmentId = notificationRelatedId(notification, "shipmentId");
  const authorId = notificationDataString(notification, "authorId");
  const authorName = notificationDataString(notification, "authorName");
  const mentionToken = authorName ? `@${authorName}` : "";
  const placeholder = authorName ? t("Reply to {0}…", authorName) : t("Reply to the author…");

  const [value, setValue] = useState("");
  const reply = useReplyToMention();

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const text = value.trim();
    if (!text || !shipmentId || reply.isPending) return;

    // The mention is attached for the person: the token renders and highlights in the
    // thread, and the box holds only what they typed.
    const comment = mentionToken && !text.includes(mentionToken) ? `${mentionToken} ${text}` : text;

    reply.mutate(
      { shipmentId, comment, mentionUserId: authorId },
      {
        onSuccess: () => {
          setValue("");
          onReplied();
        },
      },
    );
  };

  if (!shipmentId) return null;

  return (
    <form className="mt-2 flex gap-1.5" onSubmit={submit} onClick={stopPropagation}>
      <Input
        value={value}
        placeholder={placeholder}
        aria-label={placeholder}
        className="h-6.5 flex-1 text-xs md:text-xs"
        disabled={reply.isPending}
        onChange={(event) => setValue(event.target.value)}
      />
      <Button
        type="submit"
        size="xs"
        className="h-6.5 text-xs"
        isLoading={reply.isPending}
        disabled={!value.trim()}
      >
        {t("Reply")}
      </Button>
    </form>
  );
}
