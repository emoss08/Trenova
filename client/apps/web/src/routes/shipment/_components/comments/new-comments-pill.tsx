import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { ArrowDownIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";

export function NewCommentsPill({ count, onClick }: { count: number; onClick: () => void }) {
  const t = useT();

  return (
    <AnimatePresence>
      {count > 0 && (
        <m.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 8 }}
          transition={{ duration: 0.15, ease: "easeOut" }}
          className="absolute bottom-3 left-1/2 z-30 -translate-x-1/2"
        >
          <div className="border-border bg-popover rounded-full border p-0.5 shadow-md">
            <Button
              type="button"
              variant="ghost"
              size="xs"
              className="h-6 gap-1.5 rounded-full px-2.5 text-xs"
              onClick={onClick}
            >
              <ArrowDownIcon className="size-3" />
              {count === 1 ? t("1 new comment") : t("{0} new comments", count)}
            </Button>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}
