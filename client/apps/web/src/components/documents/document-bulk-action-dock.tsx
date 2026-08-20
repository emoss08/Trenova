import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@trenova/shared/components/ui/tooltip";
import { CheckCheckIcon, Trash2Icon, XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";

interface DocumentBulkActionDockProps {
  selectedCount: number;
  totalCount: number;
  onDelete: () => void;
  onClearSelection: () => void;
  onSelectAll: () => void;
  isDeleting?: boolean;
}

export function DocumentBulkActionDock({
  selectedCount,
  totalCount,
  onDelete,
  onClearSelection,
  onSelectAll,
  isDeleting = false,
}: DocumentBulkActionDockProps) {
  const allSelected = selectedCount === totalCount;

  return (
    <AnimatePresence>
      {selectedCount > 0 && (
        <m.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 20 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          className="fixed bottom-6 left-1/2 z-50 -translate-x-1/2"
        >
          <div className="border-background/20 bg-foreground flex items-center gap-1 rounded-lg border px-2 py-1.5 shadow-lg">
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    variant="ghost"
                    className="border-muted-foreground/40 text-muted-foreground hover:bg-accent/20 dark:hover:bg-accent/20 h-7 cursor-pointer rounded-md border shadow-none"
                    size="sm"
                    onClick={onClearSelection}
                  >
                    <span className="text-background text-sm font-medium tabular-nums">
                      {selectedCount} selected
                    </span>{" "}
                    <XIcon className="text-background size-3" />
                  </Button>
                }
              />
              <TooltipContent sideOffset={10}>Clear selection</TooltipContent>
            </Tooltip>
            <div className="flex items-center gap-1 pl-1">
              {!allSelected && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-background hover:bg-accent/20 hover:text-background dark:hover:bg-accent/20 h-7 gap-1.5 px-2"
                  onClick={onSelectAll}
                >
                  <CheckCheckIcon className="size-4" />
                  Select All
                </Button>
              )}
              <Button
                variant="destructive"
                size="sm"
                disabled={isDeleting}
                className="h-7 gap-1.5 px-2"
                onClick={onDelete}
              >
                {isDeleting ? <Spinner /> : <Trash2Icon className="size-4" />}
                {isDeleting ? "Deleting..." : "Delete"}
              </Button>
            </div>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}
