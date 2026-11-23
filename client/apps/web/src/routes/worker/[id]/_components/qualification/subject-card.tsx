import { Badge } from "@trenova/shared/components/ui/badge";
import type { ReactNode } from "react";

/** The record a pane acts on, at the top of its body: what it is, how it stands, its dates. */
export function SubjectCard({
  title,
  badge,
  detail,
}: {
  title: string;
  badge?: { label: string; variant: "success" | "warning" | "danger" | "neutral" | "info" } | null;
  detail?: ReactNode;
}) {
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-0.5 rounded-md border border-border-subtle bg-sunken px-2.5 py-2">
      <span className="truncate text-sm font-medium">{title}</span>
      {badge ? (
        <Badge variant={badge.variant} dot>
          {badge.label}
        </Badge>
      ) : (
        <span />
      )}
      {detail ? (
        <span className="col-span-2 truncate text-xs text-muted-foreground tabular-nums">
          {detail}
        </span>
      ) : null}
    </div>
  );
}
