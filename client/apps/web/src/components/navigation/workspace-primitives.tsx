import { cn } from "@trenova/shared/lib/utils";
import type { ComponentType, ReactNode } from "react";
import { Link } from "react-router";

export const WORKSPACE_SIDEBAR_WIDTH_CLASS = "w-56";

/**
 * One row of the workspace sidebar: 28px tall, quiet until hovered, tinted
 * with the brand when it is the page you are on. Icons are optional because
 * most pages are named, not pictured.
 */
export function WorkspaceNavRow({
  to,
  active,
  disabled,
  icon: Icon,
  sub,
  onClick,
  children,
  className,
}: {
  to: string;
  active?: boolean;
  disabled?: boolean;
  icon?: ComponentType<{ className?: string; strokeWidth?: number }>;
  /** A row under a group label sits a little tighter and lighter. */
  sub?: boolean;
  onClick?: () => void;
  children: ReactNode;
  className?: string;
}) {
  return (
    <Link
      to={to}
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      aria-disabled={disabled}
      tabIndex={disabled ? -1 : undefined}
      className={cn(
        "flex items-center gap-2 rounded-md pr-2 pl-2 transition-colors outline-none",
        "focus-visible:ring-ring/50 focus-visible:ring-2",
        sub ? "h-6.5 text-sm" : "h-7 text-base",
        active
          ? "bg-nav-active text-nav-active-foreground font-semibold"
          : cn("text-foreground hover:bg-muted", sub && "text-foreground/85"),
        disabled && "pointer-events-none opacity-40",
        className,
      )}
    >
      {Icon && (
        <Icon
          className={cn(
            "size-3.5 shrink-0",
            active ? "text-nav-active-foreground" : "text-muted-foreground",
          )}
          strokeWidth={1.75}
        />
      )}
      {children}
    </Link>
  );
}

export function WorkspaceRowLabel({ children }: { children: ReactNode }) {
  return <span className="min-w-0 flex-1 truncate">{children}</span>;
}

/**
 * A group heading inside a module. It shares the rows' left edge, and reads
 * as structure rather than as another link.
 */
export function WorkspaceGroupLabel({ children }: { children: ReactNode }) {
  return (
    <span className="text-muted-foreground block truncate pt-3 pr-2 pb-1 pl-2 text-xs font-semibold tracking-wider uppercase select-none">
      {children}
    </span>
  );
}

export function WorkspaceDivider({ className }: { className?: string }) {
  return <div aria-hidden className={cn("bg-border mx-4 my-1.5 h-px shrink-0", className)} />;
}

/**
 * The tinted square that stands for a module wherever the workspace shows
 * one: the sidebar head, the modules menu, the collapsed rail.
 */
export function ModuleTile({
  icon: Icon,
  active = true,
  size = "md",
  className,
}: {
  icon: ComponentType<{ className?: string; strokeWidth?: number }>;
  active?: boolean;
  size?: "sm" | "md";
  className?: string;
}) {
  return (
    <span
      className={cn(
        "flex shrink-0 items-center justify-center",
        size === "md" ? "size-7 rounded-md" : "size-6 rounded-md",
        active ? "bg-brand/10 text-nav-active-foreground" : "bg-muted text-foreground",
        className,
      )}
    >
      <Icon className={size === "md" ? "size-[15px]" : "size-3.5"} strokeWidth={1.75} />
    </span>
  );
}
