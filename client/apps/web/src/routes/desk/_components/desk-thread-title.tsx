import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useRef, useState, type KeyboardEvent } from "react";

/** The longest name the server keeps for a conversation. */
const MAX_TITLE_LENGTH = 200;

/**
 * The open conversation's name in the top bar. A click turns it into a
 * field: Enter or leaving the field keeps the new name, Escape keeps the old
 * one. A blank or unchanged name is not sent.
 *
 * Mounted per conversation, so moving to another one drops an unfinished
 * edit rather than carrying it over.
 */
export function DeskThreadTitle({
  thread,
  onRename,
}: {
  thread: AssistantThread;
  onRename: (title: string) => void;
}) {
  const t = useT();
  const [editing, setEditing] = useState(false);
  // Enter and Escape end the edit before the field loses focus; the blur
  // that follows must not end it a second time.
  const settled = useRef(false);

  const finish = (value: string | null) => {
    if (settled.current) {
      return;
    }
    settled.current = true;
    setEditing(false);
    const title = value?.trim() ?? "";
    if (title !== "" && title !== thread.title) {
      onRename(title);
    }
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      finish(event.currentTarget.value);
    } else if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      finish(null);
    }
  };

  if (editing) {
    return (
      <input
        className="dk-ttl-in"
        // oxlint-disable-next-line jsx-a11y/no-autofocus -- the name turned into this field on purpose
        autoFocus
        defaultValue={thread.title}
        maxLength={MAX_TITLE_LENGTH}
        aria-label={t("Conversation name")}
        onFocus={(event) => event.target.select()}
        onBlur={(event) => finish(event.target.value)}
        onKeyDown={onKeyDown}
      />
    );
  }

  return (
    <Button
      variant="bare"
      size="bare"
      className="dk-ttl-t"
      onClick={() => {
        settled.current = false;
        setEditing(true);
      }}
    >
      <span>{thread.title || t("Untitled conversation")}</span>
    </Button>
  );
}
