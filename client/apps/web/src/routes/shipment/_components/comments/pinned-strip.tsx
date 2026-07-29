import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { usePermission } from "@/hooks/use-permission";
import { Operation } from "@trenova/shared/types/permission";
import { ChevronDownIcon, PinIcon, PinOffIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useState } from "react";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";

export function PinnedStrip({
  pinnedComments,
  isLoading,
  onJumpToComment,
  onUnpin,
}: {
  pinnedComments: LocalShipmentComment[];
  isLoading: boolean;
  onJumpToComment: (commentId: string) => void;
  onUnpin: (comment: LocalShipmentComment) => void;
}) {
  const [isExpanded, setIsExpanded] = useState(false);
  const { allowed: canUnpin } = usePermission("shipment_comment", Operation.Unpin);

  if (isLoading) {
    return (
      <div className="border-b border-border px-4 py-2">
        <Skeleton className="h-5 w-48" />
      </div>
    );
  }

  if (pinnedComments.length === 0) return null;

  const first = pinnedComments[0];

  return (
    <div className="border-b border-border bg-amber-500/[0.04]">
      <button
        type="button"
        className="flex w-full items-center gap-2 px-4 py-2 text-left transition-colors hover:bg-amber-500/[0.08]"
        onClick={() => setIsExpanded((expanded) => !expanded)}
        aria-expanded={isExpanded}
      >
        <PinIcon className="size-3.5 shrink-0 text-amber-500" />
        <span className="shrink-0 text-xs font-medium">
          {pinnedComments.length === 1 ? "1 pinned" : `${pinnedComments.length} pinned`}
        </span>
        {!isExpanded && (
          <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">
            {first.user?.name ? `${first.user.name}: ` : ""}
            {first.comment}
          </span>
        )}
        <ChevronDownIcon
          className={`ml-auto size-3.5 shrink-0 text-muted-foreground transition-transform duration-200 ${
            isExpanded ? "rotate-180" : ""
          }`}
        />
      </button>
      <AnimatePresence initial={false}>
        {isExpanded && (
          <m.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.18, ease: "easeOut" }}
            className="overflow-hidden"
          >
            <div className="space-y-0.5 px-2 pb-2">
              {pinnedComments.map((comment) => (
                <div
                  key={comment.id}
                  className="group/pinned flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-muted/60"
                >
                  <button
                    type="button"
                    className="min-w-0 flex-1 text-left"
                    onClick={() => onJumpToComment(comment.id)}
                  >
                    <span className="block truncate text-xs">
                      <span className="font-medium">{comment.user?.name ?? "Unknown"}</span>
                      <span className="text-muted-foreground"> — {comment.comment}</span>
                    </span>
                  </button>
                  {canUnpin && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-xs"
                      className="size-5 shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover/pinned:opacity-100"
                      aria-label="Unpin comment"
                      onClick={() => onUnpin(comment)}
                    >
                      <PinOffIcon className="size-3" />
                    </Button>
                  )}
                </div>
              ))}
            </div>
          </m.div>
        )}
      </AnimatePresence>
    </div>
  );
}
