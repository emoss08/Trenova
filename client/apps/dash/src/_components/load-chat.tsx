import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Message, MessageContent, MessageFooter, MessageHeader } from "@trenova/shared/components/ui/message";
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@trenova/shared/components/ui/message-scroller";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  createMyLoadComment,
  fetchMyLoadComments,
  type PortalLoadComment,
} from "@trenova/shared/lib/graphql/driver-portal";
import { cn } from "@trenova/shared/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { MessageSquareTextIcon, SendIcon } from "lucide-react";
import { useDashFeatures } from "./use-dash-features";
import { useMemo, useState } from "react";
import { toast } from "sonner";

const commentTypeLabels: Record<string, string> = {
  Internal: "Note",
  Dispatch: "Dispatch",
  DriverUpdate: "You",
  PickupInstruction: "Pickup instructions",
  DeliveryInstruction: "Delivery instructions",
  StatusUpdate: "Status update",
  Exception: "Exception",
  CustomerUpdate: "Customer update",
  Appointment: "Appointment",
  Document: "Documents",
  Billing: "Billing",
  Compliance: "Compliance",
};

function messageTime(unix: number): string {
  const date = new Date(unix * 1000);
  const sameDay = new Date().toDateString() === date.toDateString();
  const time = new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
  if (sameDay) {
    return time;
  }
  const day = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" }).format(date);
  return `${day}, ${time}`;
}

function ChatMessage({ comment }: { comment: PortalLoadComment }) {
  const isMine = comment.type === "DriverUpdate";
  const urgent = comment.priority === "High" || comment.priority === "Urgent";

  return (
    <Message align={isMine ? "end" : "start"}>
      <MessageContent className="max-w-[85%]">
        <MessageHeader className="gap-1.5">
          {isMine ? "You" : comment.authorName}
          {!isMine && commentTypeLabels[comment.type] && comment.type !== "Dispatch" ? (
            <span className="text-2xs text-muted-foreground/70">
              · {commentTypeLabels[comment.type]}
            </span>
          ) : null}
          {urgent ? (
            <Badge variant={comment.priority === "Urgent" ? "inactive" : "orange"}>
              {comment.priority}
            </Badge>
          ) : null}
        </MessageHeader>
        <div
          className={cn(
            "w-fit rounded-2xl px-3 py-2 text-sm whitespace-pre-wrap",
            isMine
              ? "self-end rounded-br-md bg-primary text-primary-foreground"
              : "rounded-bl-md bg-muted",
            urgent && !isMine && "border border-orange-500/40 bg-orange-500/10",
          )}
        >
          {comment.comment}
        </div>
        <MessageFooter>{messageTime(comment.createdAt)}</MessageFooter>
      </MessageContent>
    </Message>
  );
}

export function LoadChat({ shipmentId }: { shipmentId: string }) {
  const features = useDashFeatures();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState("");

  const comments = useQuery({
    queryKey: ["dash-load-comments", shipmentId],
    queryFn: () => fetchMyLoadComments(shipmentId),
    enabled: shipmentId.length > 0,
  });

  const thread = useMemo(() => [...(comments.data ?? [])].reverse(), [comments.data]);

  const send = useMutation({
    mutationFn: () => createMyLoadComment({ shipmentId, comment: draft.trim() }),
    onSuccess: async () => {
      setDraft("");
      await queryClient.invalidateQueries({ queryKey: ["dash-load-comments", shipmentId] });
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't send your message."),
  });

  const handleSend = () => {
    if (draft.trim().length === 0 || send.isPending) {
      return;
    }
    send.mutate();
  };

  if (comments.isPending) {
    return <Skeleton className="h-64 w-full rounded-2xl" />;
  }

  return (
    <div className="flex flex-col overflow-hidden rounded-2xl border border-border bg-card">
      <div className="flex items-center gap-2 border-b border-border px-4 py-3">
        <MessageSquareTextIcon className="size-4 text-muted-foreground" />
        <h2 className="text-sm font-semibold">Dispatch chat</h2>
        <span className="ml-auto flex items-center gap-1.5 text-xs text-muted-foreground">
          <span className="size-1.5 rounded-full bg-green-500" />
          Live
        </span>
      </div>

      <MessageScrollerProvider autoScroll defaultScrollPosition="end">
        <MessageScroller className="h-80 border-0">
          <MessageScrollerViewport className="px-4 py-3">
            <MessageScrollerContent className="gap-4">
              {thread.length === 0 ? (
                <p className="py-10 text-center text-xs text-muted-foreground">
                  No messages yet. Say something and dispatch sees it on the shipment right away.
                </p>
              ) : (
                thread.map((comment) => (
                  <MessageScrollerItem
                    key={comment.id}
                    messageId={comment.id}
                    scrollAnchor={comment.type === "DriverUpdate"}
                  >
                    <ChatMessage comment={comment} />
                  </MessageScrollerItem>
                ))
              )}
            </MessageScrollerContent>
          </MessageScrollerViewport>
          <MessageScrollerButton />
        </MessageScroller>
      </MessageScrollerProvider>

      {features.allowLoadComments ? (
        <div className="flex items-end gap-2 border-t border-border p-3">
          <Textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                handleSend();
              }
            }}
            placeholder="Message dispatch..."
            rows={1}
            maxLength={5000}
            className="max-h-24 min-h-9 flex-1 resize-none"
          />
          <Button
            size="icon"
            aria-label="Send message"
            disabled={draft.trim().length === 0 || send.isPending}
            onClick={handleSend}
          >
            <SendIcon className="size-4" />
          </Button>
        </div>
      ) : null}
    </div>
  );
}
