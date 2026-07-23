"use no memo";
import { Button } from "@trenova/shared/components/ui/button";
import { RefreshCwIcon, XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";

type DataTableRefreshPillProps = {
  visible: boolean;
  onRefresh: () => void;
  onDismiss: () => void;
};

export function DataTableRefreshPill({ visible, onRefresh, onDismiss }: DataTableRefreshPillProps) {
  return (
    <AnimatePresence>
      {visible && (
        <m.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -8 }}
          transition={{ duration: 0.15, ease: "easeOut" }}
          className="absolute top-2 left-1/2 z-30 -translate-x-1/2"
        >
          <div className="flex items-center gap-0.5 rounded-full border border-border bg-popover py-0.5 pr-0.5 pl-1 shadow-md">
            <Button
              type="button"
              variant="ghost"
              size="xs"
              className="h-6 gap-1.5 rounded-full px-2 text-xs"
              onClick={onRefresh}
            >
              <RefreshCwIcon className="size-3" />
              New data available
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              className="size-5 rounded-full text-muted-foreground"
              onClick={onDismiss}
              aria-label="Dismiss refresh notification"
            >
              <XIcon className="size-3" />
            </Button>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}
