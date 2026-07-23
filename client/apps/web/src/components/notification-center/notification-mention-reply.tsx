import { Button } from "@trenova/shared/components/ui/button";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useNotificationAction, useReplyToMention } from "@trenova/shared/hooks/use-notifications";
import type { Notification } from "@trenova/shared/types/notification";
import { CheckIcon, CornerUpLeftIcon } from "lucide-react";
import { useState, type KeyboardEvent, type MouseEvent } from "react";
import {
  getNotificationLink,
  notificationDataString,
  notificationRelatedId,
} from "./notification-registry";

function stopPropagation(event: MouseEvent) {
  event.stopPropagation();
}

export function MentionReply({
  notification,
  onNavigate,
}: {
  notification: Notification;
  onNavigate?: (link: string) => void;
}) {
  const shipmentId = notificationRelatedId(notification, "shipmentId");
  const authorId = notificationDataString(notification, "authorId");
  const authorName = notificationDataString(notification, "authorName");
  const authorDisplayName = authorName ?? "the author";
  const mentionToken = authorName ? `@${authorName}` : "";
  const link = getNotificationLink(notification);

  const [composing, setComposing] = useState(false);
  const [value, setValue] = useState("");
  const [sent, setSent] = useState(false);

  const reply = useReplyToMention();
  const markRead = useNotificationAction("read");

  const openThread = (event: MouseEvent) => {
    event.stopPropagation();
    if (link) onNavigate?.(link);
  };

  const openComposer = (event: MouseEvent) => {
    event.stopPropagation();
    setComposing(true);
  };

  const closeComposer = (event: MouseEvent) => {
    event.stopPropagation();
    setValue("");
    setComposing(false);
  };

  const submit = () => {
    const text = value.trim();
    if (!text || !shipmentId || reply.isPending) return;

    // Attach the mention automatically: prepend the @token so it renders and
    // highlights in the thread, while the compose box stays clean.
    const comment =
      mentionToken && !text.includes(mentionToken) ? `${mentionToken} ${text}` : text;

    reply.mutate(
      { shipmentId, comment, mentionUserId: authorId },
      {
        onSuccess: () => {
          setSent(true);
          setComposing(false);
          setValue("");
          markRead.mutate([notification.id]);
        },
      },
    );
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      submit();
    } else if (event.key === "Escape") {
      event.preventDefault();
      setValue("");
      setComposing(false);
    }
  };

  return (
    <div className="mt-1.5 flex flex-col gap-1.5">
      {notification.message && (
        <div className="rounded-md border-l-2 border-border bg-muted/50 px-2.5 py-1.5 text-2xs leading-relaxed text-muted-foreground">
          {notification.message}
        </div>
      )}

      {sent ? (
        <div className="flex items-center gap-1.5 text-2xs text-muted-foreground">
          <CheckIcon className="size-3 text-success" />
          <span>Reply sent</span>
          {link && (
            <>
              <span aria-hidden>·</span>
              <button
                type="button"
                className="font-medium text-brand hover:underline"
                onClick={openThread}
              >
                View conversation
              </button>
            </>
          )}
        </div>
      ) : composing ? (
        <div className="flex flex-col gap-1.5">
          <Textarea
            autoFocus
            value={value}
            minRows={2}
            maxRows={6}
            placeholder={`Reply to ${authorDisplayName}…`}
            className="text-2xs"
            onClick={stopPropagation}
            onChange={(event) => setValue(event.target.value)}
            onKeyDown={onKeyDown}
          />
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-muted-foreground/60">
              {mentionToken ? `${authorDisplayName} will be notified` : "Enter to send"}
            </span>
            <div className="flex items-center gap-1">
              <Button variant="ghost" size="xs" className="text-2xs" onClick={closeComposer}>
                Cancel
              </Button>
              <Button
                size="xs"
                className="text-2xs"
                isLoading={reply.isPending}
                disabled={!value.trim()}
                onClick={(event) => {
                  event.stopPropagation();
                  submit();
                }}
              >
                Send
              </Button>
            </div>
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-1">
          {shipmentId && (
            <Button variant="outline" size="xs" className="text-2xs" onClick={openComposer}>
              <CornerUpLeftIcon className="size-3" />
              Reply
            </Button>
          )}
          {link && (
            <Button
              variant="ghost"
              size="xs"
              className="text-2xs text-muted-foreground"
              onClick={openThread}
            >
              View conversation
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
