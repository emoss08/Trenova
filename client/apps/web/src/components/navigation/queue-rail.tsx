import { cn } from "@trenova/shared/lib/utils";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

/*
 * The rail beside a work queue: its views, by state, with how much each holds.
 * The inbox and the intake queue both lead with one, and a person moving
 * between them should find the same thing in the same place.
 */

export function RailSection({
  title,
  action,
  children,
}: {
  title?: string;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-0.5">
      {title !== undefined && (
        <div className="flex items-center justify-between px-2 pb-1">
          <h3 className="text-foreground-subtle text-xs font-medium">{title}</h3>
          {action}
        </div>
      )}
      {children}
    </div>
  );
}

export function RailItem({
  icon: Icon,
  label,
  hint,
  count,
  loud = false,
  muted = false,
  active,
  onClick,
}: {
  icon: LucideIcon;
  label: string;
  hint?: string;
  count?: number;
  loud?: boolean;
  muted?: boolean;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      title={hint}
      className={cn(
        "ui-focus-ring flex h-8 w-full items-center gap-2.5 rounded-md px-2 text-left text-sm transition-colors",
        active
          ? "bg-nav-active text-nav-active-foreground"
          : "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
        muted && !active && "text-foreground-subtle",
      )}
    >
      <Icon className="size-4 shrink-0" aria-hidden />
      <span className={cn("min-w-0 flex-1 truncate", loud && "text-foreground font-medium")}>
        {label}
      </span>
      {count !== undefined && count > 0 && (
        <span
          className={cn(
            "shrink-0 text-xs tabular-nums",
            loud ? "text-foreground font-medium" : "text-foreground-subtle",
          )}
        >
          {count}
        </span>
      )}
    </button>
  );
}
