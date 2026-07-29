import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { AnimatePresence, m } from "motion/react";
import type { ShipmentViewer } from "@/hooks/shipment-comments/use-shipment-viewers";

const MAX_VISIBLE_VIEWERS = 4;

export function ViewersStack({ viewers }: { viewers: ShipmentViewer[] }) {
  if (viewers.length === 0) return null;

  const visible = viewers.slice(0, MAX_VISIBLE_VIEWERS);
  const overflow = viewers.length - visible.length;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div className="flex shrink-0 items-center -space-x-2">
            <AnimatePresence initial={false}>
              {visible.map((viewer) => (
                <m.span
                  key={viewer.userId}
                  initial={{ scale: 0.5, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  exit={{ scale: 0.5, opacity: 0 }}
                  transition={{ type: "spring", stiffness: 500, damping: 30 }}
                  className="block rounded-full ring-2 ring-background"
                >
                  <ResolvedUserAvatar
                    userId={viewer.userId}
                    name={viewer.name}
                    className="size-6 rounded-full after:rounded-full"
                    imageClassName="rounded-full"
                    fallbackClassName="rounded-full text-2xs"
                  />
                </m.span>
              ))}
            </AnimatePresence>
            {overflow > 0 && (
              <span className="z-10 flex size-6 items-center justify-center rounded-full bg-muted text-2xs font-medium ring-2 ring-background">
                +{overflow}
              </span>
            )}
          </div>
        }
      />
      <TooltipContent side="top">
        Viewing now: {viewers.map((viewer) => viewer.name).join(", ")}
      </TooltipContent>
    </Tooltip>
  );
}
