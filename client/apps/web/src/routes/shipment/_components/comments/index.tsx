import { MessageScrollerProvider } from "@trenova/shared/components/ui/message-scroller";
import { TooltipProvider } from "@trenova/shared/components/ui/tooltip";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { WifiOffIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useCommentMutations } from "@/hooks/shipment-comments/use-comment-mutations";
import {
  usePinnedShipmentComments,
  useShipmentCommentList,
} from "@/hooks/shipment-comments/use-shipment-comments";
import { useShipmentTyping } from "@/hooks/shipment-comments/use-shipment-typing";
import { useShipmentViewers } from "@/hooks/shipment-comments/use-shipment-viewers";
import { useOnlineUsers } from "@/hooks/use-online-users";
import { useRealtimeStore } from "@/stores/realtime-store";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";
import { CommentItem, type CommentItemActions } from "./comment-item";
import { CommentStream, type CommentJumpRequest } from "./comment-stream";
import { CommentThread } from "./comment-thread";
import { CommentComposer } from "./composer";
import {
  CommentsToolbar,
  EMPTY_TOOLBAR_FILTERS,
  toolbarFiltersToQueryFilter,
  type CommentToolbarFilterState,
} from "./comments-toolbar";
import { PinnedStrip } from "./pinned-strip";
import { TypingIndicator } from "./typing-indicator";
import { ViewersStack } from "./viewers-stack";

