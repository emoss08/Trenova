import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { nanoid } from "nanoid";
import { useCallback, useRef } from "react";
import { toast } from "sonner";
import {
  adjustCommentCount,
  applyCommentPatch,
  applyCommentRemoval,
  applyCommentUpsert,
  applyReplyCountDelta,
  commentClientRef,
  commentKeys,
  findCommentInCaches,
  snapshotCommentCaches,
  syncPinnedPage,
  type CommentPage,
  type LocalShipmentComment,
} from "@/lib/shipment-comment-cache";
import { applyCommentEntitySync, markCommentSeen } from "@/lib/shipment-comment-realtime";
import { apiService } from "@/services/api";
import type {
  ShipmentComment,
  ShipmentCommentCreateInput,
  ShipmentCommentUpdateInput,
} from "@/types/shipment-comment";

type CreateCommentVariables = ShipmentCommentCreateInput & { clientRef: string };

type CommentUserSnapshot = NonNullable<LocalShipmentComment["user"]>;

function nowUnix(): number {
  return Math.floor(Date.now() / 1000);
}

function toUserSnapshot(
  user: {
    id?: string;
    name: string;
    emailAddress: string;
    profilePicUrl?: string | null;
    thumbnailUrl?: string | null;
  } | null,
): CommentUserSnapshot | null {
  if (!user?.id) return null;
  return {
    id: user.id,
    name: user.name,
    emailAddress: user.emailAddress,
    profilePicUrl: user.profilePicUrl ?? null,
    thumbnailUrl: user.thumbnailUrl ?? null,
  };
}

