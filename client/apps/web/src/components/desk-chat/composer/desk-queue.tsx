import type { ConversationQueue } from "@/components/assistant/use-conversation-queue";
import type { QueuedMessage } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { Button } from "@trenova/shared/components/ui/button";
import { DeskIcon } from "../desk-icons";
import { deskLinkClass, deskSmallIconClass } from "../desk-button-styles";

export type DeskQueueProps = {
  queue: ConversationQueue;
  /** A reply is being written: messages wait for it, and sending one steers it. */
  busy: boolean;
};

/**
 * What the person left for the conversation while its agent worked, above the
 * composer: each message in the order it will be sent, the ones handed to the
 * reply under way marked as such. A message can be edited, moved, removed or
 * sent at once — into the reply under way when there is one. When nothing is
 * being written and messages still wait (the last reply was stopped or
 * failed), the queue says it is paused and offers to send the next one.
 */
export function DeskQueue({ queue, busy }: DeskQueueProps) {
  const t = useT();
  const { items } = queue;
  const [folded, setFolded] = useState(false);
  const listRef = useRef<HTMLOListElement>(null);
  const count = useRef(items.length);

  // A message the person just added lands at the end of a list that may
  // already scroll; bring it into view so they see it was taken.
  useEffect(() => {
    const list = listRef.current;
    if (list && items.length > count.current) {
      list.scrollTop = list.scrollHeight;
    }
    count.current = items.length;
  }, [items.length]);

  if (items.length === 0) {
    return null;
  }

  const first = items[0];

  return (
    <section className="dk-wq" aria-label={t("Messages waiting to send")}>
      <div className="dk-wq-hd">
        <DeskIcon name={busy ? "clock" : "pause"} size={12} />
        <span className="dk-wq-tt">
          {busy
            ? t(
                "{0, plural, one {# message waits for this reply} other {# messages wait for this reply}}",
                items.length,
              )
            : t(
                "{0, plural, one {# message is waiting} other {# messages are waiting}}",
                items.length,
              )}
        </span>
        <span className="flex-1" />
        {!busy && (
          <Button
            variant="bare"
            size="bare"
            className={deskLinkClass}
            onClick={() => void queue.sendNow(first)}
          >
            {t("Send next")}
          </Button>
        )}
        <Button
          variant="quiet"
          size="bare"
          className={deskSmallIconClass}
          aria-expanded={!folded}
          aria-label={folded ? t("Show the waiting messages") : t("Hide the waiting messages")}
          title={folded ? t("Show") : t("Hide")}
          onClick={() => setFolded((value) => !value)}
        >
          <DeskIcon name={folded ? "down" : "up"} size={12} />
        </Button>
      </div>
      <ol className="dk-wq-list dk-qi-list" ref={listRef} hidden={folded}>
        {items.map((item, index) => (
          <DeskQueueItem
            key={item.id}
            item={item}
            index={index}
            last={index === items.length - 1}
            busy={busy}
            queue={queue}
          />
        ))}
      </ol>
    </section>
  );
}

type DeskQueueItemProps = {
  item: QueuedMessage;
  index: number;
  last: boolean;
  busy: boolean;
  queue: ConversationQueue;
};

function DeskQueueItem({ item, index, last, busy, queue }: DeskQueueItemProps) {
  const t = useT();
  const [draft, setDraft] = useState<string | null>(null);
  // A message handed to the reply under way is the reply's now; it can be
  // taken back only by stopping the reply.
  const handed = busy && item.steer;
  const editing = draft !== null;
  const records = item.request.mentions.length;
  const files = item.request.attachmentDocumentIds.length;

  const save = async () => {
    if (draft === null) {
      return;
    }
    const content = draft.trim();
    if (content === "" || content === item.content) {
      setDraft(null);
      return;
    }
    if (await queue.edit(item, content)) {
      setDraft(null);
    }
  };

  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      setDraft(null);
    } else if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      void save();
    }
  };

  return (
    <li className={cn("dk-wq-it dk-qi", handed && "dk-wq-handed")}>
      <span
        className="dk-qi-n"
        title={
          handed
            ? t("Steering: the agent reads it at its next step")
            : index === 0
              ? t("Sends next")
              : t("Sends after the one above")
        }
      >
        {handed ? <DeskIcon name="enter" size={11} stroke={2} /> : index + 1}
      </span>
      {editing ? (
        <div className="dk-wq-ed">
          <textarea
            className="ui-field dk-wq-ta"
            value={draft}
            rows={2}
            autoFocus
            aria-label={t("Edit the waiting message")}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={onKeyDown}
          />
          <div className="dk-wq-eda">
            <Button
              variant="bare"
              size="bare"
              className={deskLinkClass}
              onClick={() => setDraft(null)}
            >
              {t("Cancel")}
            </Button>
            <Button
              variant="bare"
              size="bare"
              className={deskLinkClass}
              onClick={() => void save()}
            >
              {t("Save")}
            </Button>
          </div>
        </div>
      ) : (
        <p className="dk-wq-tx" title={item.content}>
          {item.content}
        </p>
      )}
      {(records > 0 || files > 0) && !editing && (
        <span className="dk-wq-meta">
          {records > 0 && t("{0, plural, one {# record} other {# records}}", records)}
          {records > 0 && files > 0 && " · "}
          {files > 0 && t("{0, plural, one {# file} other {# files}}", files)}
        </span>
      )}
      {!editing && !handed && (
        <span className="dk-wq-acts">
          <Button
            variant="quiet"
            size="bare"
            className={deskSmallIconClass}
            title={t("Edit")}
            aria-label={t("Edit the waiting message")}
            onClick={() => setDraft(item.content)}
          >
            <DeskIcon name="edit" size={13} />
          </Button>
          <Button
            variant="quiet"
            size="bare"
            className={deskSmallIconClass}
            title={t("Move up")}
            aria-label={t("Move the message up")}
            disabled={index === 0}
            onClick={() => void queue.move(item, -1)}
          >
            <DeskIcon name="up" size={13} />
          </Button>
          <Button
            variant="quiet"
            size="bare"
            className={deskSmallIconClass}
            title={t("Move down")}
            aria-label={t("Move the message down")}
            disabled={last}
            onClick={() => void queue.move(item, 1)}
          >
            <DeskIcon name="down" size={13} />
          </Button>
          {!(busy && files > 0) && (
            <Button
              variant="quiet"
              size="bare"
              className={deskSmallIconClass}
              title={busy ? t("Steer the reply with it now") : t("Send it now")}
              aria-label={busy ? t("Steer the reply with it now") : t("Send it now")}
              onClick={() => void queue.sendNow(item)}
            >
              <DeskIcon name="enter" size={13} />
            </Button>
          )}
          <Button
            variant="quiet"
            size="bare"
            className={deskSmallIconClass}
            title={t("Remove")}
            aria-label={t("Remove the waiting message")}
            onClick={() => void queue.remove(item)}
          >
            <DeskIcon name="trash" size={13} />
          </Button>
        </span>
      )}
    </li>
  );
}
