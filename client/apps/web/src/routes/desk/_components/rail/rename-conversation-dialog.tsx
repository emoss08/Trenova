import type { AssistantThread } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, useState } from "react";

/** The longest title the server keeps for a conversation. */
const MAX_TITLE = 200;

/**
 * Renaming a conversation from its row on the rail. Open while `thread` is
 * set. A name a person gives is kept: the model never names it again.
 */
export function RenameConversationDialog({
  thread,
  onRename,
  onClose,
}: {
  thread: AssistantThread | null;
  onRename: (thread: AssistantThread, title: string) => void;
  onClose: () => void;
}) {
  return (
    <Dialog open={thread !== null} onOpenChange={(open) => !open && onClose()}>
      {thread && (
        <RenameForm key={thread.id} thread={thread} onRename={onRename} onClose={onClose} />
      )}
    </Dialog>
  );
}

function RenameForm({
  thread,
  onRename,
  onClose,
}: {
  thread: AssistantThread;
  onRename: (thread: AssistantThread, title: string) => void;
  onClose: () => void;
}) {
  const t = useT();
  const id = useId();
  const [title, setTitle] = useState(thread.title);
  const trimmed = title.trim();
  const changed = trimmed !== "" && trimmed !== thread.title.trim();

  return (
    <DialogContent size="sm">
      <DialogHeader>
        <DialogTitle>{t("Rename conversation")}</DialogTitle>
        <DialogDescription>{t("The name it goes by on the rail and in search.")}</DialogDescription>
      </DialogHeader>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          event.stopPropagation();
          if (changed) {
            onRename(thread, trimmed);
          }
          onClose();
        }}
      >
        <label htmlFor={id} className="sr-only">
          {t("Conversation name")}
        </label>
        <Input
          id={id}
          value={title}
          maxLength={MAX_TITLE}
          autoFocus
          onFocus={(event) => event.currentTarget.select()}
          onChange={(event) => setTitle(event.currentTarget.value)}
        />
        <DialogFooter className="pt-4">
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Cancel")}
          </Button>
          <Button type="submit" disabled={trimmed === ""}>
            {t("Rename")}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  );
}
