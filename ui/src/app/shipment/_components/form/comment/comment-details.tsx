/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Alert, AlertDescription } from "@/components/ui/alert";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { UserSchema } from "@/lib/schemas/user-schema";
import { api } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { AlertCircleIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";
import { CommentContent } from "./_components/comment-content";
import { CommentForm } from "./comment-form";

export default function ShipmentCommentDetails() {
  const { getValues } = useFormContext<ShipmentSchema>();
  const shipmentId = getValues("id");

  const {
    data: commentsData,
    isLoading,
    isError,
    error,
  } = useQuery({
    ...queries.shipment.listComments(shipmentId || "", !!shipmentId),
  });

  const searchUsers = async (query: string): Promise<UserSchema[]> => {
    try {
      if (!query || query.length < 2) {
        return [];
      }

      const result = await api.user.searchUsers(query);
      return result.results || [];
    } catch (error) {
      console.error("Failed to search users:", error);
      return [];
    }
  };

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <h3 className="text-sm font-medium">Comments</h3>
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="flex gap-3">
              <Skeleton className="size-8 rounded-full" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-[200px]" />
                <Skeleton className="h-4 w-full" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex flex-col gap-4">
        <h3 className="text-sm font-medium">Comments</h3>
        <Alert variant="destructive">
          <AlertCircleIcon className="h-4 w-4" />
          <AlertDescription>
            Failed to load comments.{" "}
            {error?.message || "Please try again later."}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  const comments = commentsData?.results || [];

  return (
    <div className="flex flex-col gap-4 pb-16">
      {comments.length === 0 ? (
        <div className="text-center py-8 text-sm text-muted-foreground">
          No comments yet. Be the first to add one!
        </div>
      ) : (
        <ScrollArea className="flex flex-col overflow-y-auto max-h-[calc(100vh-14rem)]">
          {comments.map((comment, index) => (
            <CommentContent
              key={comment.id || index}
              shipmentComment={comment}
              isLast={index === comments.length - 1}
            />
          ))}
        </ScrollArea>
      )}

      <CommentForm searchUsers={searchUsers} shipmentId={shipmentId} />
    </div>
  );
}
