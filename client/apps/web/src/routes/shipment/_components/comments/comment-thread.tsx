import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { ChevronDownIcon, MessageSquareReplyIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useShipmentCommentReplies } from "@/hooks/shipment-comments/use-shipment-comments";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";
import type { ShipmentCommentCreateInput } from "@/types/shipment-comment";
import { CommentItem, type CommentItemActions } from "./comment-item";
import { CommentComposer } from "./composer";

export function CommentThread({
  shipmentId,
  parentComment,
  isExpanded,
  isReplyOpen,
  onToggleExpand,
  onCloseReply,
  onlineUserIDs,
  editingCommentId,
  onEditComment,
  onCancelEdit,
  buildReplyActions,
  createReply,
  isCreating,
  onTyping,
  onStopTyping,
}: {
  shipmentId: string;
  parentComment: LocalShipmentComment;
  isExpanded: boolean;
  isReplyOpen: boolean;
  onToggleExpand: () => void;
  onCloseReply: () => void;
  onlineUserIDs: Set<string>;
  editingCommentId: string | null;
  onEditComment: (commentId: string) => void;
  onCancelEdit: () => void;
  buildReplyActions: (reply: LocalShipmentComment) => CommentItemActions;
  createReply: (input: ShipmentCommentCreateInput) => void;
  isCreating: boolean;
  onTyping?: () => void;
  onStopTyping?: () => void;
}) {
  const replyCount = parentComment.replyCount ?? 0;
  const showRegion = isExpanded || isReplyOpen;
  const {
    replies,
    isRepliesLoading,
    isRepliesError,
    hasMoreReplies,
    isFetchingMoreReplies,
    fetchMoreReplies,
  } = useShipmentCommentReplies(shipmentId, parentComment.id, showRegion);

  if (replyCount === 0 && !isReplyOpen) return null;

  return (
    <div className="ml-12">
      {replyCount > 0 && (
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="text-2xs text-muted-foreground hover:text-foreground h-6 gap-1.5 px-1.5"
          onClick={onToggleExpand}
          aria-expanded={isExpanded}
        >
          <ChevronDownIcon
            className={`size-3 transition-transform duration-200 ${isExpanded ? "" : "-rotate-90"}`}
          />
          <MessageSquareReplyIcon className="size-3" />
          {replyCount === 1 ? "1 reply" : `${replyCount} replies`}
        </Button>
      )}
      <AnimatePresence initial={false}>
        {showRegion && (
          <m.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.18, ease: "easeOut" }}
            className="overflow-hidden"
          >
            <div className="space-y-0.5 pt-1">
              {isExpanded && isRepliesLoading && <ReplySkeleton />}
              {isExpanded && isRepliesError && (
                <p className="text-2xs px-2 py-1.5 text-red-500">Failed to load replies</p>
              )}
              {isExpanded && hasMoreReplies && !isRepliesLoading && (
                <Button
                  type="button"
                  variant="ghost"
                  size="xs"
                  className="text-2xs text-muted-foreground h-6 px-1.5"
                  disabled={isFetchingMoreReplies}
                  onClick={() => void fetchMoreReplies()}
                >
                  {isFetchingMoreReplies ? "Loading…" : "Load earlier replies"}
                </Button>
              )}
              {isExpanded &&
                replies.map((reply) => (
                  <CommentItem
                    key={reply.id}
                    comment={reply}
                    isEditing={editingCommentId === reply.id}
                    onEdit={() => onEditComment(reply.id)}
                    onCancelEdit={onCancelEdit}
                    actions={buildReplyActions(reply)}
                    isOnline={!!reply.user?.id && onlineUserIDs.has(reply.user.id)}
                  />
                ))}
              {isReplyOpen && (
                <div className="py-1.5 pr-2">
                  <CommentComposer
                    shipmentId={shipmentId}
                    parentCommentId={parentComment.id}
                    compact
                    autoFocus
                    isSubmitting={isCreating}
                    onSubmit={createReply}
                    onTyping={onTyping}
                    onStopTyping={onStopTyping}
                    onEscape={onCloseReply}
                  />
                </div>
              )}
            </div>
          </m.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function ReplySkeleton() {
  return (
    <div className="space-y-2 px-2 py-1.5">
      {Array.from({ length: 2 }).map((_, index) => (
        <div key={index} className="flex items-center gap-2">
          <Skeleton className="size-6 rounded-full" />
          <div className="flex-1 space-y-1">
            <Skeleton className="h-2.5 w-24" />
            <Skeleton className="h-2.5 w-3/4" />
          </div>
        </div>
      ))}
    </div>
  );
}
