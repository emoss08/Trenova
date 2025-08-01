/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { LazyImage } from "@/components/ui/image";
import { queries } from "@/lib/queries";
import { ShipmentCommentSchema } from "@/lib/schemas/shipment-comment-schema";
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { useCommentEditStore } from "@/stores/comment-edit-store";
import { useUser } from "@/stores/user-store";
import { faEllipsis } from "@fortawesome/pro-solid-svg-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { formatDistanceToNow } from "date-fns";
import { toast } from "sonner";
import { CommentJsonRenderer } from "./comment-json-renderer";

export function CommentContent({
  shipmentComment,
  isLast,
}: {
  shipmentComment: ShipmentCommentSchema;
  isLast: boolean;
}) {
  const user = useUser();
  const queryClient = useQueryClient();
  const { setEditingComment } = useCommentEditStore();
  const timeAgo = shipmentComment.createdAt
    ? formatDistanceToNow(new Date(shipmentComment.createdAt * 1000), {
        addSuffix: true,
      })
    : "Unknown time";

  const isOwner = user?.id === shipmentComment.userId;

  const { mutateAsync: deleteComment, isPending: isDeleting } = useMutation({
    mutationFn: async () => {
      await api.shipments.deleteComment(
        shipmentComment.shipmentId,
        shipmentComment.id,
      );

      queryClient.invalidateQueries({
        queryKey: queries.shipment.listComments(shipmentComment.shipmentId)
          .queryKey,
      });
    },
    onSuccess: () => {
      toast.success("Comment deleted successfully");
    },
    onError: (error) => {
      toast.error("Failed to delete comment");
      console.error(error);
    },
  });

  const handleDelete = () => {
    if (!isOwner) {
      toast.error("You do not have permission to delete this comment");
      return;
    }

    deleteComment();
  };

  const handleEdit = () => {
    if (!isOwner) {
      toast.error("You do not have permission to edit this comment");
      return;
    }

    setEditingComment(shipmentComment);
  };

  return (
    <div
      className={cn(
        "group relative flex gap-3 py-4 shrink-0",
        !isLast && "border-b border-border/50",
      )}
    >
      <div className="flex-shrink-0">
        <LazyImage
          src={`https://avatar.vercel.sh/${shipmentComment.user?.username || "anonymous"}.svg`}
          alt={shipmentComment.user?.name || "User"}
          className="size-6 rounded-full"
        />
      </div>
      <div className="flex-1 space-y-1 flex flex-col">
        <div className="flex items-center gap-2 justify-between">
          <div className="flex flex-row items-center gap-2">
            <span className="text-sm font-medium">
              {shipmentComment.user?.name || "Unknown User"}
            </span>
            <span className="bg-muted-foreground/60 rounded-full size-1 text-xs" />
            <span className="text-xs text-muted-foreground">{timeAgo}</span>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <div className="flex items-center gap-2 invisible group-hover:visible mr-2">
                <Button variant="ghost" size="xs">
                  <Icon icon={faEllipsis} />
                </Button>
              </div>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem
                title="Edit"
                disabled={!isOwner || isDeleting}
                onClick={handleEdit}
              />
              <DropdownMenuItem
                color="danger"
                title="Delete"
                disabled={!isOwner || isDeleting}
                onClick={handleDelete}
              />
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
        <div className="text-sm text-foreground">
          {shipmentComment.metadata?.editorContent && (
            <CommentJsonRenderer
              content={shipmentComment.metadata.editorContent}
            />
          )}
        </div>
      </div>
    </div>
  );
}
