import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link } from "react-router";

export type ActionPillTone = "danger" | "warning" | "brand" | "neutral";

const TONE_DOT: Record<ActionPillTone, string> = {
  danger: "bg-danger",
  warning: "bg-warning",
  brand: "bg-brand",
  neutral: "bg-foreground-subtle",
};

/**
 * A chip that leads somewhere: a tone dot, what it says, and an arrow that
 * takes no room until the chip is hovered or focused, when the chip widens to
 * make space for it. A row of these keeps its spacing tight at rest.
 */
export function ActionPill({
  to,
  tone = "neutral",
  className,
  children,
}: {
  to: string;
  tone?: ActionPillTone;
  className?: string;
  children: ReactNode;
}) {
  return (
    <Link
      to={to}
      className={cn(
        "group border-border bg-card text-muted-foreground ui-focus-ring inline-flex h-6 items-center rounded-badge border px-2 text-xs transition-colors",
        "hover:bg-surface-hover hover:text-foreground",
        className,
      )}
    >
      <span aria-hidden className={cn("mr-1.5 size-1.5 shrink-0 rounded-full", TONE_DOT[tone])} />
      <span className="flex min-w-0 items-center gap-1">{children}</span>
      <span
        aria-hidden
        className={cn(
          "inline-flex w-0 shrink-0 overflow-hidden opacity-0 transition-[width,margin-left,opacity] duration-200",
          "group-hover:ml-1 group-hover:w-3 group-hover:opacity-100",
          "group-focus-visible:ml-1 group-focus-visible:w-3 group-focus-visible:opacity-100",
        )}
      >
        <ArrowRightIcon
          className={cn(
            "size-3 shrink-0 -translate-x-1 transition-transform duration-200",
            "group-hover:translate-x-0 group-focus-visible:translate-x-0",
          )}
        />
      </span>
    </Link>
  );
}
