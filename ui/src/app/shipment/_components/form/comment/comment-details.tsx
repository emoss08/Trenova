import { LazyLoader } from "@/components/error-boundary";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useQuery } from "@tanstack/react-query";
import { AlertCircleIcon } from "lucide-react";
import React, { useEffect, useRef } from "react";
import { CommentContent } from "./comment-content";
import { CommentForm } from "./comment-form";

export function ShipmentCommentDetails({
  shipmentId,
}: {
  shipmentId: ShipmentSchema["id"];
}) {
  const lastCommentRef = useRef<HTMLDivElement>(null);

  const {
    data: commentsData,
    isLoading,
    isError,
    error,
  } = useQuery({
    ...queries.shipment.listComments(shipmentId),
    enabled: !!shipmentId,
  });

  const comments = commentsData?.results || [];

  useEffect(() => {
    if (comments.length > 0 && lastCommentRef.current) {
      lastCommentRef.current.scrollIntoView({
        behavior: "smooth",
        block: "end",
      });
    }
  }, [comments.length]);

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

  return (
    <CommentDetailsOuter>
      {comments.length === 0 ? (
        <div className="text-center py-8 text-sm text-muted-foreground">
          No comments yet. Be the first to add one!
        </div>
      ) : (
        <CommentDetailsInner>
          {comments.map((comment, index) => (
            <div
              key={comment.id || index}
              ref={index === comments.length - 1 ? lastCommentRef : undefined}
            >
              <CommentContent
                shipmentComment={comment}
                isLast={index === comments.length - 1}
              />
            </div>
          ))}
        </CommentDetailsInner>
      )}

      <LazyLoader fallback={<CommentFormSkeleton />}>
        <CommentForm shipmentId={shipmentId} />
      </LazyLoader>
    </CommentDetailsOuter>
  );
}

function CommentDetailsOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-4 pb-10">{children}</div>;
}

function CommentDetailsInner({ children }: { children: React.ReactNode }) {
  return (
    <ScrollArea className="flex flex-col px-4 max-h-[calc(100vh-24rem)]">
      {children}
    </ScrollArea>
  );
}

function CommentFormSkeleton() {
  return (
    <div className="flex flex-col px-2">
      <Skeleton className="h-26 w-full rounded-md" />
      <div className="flex justify-end pt-2 gap-2">
        <Skeleton className="h-6 w-14 rounded-md" />
        <Skeleton className="h-6 w-24 rounded-md" />
      </div>
    </div>
  );
}
