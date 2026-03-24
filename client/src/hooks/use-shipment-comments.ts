import { apiService } from "@/services/api";
import type {
  ShipmentCommentCreateInput,
  ShipmentCommentUpdateInput,
} from "@/types/shipment-comment";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { toast } from "sonner";

const PAGE_SIZE = 20;

export function useShipmentComments(shipmentId: string) {
  const queryClient = useQueryClient();

  const listQuery = useInfiniteQuery({
    queryKey: ["shipment-comments", shipmentId],
    queryFn: ({ pageParam }) =>
      apiService.shipmentCommentService.list(shipmentId, {
        limit: PAGE_SIZE,
        offset: pageParam,
      }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.next || lastPage.results.length === PAGE_SIZE) {
        return lastPageParam + PAGE_SIZE;
      }
      return undefined;
    },
    enabled: !!shipmentId,
  });

  const comments = useMemo(
    () => {
      const pages = listQuery.data?.pages;
      if (!pages) return [];
      const allPages = [...pages].reverse();
      return allPages.flatMap((page) => [...page.results].reverse());
    },
    [listQuery.data?.pages],
  );

  const countQuery = useQuery({
    queryKey: ["shipment-comment-count", shipmentId],
    queryFn: () => apiService.shipmentCommentService.getCount(shipmentId),
    enabled: !!shipmentId,
    staleTime: 30_000,
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["shipment-comments", shipmentId] });
    void queryClient.invalidateQueries({ queryKey: ["shipment-comment-count", shipmentId] });
  };

  const createMutation = useMutation({
    mutationFn: (data: ShipmentCommentCreateInput) =>
      apiService.shipmentCommentService.create(shipmentId, data),
    onSuccess: invalidate,
    onError: () => toast.error("Failed to create comment"),
  });

  const updateMutation = useMutation({
    mutationFn: (data: ShipmentCommentUpdateInput & { commentId: string }) =>
      apiService.shipmentCommentService.update(shipmentId, data.commentId, data),
    onSuccess: invalidate,
    onError: () => toast.error("Failed to update comment"),
  });

  const deleteMutation = useMutation({
    mutationFn: (commentId: string) =>
      apiService.shipmentCommentService.delete(shipmentId, commentId),
    onSuccess: invalidate,
    onError: () => toast.error("Failed to delete comment"),
  });

  return {
    comments,
    total: listQuery.data?.pages[0]?.count ?? 0,
    isLoading: listQuery.isLoading,
    hasNextPage: listQuery.hasNextPage,
    isFetchingNextPage: listQuery.isFetchingNextPage,
    fetchNextPage: listQuery.fetchNextPage,
    commentCount: countQuery.data?.count ?? 0,
    isCountLoading: countQuery.isLoading,
    createComment: createMutation.mutate,
    isCreating: createMutation.isPending,
    updateComment: updateMutation.mutate,
    isUpdating: updateMutation.isPending,
    deleteComment: deleteMutation.mutate,
    isDeleting: deleteMutation.isPending,
  };
}
