import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/** The space a step's inline work takes under its title, aligned with the title's text. */
export function StepBody({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <div className={cn("animate-rise flex flex-col gap-2.5 pt-0.5 pb-3.5 pl-7.75", className)}>
      {children}
    </div>
  );
}

/**
 * A step's buttons. With a note, the note fills the left and the buttons sit
 * right; without one, the children lay themselves out (a spacer where needed).
 */
export function StepFooter({
  note,
  noteTone = "subtle",
  children,
}: {
  note?: ReactNode;
  noteTone?: "subtle" | "warning";
  children: ReactNode;
}) {
  return (
    <div className="flex min-w-0 items-center gap-1.5">
      {note !== undefined && (
        <span
          className={cn(
            "min-w-0 flex-1 truncate text-xs",
            noteTone === "warning" ? "text-warning" : "text-foreground-subtle",
          )}
        >
          {note}
        </span>
      )}
      {children}
    </div>
  );
}

/** Why the step's last try did not go through, in place. */
export function StepError({ children }: { children: ReactNode }) {
  return (
    <div
      role="alert"
      className="bg-danger-subtle text-danger-subtle-foreground animate-rise flex gap-2 rounded-md px-2.5 py-2 text-sm leading-normal"
    >
      <AlertCircleIcon aria-hidden className="mt-0.5 size-3.25 shrink-0" />
      <span className="min-w-0">{children}</span>
    </div>
  );
}

/**
 * A control in a step with a short word before it ("As", "On"). The control
 * carries its own accessible name; the word is what a sighted reader scans.
 */
export function StepField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-1 items-center gap-2">
      <span aria-hidden className="text-foreground-subtle w-5 shrink-0 text-xs">
        {label}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}
