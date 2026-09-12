import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";

/**
 * The thread as it will look with people talking in it: an avatar, the name
 * and how long ago, and a line or two of what they said. The last one is a
 * reply, set in under the one it answers.
 */
const GHOST_COMMENTS: readonly { name: string; body: readonly string[]; reply?: boolean }[] = [
  { name: "w-20", body: ["w-full", "w-3/5"] },
  { name: "w-16", body: ["w-4/5"] },
  { name: "w-24", body: ["w-2/3"], reply: true },
];

type CommentsEmptyProps = {
  title: string;
  description: string;
  /** Offered when a filter is what emptied the thread. */
  onClearFilters?: () => void;
  className?: string;
};

export function CommentsEmpty({
  title,
  description,
  onClearFilters,
  className,
}: CommentsEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      className={cn("h-full justify-center", className)}
      sketchClassName="max-w-sm"
      title={title}
      description={description}
      action={
        onClearFilters ? (
          <Button variant="outline" size="sm" onClick={onClearFilters}>
            <XIcon className="size-3.5" />
            {t("Clear filters")}
          </Button>
        ) : null
      }
      sketch={
        <div className="flex flex-col gap-3 text-left">
          {GHOST_COMMENTS.map((comment, index) => (
            <div key={index} className={cn("flex gap-3", comment.reply && "pl-11")}>
              <span className="bg-muted size-8 shrink-0 rounded-full" />
              <div className="flex min-w-0 flex-1 flex-col gap-2 pt-1">
                <span className="flex items-center gap-2">
                  <GhostLine className={`h-2 ${comment.name}`} />
                  <GhostLine className="w-8" />
                </span>
                {comment.body.map((width, line) => (
                  <GhostLine key={line} className={width} />
                ))}
              </div>
            </div>
          ))}
        </div>
      }
    />
  );
}
