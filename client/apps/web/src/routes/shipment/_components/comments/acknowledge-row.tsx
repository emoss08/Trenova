import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { CheckCheckIcon, LoaderIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";

const MAX_ACK_AVATARS = 5;

export function AcknowledgeRow({
  comment,
  onAcknowledge,
  isAcknowledging,
}: {
  comment: LocalShipmentComment;
  onAcknowledge: () => void;
  isAcknowledging: boolean;
}) {
  const currentUser = useAuthStore((s) => s.user);
  const acknowledgments = comment.acknowledgments ?? [];
  const hasAcknowledged = acknowledgments.some((ack) => ack.userId === currentUser?.id);
  const visible = acknowledgments.slice(0, MAX_ACK_AVATARS);
  const overflow = acknowledgments.length - visible.length;

  return (
    <div className="mt-1.5 flex items-center gap-2">
      {!hasAcknowledged && !comment.pending && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="h-6 gap-1.5 px-2 text-xs"
          disabled={isAcknowledging}
          onClick={onAcknowledge}
        >
          {isAcknowledging ? (
            <LoaderIcon className="size-3 animate-spin" />
          ) : (
            <CheckCheckIcon className="size-3" />
          )}
          Acknowledge
        </Button>
      )}
      {acknowledgments.length > 0 && (
        <div className="flex items-center gap-1.5">
          <div className="flex -space-x-1.5">
            <AnimatePresence initial={false}>
              {visible.map((ack) => (
                <m.div
                  key={ack.id}
                  initial={{ scale: 0.5, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={{ type: "spring", stiffness: 500, damping: 30 }}
                >
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span className="ring-background block rounded-full ring-2">
                          <ResolvedUserAvatar
                            userId={ack.user?.id}
                            name={ack.user?.name}
                            profilePicUrl={ack.user?.profilePicUrl}
                            thumbnailUrl={ack.user?.thumbnailUrl}
                            className="size-5"
                            fallbackClassName="text-[9px]"
                          />
                        </span>
                      }
                    />
                    <TooltipContent side="top">
                      Acknowledged by {ack.user?.name ?? "a teammate"}
                    </TooltipContent>
                  </Tooltip>
                </m.div>
              ))}
            </AnimatePresence>
          </div>
          {overflow > 0 && <span className="text-2xs text-muted-foreground">+{overflow}</span>}
          <span className="text-2xs text-muted-foreground">
            {acknowledgments.length === 1
              ? "1 acknowledgment"
              : `${acknowledgments.length} acknowledgments`}
          </span>
        </div>
      )}
      {acknowledgments.length === 0 &&
        comment.requiresAcknowledgment &&
        hasAcknowledged === false && (
          <span className="text-2xs text-muted-foreground">Acknowledgment requested</span>
        )}
    </div>
  );
}
