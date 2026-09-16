import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantMessage, AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { InfoIcon, SendIcon, ShieldAlertIcon, WrenchIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { classifyMessage, type MessagePresentation } from "./classify-message";

export function MessageThread({ thread }: { thread: AssistantThread }) {
  const t = useT();
  const queryClient = useQueryClient();

  const [draft, setDraft] = useState("");
  const bottomRef = useRef<HTMLDivElement>(null);

  const messagesQuery = useQuery(queries.assistant.messages(thread.id));
  const messages = messagesQuery.data?.results ?? [];

  const sendMutation = useApiMutation({
    mutationFn: (content: string) => apiService.assistantService.sendMessage(thread.id, content),
    onSuccess: async () => {
      setDraft("");
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.assistant.messages(thread.id).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
      ]);
    },
    resourceName: "Message",
  });

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length, sendMutation.isPending]);

  const submit = useCallback(() => {
    const content = draft.trim();
    if (content === "" || sendMutation.isPending) {
      return;
    }
    sendMutation.mutate(content);
  }, [draft, sendMutation]);

  return (
    <>
      <ScrollArea className="min-h-0 flex-1">
        <div className="mx-auto flex max-w-3xl flex-col gap-4 p-4">
          {messagesQuery.isLoading ? (
            <>
              <Skeleton className="h-16" />
              <Skeleton className="h-16" />
            </>
          ) : (
            messages.map((message) => <MessageBubble key={message.id} message={message} />)
          )}
          {sendMutation.isPending && <ThinkingIndicator />}
          <div ref={bottomRef} />
        </div>
      </ScrollArea>

      <div className="border-border border-t p-3">
        <div className="mx-auto flex max-w-3xl items-end gap-2">
          <Textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              // Enter sends; Shift+Enter is a newline, which is what people expect
              // from a chat box rather than a form field.
              if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                submit();
              }
            }}
            placeholder={t("Ask about a shipment, a driver, or how to do something…")}
            rows={2}
            className="resize-none"
          />
          <Button
            size="sm"
            onClick={submit}
            disabled={draft.trim() === "" || sendMutation.isPending}
            aria-label={t("Send")}
          >
            <SendIcon className="size-4" />
          </Button>
        </div>
      </div>
    </>
  );
}

function MessageBubble({ message }: { message: AssistantMessage }) {
  const presentation = classifyMessage(message);

  if (presentation === "tool") {
    return <ToolResultRow message={message} />;
  }

  if (presentation === "refusal" || presentation === "declined-prompt") {
    return <RefusalBubble message={message} presentation={presentation} />;
  }

  if (presentation === "user") {
    return (
      <div className="flex justify-end">
        <div className="bg-primary text-primary-foreground max-w-[80%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap">
          {message.content}
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1">
      {message.content && (
        <div className="bg-muted max-w-[85%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap">
          {message.content}
        </div>
      )}
      {message.toolCalls && message.toolCalls.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {message.toolCalls.map((call) => (
            <Badge key={call.id} variant="secondary" className="gap-1">
              <WrenchIcon className="size-3" />
              {call.name}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * A refusal is rendered as a boundary rather than as an error. The assistant did
 * not fail; it declined, and the message explains what it does cover.
 */
function RefusalBubble({
  message,
  presentation,
}: {
  message: AssistantMessage;
  presentation: MessagePresentation;
}) {
  const t = useT();

  if (presentation === "declined-prompt") {
    return (
      <div className="flex justify-end">
        <div className="border-border text-muted-foreground max-w-[80%] rounded-lg border border-dashed px-3 py-2 text-sm whitespace-pre-wrap">
          {message.content}
        </div>
      </div>
    );
  }

  return (
    <Alert variant="warning" className="max-w-[85%]">
      <ShieldAlertIcon className="size-4" />
      <AlertTitle>{t("Outside what this assistant covers")}</AlertTitle>
      <AlertDescription>{message.content}</AlertDescription>
    </Alert>
  );
}

/**
 * Tool traffic is shown rather than hidden. "Which records did it read before
 * saying that" is the first question anyone asks of an answer, and a collapsed
 * row keeps it available without crowding the conversation.
 */
function ToolResultRow({ message }: { message: AssistantMessage }) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="text-muted-foreground text-xs">
      <button
        type="button"
        className="hover:text-foreground flex items-center gap-1.5"
        onClick={() => setExpanded((value) => !value)}
      >
        {message.toolFailed ? (
          <ShieldAlertIcon className="size-3" />
        ) : (
          <InfoIcon className="size-3" />
        )}
        <span>
          {message.toolFailed
            ? t("{0} failed", message.toolName)
            : t("Looked up {0}", message.toolName)}
        </span>
      </button>
      {expanded && (
        <pre className="bg-muted mt-1 max-h-64 overflow-auto rounded-md p-2 text-[11px] whitespace-pre-wrap">
          {message.content}
        </pre>
      )}
    </div>
  );
}

function ThinkingIndicator() {
  const t = useT();

  return (
    <div className="text-muted-foreground flex items-center gap-2 text-sm">
      <span className="bg-muted-foreground size-1.5 animate-pulse rounded-full" />
      <span>{t("Working…")}</span>
    </div>
  );
}
