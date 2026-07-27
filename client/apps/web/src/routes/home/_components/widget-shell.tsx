import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router";

export type WidgetShellProps = {
  title: string;
  icon?: LucideIcon;
  /** Rendered beside the title as a count or a short status. */
  badge?: ReactNode;
  /** Controls that belong to the widget itself, left of the edit controls. */
  actions?: ReactNode;
  /** The single "see everything" destination. Home is a launcher: every widget has one. */
  href?: string;
  hrefLabel?: string;
  /** Set when the body scrolls itself, e.g. a list inside a fixed-height tile. */
  scroll?: boolean;
  className?: string;
  bodyClassName?: string;
  children: ReactNode;
};

/**
 * The one chrome every home widget wears: a micro label, an optional count, the
 * widget's own controls, and exactly one way through to the full view. Widgets
 * supply content, never their own frame, so a canvas of twenty cards still
 * reads as one surface.
 */
export function WidgetShell({
  title,
  icon: Icon,
  badge,
  actions,
  href,
  hrefLabel = "View all",
  scroll = true,
  className,
  bodyClassName,
  children,
}: WidgetShellProps) {
  const body = (
    <div className={cn("flex min-h-0 flex-1 flex-col p-3", bodyClassName)}>{children}</div>
  );

  return (
    <div className={cn("flex min-h-0 flex-1 flex-col", className)}>
      <div className="flex h-8 shrink-0 items-center gap-1.5 border-b border-border px-2.5">
        {Icon && <Icon className="size-3 shrink-0 text-muted-foreground" />}
        <span className="cc-label min-w-0 flex-1 truncate text-foreground">{title}</span>
        {badge}
        {actions}
      </div>

      {scroll ? (
        <ScrollArea className="min-h-0 flex-1" maskVariant="card">
          {body}
        </ScrollArea>
      ) : (
        body
      )}

      {href && (
        <Link
          to={href}
          className="flex h-7 shrink-0 items-center justify-between border-t border-border px-2.5 text-2xs text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
        >
          <span>{hrefLabel}</span>
          <ArrowRightIcon className="size-3" />
        </Link>
      )}
    </div>
  );
}

export function WidgetCount({
  value,
  tone = "muted",
}: {
  value: number | undefined;
  tone?: "muted" | "danger" | "warning" | "brand" | "success";
}) {
  if (value === undefined) return null;

  const toneClass = {
    muted: "bg-muted text-muted-foreground",
    danger: "bg-destructive/12 text-destructive",
    warning: "bg-warning/15 text-warning",
    brand: "bg-brand/15 text-brand",
    success: "bg-success/15 text-success",
  }[tone];

  return (
    <span
      className={cn(
        "inline-flex min-w-4.5 shrink-0 justify-center rounded px-1 font-table text-[9px] tabular-nums",
        toneClass,
      )}
    >
      {value > 999 ? "999+" : value}
    </span>
  );
}

/**
 * An empty state that says what the emptiness means. "No unassigned loads" is
 * information; "No data" is an apology.
 */
export function WidgetEmpty({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-1 py-4 text-center">
      <p className="text-2xs text-muted-foreground">{children}</p>
    </div>
  );
}

export function WidgetSkeleton({ rows = 3 }: { rows?: number }) {
  return (
    <div className="flex flex-col gap-1.5">
      {Array.from({ length: rows }, (_, index) => (
        <Skeleton key={index} className="h-6 w-full rounded" />
      ))}
    </div>
  );
}

/**
 * The action a widget offers when it cannot show anything useful yet — an
 * unconfigured report tile, say. It is a button rather than a message because
 * an author looking at it can fix it from here.
 */
export function WidgetNeedsSetup({
  message,
  actionLabel,
  onAction,
}: {
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 py-4 text-center">
      <p className="max-w-50 text-2xs text-muted-foreground">{message}</p>
      {actionLabel && onAction && (
        <Button variant="outline" size="xs" onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
