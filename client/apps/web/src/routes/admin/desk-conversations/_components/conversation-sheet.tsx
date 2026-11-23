import {
  openDeskConversation,
  type DeskConversationRow,
  type OpenedDeskConversation,
} from "@/lib/graphql/desk-transcripts";
import { downloadDeskConversation } from "@/lib/desk-transcripts";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { toast } from "@trenova/shared/components/ui/toast";
import { Download01Icon, EyeIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useMutation } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { originLabel } from "./conversation-scope";
import { ConversationSheetSkeleton } from "./conversation-sheet-skeleton";
import { ConversationTranscript } from "./conversation-transcript";

type ConversationSheetProps = {
  /** The conversation to read; null closes the sheet. */
  row: DeskConversationRow | null;
  canExport: boolean;
  onClose: () => void;
};

/**
 * Another person's conversation, read-only: whose it is, that opening it was
 * recorded, and what was said. Nothing here can post, rename, pin, snooze,
 * delete or continue it; Download is the only action, for holders of Export.
 */
export function ConversationSheet({ row, canExport, onClose }: ConversationSheetProps) {
  const t = useT();
  const opened = useMutation({ mutationFn: openDeskConversation });
  const openedFor = useRef<string | null>(null);
  const { mutate, reset } = opened;

  useEffect(() => {
    if (row === null) {
      openedFor.current = null;
      reset();
      return;
    }
    if (openedFor.current === row.id) {
      return;
    }
    openedFor.current = row.id;
    mutate(row.id);
  }, [row, mutate, reset]);

  const title = row?.title || t("Untitled conversation");

  return (
    <Sheet open={row !== null} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-3xl">
        <SheetHeader className="gap-1 pr-12">
          <SheetTitle className="truncate">{title}</SheetTitle>
          {row ? (
            <SheetDescription>
              {t("{0}'s conversation with {1}", row.ownerName, row.agentName)}
            </SheetDescription>
          ) : null}
        </SheetHeader>
        {row ? (
          <SheetBody
            row={row}
            canExport={canExport}
            state={opened.status}
            data={opened.data ?? null}
            error={opened.error}
            onRetry={() => mutate(row.id)}
          />
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

function SheetBody({
  row,
  canExport,
  state,
  data,
  error,
  onRetry,
}: {
  row: DeskConversationRow;
  canExport: boolean;
  state: "idle" | "pending" | "success" | "error";
  data: OpenedDeskConversation | null;
  error: Error | null;
  onRetry: () => void;
}) {
  const t = useT();
  const [downloading, setDownloading] = useState(false);

  const download = async () => {
    if (downloading) {
      return;
    }
    setDownloading(true);
    try {
      await downloadDeskConversation(row.id);
    } catch (downloadError) {
      toast.error(t("The transcript could not be downloaded"), {
        description: graphQLErrorMessage(downloadError, t("Try again shortly.")),
      });
    } finally {
      setDownloading(false);
    }
  };

  if (state === "error") {
    const kind = describeError(error).kind;
    return (
      <div className="flex flex-col gap-3 px-4 pb-4">
        <Alert variant="destructive" size="sm">
          <AlertDescription>
            {kind === "forbidden"
              ? t("You no longer have permission to read other people's Desk conversations.")
              : kind === "not-found"
                ? t("This conversation no longer exists.")
                : t("This conversation could not be opened. Try again shortly.")}
          </AlertDescription>
        </Alert>
        {kind !== "forbidden" && kind !== "not-found" ? (
          <div>
            <Button type="button" size="sm" variant="outline" onClick={onRetry}>
              {t("Try again")}
            </Button>
          </div>
        ) : null}
      </div>
    );
  }

  if (state !== "success" || data === null) {
    return <ConversationSheetSkeleton />;
  }

  const conversation = data.conversation;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 px-4 pb-4">
      <Alert variant="info" size="sm">
        <EyeIcon />
        <AlertDescription>
          {t(
            "You are reading {0}'s conversation as they saw it. Opening it was recorded in the audit log; nothing here changes it for them.",
            conversation.ownerName,
          )}
        </AlertDescription>
      </Alert>
      <div className="flex items-start gap-4">
        <DescriptionList className="flex-1">
          <DescriptionItem label={t("Person")}>{conversation.ownerName}</DescriptionItem>
          <DescriptionItem label={t("Agent")}>{conversation.agentName}</DescriptionItem>
          <DescriptionItem label={t("Where it began")}>
            {originLabel(t, conversation.origin)}
          </DescriptionItem>
          <DescriptionItem label={t("Messages")} numeric>
            {conversation.messageCount}
          </DescriptionItem>
          <DescriptionItem label={t("Started")}>
            {formatUnixDateTimeMedium(conversation.startedAt)}
          </DescriptionItem>
          <DescriptionItem label={t("Last message")}>
            {conversation.lastMessageAt ? (
              formatUnixDateTimeMedium(conversation.lastMessageAt)
            ) : (
              <DescriptionEmpty />
            )}
          </DescriptionItem>
        </DescriptionList>
        {canExport ? (
          <Button type="button" size="sm" isLoading={downloading} onClick={() => void download()}>
            <Download01Icon className="size-3.5" />
            {t("Download")}
          </Button>
        ) : null}
      </div>
      <ScrollArea className="min-h-0 flex-1">
        <ConversationTranscript
          conversationId={conversation.id}
          transcript={data.transcript}
          ownerName={conversation.ownerName}
          agentName={conversation.agentName}
        />
      </ScrollArea>
    </div>
  );
}
