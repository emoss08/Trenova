import {
  openDeskConversation,
  type DeskConversation,
  type DeskConversationRow,
  type OpenedDeskConversation,
} from "@/lib/graphql/desk-transcripts";
import { downloadDeskConversation } from "@/lib/desk-transcripts";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
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
 * Another person's conversation, read-only, wide enough to read as the Desk
 * draws it: whose it is and when in the head, that opening it was recorded,
 * and what was said. Nothing here can post, rename, pin, snooze, delete or
 * continue it; Download, beside the title, is the only action, for holders of
 * Export.
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

  const conversation = opened.data?.conversation ?? row;
  const title = conversation?.title || t("Untitled conversation");

  return (
    <Sheet open={row !== null} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="right" size="3xl" className="gap-0">
        <SheetHeader className="border-border gap-1 border-b pr-12">
          <div className="flex min-w-0 items-center gap-3">
            <SheetTitle className="min-w-0 flex-1 truncate">{title}</SheetTitle>
            {row && canExport && opened.status === "success" ? (
              <DownloadButton conversationId={row.id} />
            ) : null}
          </div>
          {conversation ? <ConversationFacts conversation={conversation} /> : null}
        </SheetHeader>
        {row ? (
          <SheetBody
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

/** Whose conversation it is, where it began, how long it is and when it was held, as one line. */
function ConversationFacts({
  conversation,
}: {
  conversation: Pick<
    DeskConversation,
    "ownerName" | "agentName" | "origin" | "messageCount" | "startedAt" | "lastMessageAt"
  >;
}) {
  const t = useT();
  const facts = [
    t("{0} with {1}", conversation.ownerName, conversation.agentName),
    originLabel(t, conversation.origin),
    t("{0, plural, one {# message} other {# messages}}", conversation.messageCount),
    t("Started {0}", formatUnixDateTimeMedium(conversation.startedAt)),
    ...(conversation.lastMessageAt
      ? [t("Last message {0}", formatUnixDateTimeMedium(conversation.lastMessageAt))]
      : []),
  ];

  return (
    <SheetDescription className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5">
      {facts.map((fact, index) => (
        <span key={fact} className="flex min-w-0 items-center gap-x-1.5">
          {index > 0 ? (
            <span aria-hidden className="text-foreground-subtle">
              ·
            </span>
          ) : null}
          <span className="min-w-0 break-words">{fact}</span>
        </span>
      ))}
    </SheetDescription>
  );
}

function DownloadButton({ conversationId }: { conversationId: string }) {
  const t = useT();
  const [downloading, setDownloading] = useState(false);

  const download = async () => {
    if (downloading) {
      return;
    }
    setDownloading(true);
    try {
      await downloadDeskConversation(conversationId);
    } catch (downloadError) {
      toast.error(t("The transcript could not be downloaded"), {
        description: graphQLErrorMessage(downloadError, t("Try again shortly.")),
      });
    } finally {
      setDownloading(false);
    }
  };

  return (
    <Button
      type="button"
      size="sm"
      className="shrink-0"
      isLoading={downloading}
      onClick={() => void download()}
    >
      <Download01Icon className="size-3.5" />
      {t("Download")}
    </Button>
  );
}

function SheetBody({
  state,
  data,
  error,
  onRetry,
}: {
  state: "idle" | "pending" | "success" | "error";
  data: OpenedDeskConversation | null;
  error: Error | null;
  onRetry: () => void;
}) {
  const t = useT();

  if (state === "error") {
    const kind = describeError(error).kind;
    return (
      <div className="flex flex-col gap-3 p-4 sm:px-5">
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
    <ScrollArea maskVariant="card" className="min-h-0 flex-1">
      <div className="flex min-w-0 flex-col gap-6 overflow-x-clip pt-4 pb-8">
        <div className="px-4 sm:px-5">
          <Alert variant="info" size="sm">
            <EyeIcon />
            <AlertDescription>
              {t(
                "Opening {0}'s conversation was recorded in the audit log. Nothing here changes it for them.",
                conversation.ownerName,
              )}
            </AlertDescription>
          </Alert>
        </div>
        <ConversationTranscript conversationId={conversation.id} transcript={data.transcript} />
      </div>
    </ScrollArea>
  );
}
