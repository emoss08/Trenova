import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type EditFieldRowProps = {
  label: ReactNode;
  hint?: ReactNode;
  /** The field's error, shown under the control in place of nothing. */
  error?: string;
  htmlFor?: string;
  /** Stack the label over the control, for a control as wide as the sheet. */
  stacked?: boolean;
  className?: string;
  children: ReactNode;
};

/**
 * One row of an editor: the label and its hint in a fixed column, the control beside it.
 * Every editor lays its fields out with this, so they line up from one sheet to the next.
 */
export function EditFieldRow({
  label,
  hint,
  error,
  htmlFor,
  stacked = false,
  className,
  children,
}: EditFieldRowProps) {
  return (
    <div
      className={cn(
        "grid gap-x-4 gap-y-1.5",
        stacked ? "grid-cols-1" : "grid-cols-1 sm:grid-cols-[190px_minmax(0,1fr)]",
        className,
      )}
    >
      <div className="flex flex-col gap-0.5 pt-1.5">
        <label htmlFor={htmlFor} className="text-xs font-medium">
          {label}
        </label>
        {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
      </div>
      <div className="flex min-w-0 flex-col gap-1">
        {children}
        {error && <p className="text-xs text-danger">{error}</p>}
      </div>
    </div>
  );
}
