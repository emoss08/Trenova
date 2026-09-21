import { AgentTile } from "@/components/agent-identity/agent-tile";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ListChecksIcon } from "lucide-react";
import { useEffect, useRef } from "react";
import type { DecisionRowView } from "./decision-presenters";

/**
 * One thing waiting on a person. The row says who proposed it and what in
 * a sentence; the detail beside the list says why and what it would change.
 * Focus is the keyboard's cursor; the checkbox is the batch.
 */
export function DecisionRow({
  row,
  focused,
  selected,
  now,
  onFocus,
  onToggle,
}: {
  row: DecisionRowView;
  focused: boolean;
  selected: boolean;
  now: number;
  onFocus: () => void;
  onToggle: () => void;
}) {
  const t = useT();
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (focused) {
      ref.current?.scrollIntoView({ block: "nearest" });
    }
  }, [focused]);

  return (
    <div
      ref={ref}
      role="option"
      aria-selected={focused}
      data-decision-row
      tabIndex={-1}
      onClick={onFocus}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          onFocus();
        }
      }}
      className={cn(
        "group flex cursor-pointer items-start gap-3 px-3 py-2.5 transition-colors",
        focused ? "bg-surface-selected" : "hover:bg-surface-hover",
      )}
    >
      <span className="pt-0.5" onClick={(event) => event.stopPropagation()}>
        <Checkbox
          checked={selected}
          disabled={!row.batchable}
          aria-label={row.batchable ? t("Include in batch") : t("A plan is decided on its own")}
          onCheckedChange={() => onToggle()}
        />
      </span>
      <AgentTile agent={row.agent} size="md" className="mt-0.5" />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex items-center gap-2">
          <span className="text-muted-foreground min-w-0 truncate text-xs">
            {row.agent?.name ?? t("Retired agent")} · {row.title}
          </span>
          {row.kind === "plan" && (
            <Badge variant="neutral" className="h-4 shrink-0 gap-1 px-1 text-2xs">
              <ListChecksIcon className="size-3" />
              {t("{0, plural, one {# step} other {# steps}}", row.stepCount)}
            </Badge>
          )}
          <span className="text-muted-foreground ml-auto shrink-0 text-xs tabular-nums">
            {formatSecondsAgo(now - row.createdAt)}
          </span>
        </span>
        <span className="line-clamp-2 text-sm leading-snug">{row.summary}</span>
      </span>
    </div>
  );
}
