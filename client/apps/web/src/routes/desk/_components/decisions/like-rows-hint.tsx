import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { ListChecksIcon } from "lucide-react";

/**
 * Says, while nothing is marked, that the focused change has others like it
 * waiting, and marks them all in one click, so the batch the queue already
 * supports is found rather than stumbled on.
 */
export function LikeRowsHint({
  count,
  title,
  onSelect,
}: {
  count: number;
  title: string;
  onSelect: () => void;
}) {
  const t = useT();

  return (
    <div className="border-border flex items-center gap-2 border-b px-3 py-1.5 text-xs">
      <ListChecksIcon className="text-foreground-muted size-3.5 shrink-0" />
      <span className="text-foreground-muted min-w-0 flex-1 truncate">
        {t(
          "{0, plural, one {# change like this is waiting} other {# changes like this are waiting}}: {1}",
          count,
          title,
        )}
      </span>
      <Button size="xs" variant="outline" onClick={onSelect}>
        {t("Select all {0}", count)}
      </Button>
    </div>
  );
}
