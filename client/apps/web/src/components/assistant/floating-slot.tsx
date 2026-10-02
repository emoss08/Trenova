import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * The place at the foot of a conversation where the composer floats, and
 * whatever stands in its place: the approval box while a decision waits, the
 * read-only notice once nothing can be sent. The thread runs to the bottom
 * and its last lines fade under the slot rather than stopping at a rule, and
 * the thread measures the slot through `ref` to pad its last message clear.
 */
export function FloatingSlot({
  compact = false,
  raised = false,
  className,
  children,
  ref,
}: {
  compact?: boolean;
  /** Stands above another slot while that one takes its place: the box on its way out. */
  raised?: boolean;
  /** Classes for the column the slot's content stands in. */
  className?: string;
  children: ReactNode;
  ref?: React.Ref<HTMLDivElement>;
}) {
  return (
    <div
      ref={ref}
      className={cn("pointer-events-none absolute inset-x-0 bottom-0", raised ? "z-20" : "z-10")}
    >
      <div
        aria-hidden
        className={cn(
          "from-popover pointer-events-none bg-gradient-to-t to-transparent",
          compact ? "h-6" : "h-10",
        )}
      />
      <div
        className={cn(
          "bg-popover pointer-events-auto",
          compact ? "px-3 pt-0.5 pb-1.5" : "px-4 pt-0.5 pb-2.5",
        )}
      >
        <div className={cn("mx-auto flex flex-col gap-1", !compact && "max-w-3xl", className)}>
          {children}
        </div>
      </div>
    </div>
  );
}
