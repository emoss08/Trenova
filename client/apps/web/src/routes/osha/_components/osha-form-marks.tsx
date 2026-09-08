import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * The letter or number a figure carries on the paper 300 and 300A. Somebody
 * reading this screen beside the form should find column H and type (3) in
 * the same places, so the marks are the form's own, set small and quiet.
 */
export function FormMark({
  children,
  title,
  className,
}: {
  children: ReactNode;
  title?: string;
  className?: string;
}) {
  return (
    <span
      title={title}
      className={cn(
        "inline-flex h-4 min-w-4 shrink-0 items-center justify-center rounded-sm border border-border bg-muted/60 px-1 text-[10px] leading-none font-semibold text-muted-foreground tabular-nums",
        className,
      )}
    >
      {children}
    </span>
  );
}

/** One boxed section of the 300A, titled the way the form titles it. */
export function FormBlock({
  title,
  children,
  className,
}: {
  title: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section aria-label={title} className={cn("flex min-w-0 flex-col gap-2", className)}>
      <h3 className="text-xs font-medium">{title}</h3>
      {children}
    </section>
  );
}

/** A figure with its mark and label, the cell of a form. */
export function FormFigure({
  mark,
  label,
  value,
  alarm,
  muted,
}: {
  mark: ReactNode;
  label: string;
  value: number;
  /** A figure that should never be anything but zero. */
  alarm?: boolean;
  muted?: boolean;
}) {
  return (
    <div role="group" aria-label={label} className="flex min-w-0 flex-col gap-1">
      <div className="flex items-center gap-1.5">
        <FormMark>{mark}</FormMark>
        <span className="text-muted-foreground truncate text-2xs leading-none">{label}</span>
      </div>
      <span
        className={cn(
          "text-lg leading-none font-semibold tabular-nums",
          alarm && "text-red-600 dark:text-red-400",
          muted && value === 0 && "text-muted-foreground/70",
        )}
      >
        {value.toLocaleString("en-US")}
      </span>
    </div>
  );
}