export function useCommentMutations(shipmentId: string) {
  const queryClient = useQueryClient();
  const currentUser = useAuthStore((s) => s.user);
  const pendingInputsRef = useRef<Map<string, ShipmentCommentCreateInput>>(new Map());

  const buildOptimisticComment = useCallback(
    (variables: CreateCommentVariables): LocalShipmentComment => {
      const now = nowUnix();
      return {
        id: `optimistic-${variables.clientRef}`,
        shipmentId,
        userId: currentUser?.id ?? null,
        parentCommentId: variables.parentCommentId ?? null,
        replyCount: 0,
        comment: variables.comment,
        body: variables.body ?? null,
        type: variables.type ?? "Internal",
        visibility: variables.visibility ?? "Internal",
        priority: variables.priority ?? "Normal",
        source: "User",
        metadata: { clientRef: variables.clientRef },
        editedAt: null,
        pinnedAt: null,
        pinnedById: null,
        resolvedAt: null,
        resolvedById: null,
        requiresAcknowledgment: variables.requiresAcknowledgment ?? false,
        deletedAt: null,
        version: 0,
        createdAt: now,
        updatedAt: now,
        mentionedUserIds: variables.mentionedUserIds ?? [],
        user: toUserSnapshot(currentUser),
        pinnedBy: null,
        resolvedBy: null,
        mentionedUsers: null,
        acknowledgments: null,
        attachments: null,
        pending: true,
      };
    },
    [shipmentId, currentUser],
  );

  const createMutation = useMutation({
    mutationFn: (variables: CreateCommentVariables) =>
      apiService.shipmentCommentService.create(shipmentId, variables),
    onMutate: async (variables) => {
      await queryClient.cancelQueries({ queryKey: commentKeys.root(shipmentId) });
      const optimistic = buildOptimisticComment(variables);
      applyCommentUpsert(queryClient, shipmentId, optimistic);
      adjustCommentCount(queryClient, shipmentId, 1);
      return { optimisticId: optimistic.id };
    },
    onSuccess: (created, variables) => {
      pendingInputsRef.current.delete(variables.clientRef);
      markCommentSeen(shipmentId, created.id);
      applyCommentUpsert(queryClient, shipmentId, created as LocalShipmentComment);
    },
    onError: (_error, _variables, context) => {
      if (context?.optimisticId) {
        applyCommentPatch(queryClient, shipmentId, context.optimisticId, {
          pending: false,
          failed: true,
        });
      }
      toast.error("Failed to send comment", {
        description: "Your draft is preserved — retry or discard it from the comment.",
      });
    },
  });

  // useMutation returns a new object every render; only its members are
  // referentially stable. Depending on the object rebuilds these callbacks on
  // every render, which defeats the memoization and re-renders every consumer.
  const { mutate: mutateCreate } = createMutation;

  const createComment = useCallback(
    (input: ShipmentCommentCreateInput) => {
      const clientRef = nanoid();
      pendingInputsRef.current.set(clientRef, input);
      mutateCreate({ ...input, clientRef });
    },
    [mutateCreate],
  );

  const retryComment = useCallback(
    (comment: LocalShipmentComment) => {
      const clientRef = commentClientRef(comment);
      if (!clientRef) return;
      const input = pendingInputsRef.current.get(clientRef);
      if (!input) return;
      applyCommentPatch(queryClient, shipmentId, comment.id, { pending: true, failed: false });
      mutateCreate({ ...input, clientRef });
    },
    [mutateCreate, queryClient, shipmentId],
  );

  const discardComment = useCallback(
    (comment: LocalShipmentComment) => {
      const clientRef = commentClientRef(comment);
      if (clientRef) pendingInputsRef.current.delete(clientRef);
      applyCommentRemoval(queryClient, shipmentId, comment.id);
      adjustCommentCount(queryClient, shipmentId, -1);
      if (comment.parentCommentId) {
        applyReplyCountDelta(queryClient, shipmentId, comment.parentCommentId, -1);
      }
    },
    [queryClient, shipmentId],
  );

  const updateMutation = useMutation({
    mutationFn: (variables: ShipmentCommentUpdateInput) =>
      apiService.shipmentCommentService.update(shipmentId, variables.id, variables),
    onMutate: async (variables) => {
      await queryClient.cancelQueries({ queryKey: commentKeys.root(shipmentId) });
      const restore = snapshotCommentCaches(queryClient, shipmentId);
      applyCommentPatch(queryClient, shipmentId, variables.id, {
        comment: variables.comment,
        body: variables.body ?? null,
        type: variables.type,
        visibility: variables.visibility,
        priority: variables.priority,
        requiresAcknowledgment: variables.requiresAcknowledgment ?? false,
        editedAt: nowUnix(),
      });
      return { restore };
    },
    onSuccess: (updated) => {
      applyCommentEntitySync(queryClient, shipmentId, updated as LocalShipmentComment);
    },
    onError: (error, _variables, context) => {
      context?.restore();
      void queryClient.invalidateQueries({
        queryKey: commentKeys.root(shipmentId),
        refetchType: "all",
      });
      const conflict = error instanceof Error && /not found|version|conflict/i.test(error.message);
      toast.error(
        conflict
          ? "Comment changed elsewhere — refreshed with the latest version"
          : "Failed to update comment",
      );
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (comment: ShipmentComment) =>
      apiService.shipmentCommentService.delete(shipmentId, comment.id),
    onMutate: async (comment) => {
      await queryClient.cancelQueries({ queryKey: commentKeys.root(shipmentId) });
      const restore = snapshotCommentCaches(queryClient, shipmentId);

      if ((comment.replyCount ?? 0) > 0) {
        applyCommentPatch(queryClient, shipmentId, comment.id, {
          deletedAt: nowUnix(),
          comment: "",
          body: null,
          pinnedAt: null,
          pinnedById: null,
          requiresAcknowledgment: false,
          attachments: null,
          mentionedUsers: null,
          mentionedUserIds: [],
        });
      } else {
        applyCommentRemoval(queryClient, shipmentId, comment.id);
        if (comment.parentCommentId) {
          applyReplyCountDelta(queryClient, shipmentId, comment.parentCommentId, -1);
        }
      }
      adjustCommentCount(queryClient, shipmentId, -1);
      return { restore };
    },
    onError: (_error, _comment, context) => {
      context?.restore();
      toast.error("Failed to delete comment");
    },
  });

  const applyOptimisticFlag = useCallback(
    (commentId: string, patch: Partial<LocalShipmentComment>) => {
      const current = findCommentInCaches(queryClient, shipmentId, commentId);
      applyCommentPatch(queryClient, shipmentId, commentId, patch);
      if (current) {
        const next = { ...current, ...patch };
        queryClient.setQueryData(commentKeys.pinned(shipmentId), (page: CommentPage | undefined) =>
          page ? syncPinnedPage(page, next) : page,
        );
      }
    },
    [queryClient, shipmentId],
  );

  const pinMutation = useMutation({
    mutationFn: ({ comment, pinned }: { comment: ShipmentComment; pinned: boolean }) =>
      apiService.shipmentCommentService.setPinned(shipmentId, comment.id, pinned),
    onMutate: async ({ comment, pinned }) => {
      await queryClient.cancelQueries({ queryKey: commentKeys.root(shipmentId) });
      const restore = snapshotCommentCaches(queryClient, shipmentId);
      applyOptimisticFlag(comment.id, {
        pinnedAt: pinned ? nowUnix() : null,
        pinnedById: pinned ? (currentUser?.id ?? null) : null,
      });
      return { restore };
    },
    onSuccess: (updated) => {
      applyCommentEntitySync(queryClient, shipmentId, updated as LocalShipmentComment);
    },
    onError: (error, { pinned }, context) => {
      context?.restore();
      const message = error instanceof Error && error.message ? error.message : null;
      toast.error(message ?? (pinned ? "Failed to pin comment" : "Failed to unpin comment"));
    },
  });

  const resolveMutation = useMutation({
    mutationFn: ({ comment, resolved }: { comment: ShipmentComment; resolved: boolean }) =>
      apiService.shipmentCommentService.setResolved(shipmentId, comment.id, resolved),
    onMutate: async ({ comment, resolved }) => {
      await queryClient.cancelQueries({ queryKey: commentKeys.root(shipmentId) });
      const restore = snapshotCommentCaches(queryClient, shipmentId);
      applyOptimisticFlag(comment.id, {
        resolvedAt: resolved ? nowUnix() : null,
        resolvedById: resolved ? (currentUser?.id ?? null) : null,
        resolvedBy: resolved ? toUserSnapshot(currentUser) : null,
      });
      return { restore };
    },
    onSuccess: (updated) => {
      applyCommentEntitySync(queryClient, shipmentId, updated as LocalShipmentComment);
    },
    onError: (_error, { resolved }, context) => {
      context?.restore();
      toast.error(resolved ? "Failed to resolve comment" : "Failed to reopen comment");
    },
  });

  const acknowledgeMutation = useMutation({
    mutationFn: (comment: ShipmentComment) =>
      apiService.shipmentCommentService.acknowledge(shipmentId, comment.id),
    onMutate: async (comment) => {
      await queryClient.cancelQueries({ queryKey: commentKeys.root(shipmentId) });
      const restore = snapshotCommentCaches(queryClient, shipmentId);
      const me = toUserSnapshot(currentUser);
      if (me) {
        const existing = comment.acknowledgments ?? [];
        if (!existing.some((ack) => ack.userId === me.id)) {
          applyCommentPatch(queryClient, shipmentId, comment.id, {
            acknowledgments: [
              ...existing,
              {
                id: `optimistic-ack-${comment.id}`,
                commentId: comment.id,
                userId: me.id,
                acknowledgedAt: nowUnix(),
                user: me,
              },
            ],
          });
        }
      }
      return { restore };
    },
    onSuccess: (updated) => {
      applyCommentEntitySync(queryClient, shipmentId, updated as LocalShipmentComment);
    },
    onError: (_error, _comment, context) => {
      context?.restore();
      toast.error("Failed to acknowledge comment");
    },
  });

  return {
    createComment,
    retryComment,
    discardComment,
    isCreating: createMutation.isPending,
    updateComment: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
    deleteComment: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
    setPinned: pinMutation.mutate,
    isPinning: pinMutation.isPending,
    setResolved: resolveMutation.mutate,
    isResolving: resolveMutation.isPending,
    acknowledgeComment: acknowledgeMutation.mutate,
    isAcknowledging: acknowledgeMutation.isPending,
  };
}