export default function ShipmentCommentsTab({ shipmentId }: { shipmentId: string }) {
  const currentUser = useAuthStore((s) => s.user);
  const connectionState = useRealtimeStore((s) => s.connectionState);

  const [toolbarFilters, setToolbarFilters] =
    useState<CommentToolbarFilterState>(EMPTY_TOOLBAR_FILTERS);
  const [editingCommentId, setEditingCommentId] = useState<string | null>(null);
  const [replyingToId, setReplyingToId] = useState<string | null>(null);
  const [expandedThreads, setExpandedThreads] = useState<Set<string>>(new Set());
  const [jumpRequest, setJumpRequest] = useState<CommentJumpRequest | null>(null);
  const [highlightedCommentId, setHighlightedCommentId] = useState<string | null>(null);

  const queryFilter = useMemo(
    () => toolbarFiltersToQueryFilter(toolbarFilters, currentUser?.id),
    [toolbarFilters, currentUser?.id],
  );

  const list = useShipmentCommentList(shipmentId, queryFilter);
  const { pinnedComments, isPinnedLoading } = usePinnedShipmentComments(shipmentId);
  const mutations = useCommentMutations(shipmentId);
  const { typingUsers, publishTyping, publishStopTyping } = useShipmentTyping(shipmentId);
  const { viewers } = useShipmentViewers(shipmentId);
  const { onlineUserIDs } = useOnlineUsers();

  const toggleThread = useCallback((commentId: string) => {
    setExpandedThreads((previous) => {
      const next = new Set(previous);
      if (next.has(commentId)) {
        next.delete(commentId);
      } else {
        next.add(commentId);
      }
      return next;
    });
  }, []);

  const openReply = useCallback((commentId: string) => {
    setReplyingToId(commentId);
    setExpandedThreads((previous) => new Set(previous).add(commentId));
  }, []);

  const buildActions = useCallback(
    (comment: LocalShipmentComment, isReply: boolean): CommentItemActions => ({
      onReply: () => openReply(isReply ? (comment.parentCommentId ?? comment.id) : comment.id),
      onSaveEdit: (input) =>
        mutations.updateComment(input, { onSuccess: () => setEditingCommentId(null) }),
      isUpdating: mutations.isUpdating,
      onDelete: () => mutations.deleteComment(comment),
      isDeleting: mutations.isDeleting,
      onTogglePin: isReply ? undefined : (pinned) => mutations.setPinned({ comment, pinned }),
      onToggleResolve: (resolved) => mutations.setResolved({ comment, resolved }),
      onAcknowledge: () => mutations.acknowledgeComment(comment),
      isAcknowledging: mutations.isAcknowledging,
      onRetry: () => mutations.retryComment(comment),
      onDiscard: () => mutations.discardComment(comment),
    }),
    [mutations, openReply],
  );

  const jumpToComment = useCallback((commentId: string) => {
    setJumpRequest({ commentId, nonce: Date.now() });
  }, []);

  const renderComment = useCallback(
    (comment: LocalShipmentComment, highlighted: boolean) => (
      <>
        <CommentItem
          comment={comment}
          isEditing={editingCommentId === comment.id}
          onEdit={() => setEditingCommentId(comment.id)}
          onCancelEdit={() => setEditingCommentId(null)}
          actions={buildActions(comment, false)}
          isOnline={!!comment.user?.id && onlineUserIDs.has(comment.user.id)}
          highlighted={highlighted}
        />
        {comment.deletedAt == null || (comment.replyCount ?? 0) > 0 ? (
          <CommentThread
            shipmentId={shipmentId}
            parentComment={comment}
            isExpanded={expandedThreads.has(comment.id)}
            isReplyOpen={replyingToId === comment.id}
            onToggleExpand={() => toggleThread(comment.id)}
            onCloseReply={() => setReplyingToId(null)}
            onlineUserIDs={onlineUserIDs}
            editingCommentId={editingCommentId}
            onEditComment={setEditingCommentId}
            onCancelEdit={() => setEditingCommentId(null)}
            buildReplyActions={(reply) => buildActions(reply, true)}
            createReply={(input) => {
              mutations.createComment(input);
              setReplyingToId(null);
            }}
            isCreating={mutations.isCreating}
            onTyping={publishTyping}
            onStopTyping={publishStopTyping}
          />
        ) : null}
      </>
    ),
    [
      editingCommentId,
      buildActions,
      onlineUserIDs,
      shipmentId,
      expandedThreads,
      replyingToId,
      toggleThread,
      mutations,
      publishTyping,
      publishStopTyping,
    ],
  );

  return (
    <TooltipProvider>
      <MessageScrollerProvider autoScroll defaultScrollPosition="end">
        <div className="flex h-full flex-col">
          <CommentsToolbar
            filters={toolbarFilters}
            onFiltersChange={setToolbarFilters}
            isFiltering={list.isFiltering}
          />
          {!list.isFiltered && (
            <PinnedStrip
              pinnedComments={pinnedComments}
              isLoading={isPinnedLoading}
              onJumpToComment={jumpToComment}
              onUnpin={(comment) => mutations.setPinned({ comment, pinned: false })}
            />
          )}
          <div className="relative min-h-0 flex-1">
            <CommentStream
              shipmentId={shipmentId}
              comments={list.comments}
              isLoading={list.isLoading}
              isError={list.isError}
              onRetry={() => void list.refetch()}
              isFiltered={list.isFiltered}
              hasNextPage={list.hasNextPage}
              isFetchingNextPage={list.isFetchingNextPage}
              fetchNextPage={list.fetchNextPage}
              jumpRequest={jumpRequest}
              highlightedCommentId={highlightedCommentId}
              onHighlight={setHighlightedCommentId}
              onClearFilters={() => setToolbarFilters(EMPTY_TOOLBAR_FILTERS)}
              renderComment={renderComment}
            />
          </div>
          {connectionState !== "connected" && (
            <div className="flex items-center gap-1.5 border-t border-border bg-amber-500/5 px-4 py-1 text-2xs text-muted-foreground">
              <WifiOffIcon className="size-3" />
              Live updates paused — reconnecting…
            </div>
          )}
          <div className="flex items-center justify-between gap-2 pr-4">
            <TypingIndicator typingUsers={typingUsers} />
            <ViewersStack viewers={viewers} />
          </div>
          <div className="shrink-0 border-t border-border px-4 pt-3 pb-4">
            <CommentComposer
              shipmentId={shipmentId}
              isSubmitting={mutations.isCreating}
              onSubmit={mutations.createComment}
              onTyping={publishTyping}
              onStopTyping={publishStopTyping}
            />
          </div>
        </div>
      </MessageScrollerProvider>
    </TooltipProvider>
  );
}
