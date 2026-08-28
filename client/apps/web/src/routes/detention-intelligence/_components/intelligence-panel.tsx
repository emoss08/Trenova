import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { RotateCwIcon, TriangleAlertIcon } from "lucide-react";
import { m } from "motion/react";

type IconComponent = React.ComponentType<{ className?: string }>;

/**
 * The shared chrome for every panel on this page: hairline card, a titled
 * header with room for a control, and a staggered entrance so the page
 * assembles itself instead of appearing all at once.
 */
export function Panel({
  index,
  icon: Icon,
  title,
  description,
  action,
  footer,
  className,
  children,
}: {
  index: number;
  icon: IconComponent;
  title: string;
  description: string;
  action?: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <m.section
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.32, delay: index * 0.07, ease: [0.22, 1, 0.36, 1] }}
      className={cn(
        "border-border bg-card flex flex-col overflow-hidden rounded-lg border",
        className,
      )}
    >
      <header className="border-border flex items-start justify-between gap-3 border-b px-3 py-2.5">
        <div className="flex min-w-0 gap-2.5">
          <span className="bg-muted text-muted-foreground mt-px inline-flex size-6 shrink-0 items-center justify-center rounded-md">
            <Icon className="size-3.5" />
          </span>
          <div className="min-w-0">
            <h3 className="text-sm leading-tight font-medium">{title}</h3>
            <p className="text-2xs text-muted-foreground mt-1 leading-snug">{description}</p>
          </div>
        </div>
        {action ? <div className="shrink-0">{action}</div> : null}
      </header>

      <div className="min-w-0 flex-1">{children}</div>

      {footer ? <footer className="border-border border-t px-3 py-2">{footer}</footer> : null}
    </m.section>
  );
}

export function PanelEmpty({ icon: Icon, message }: { icon: IconComponent; message: string }) {
  return (
    <div className="flex flex-col items-center gap-2.5 px-4 py-10 text-center">
      <Icon className="text-muted-foreground/50 size-5" />
      <p className="text-muted-foreground max-w-[20rem] text-xs">{message}</p>
    </div>
  );
}

export function PanelError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center gap-2.5 px-4 py-10 text-center">
      <TriangleAlertIcon className="size-5 text-amber-500" />
      <p className="text-muted-foreground max-w-[20rem] text-xs">
        These figures could not be loaded. The window may be too wide, or the aggregation timed out.
      </p>
      <Button type="button" size="sm" variant="outline" className="h-7" onClick={onRetry}>
        <RotateCwIcon className="mr-1.5 size-3.5" />
        Try again
      </Button>
    </div>
  );
}

export function PanelRowsSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="divide-border divide-y">
      {Array.from({ length: rows }).map((_, index) => (
        <div key={index} className="flex items-center gap-3 px-3 py-3">
          <Skeleton className="size-3 shrink-0 rounded-sm" />
          <Skeleton className="h-3.5 min-w-0 flex-1 rounded-sm" />
          <Skeleton className="h-3.5 w-16 shrink-0 rounded-sm" />
          <Skeleton className="h-3.5 w-20 shrink-0 rounded-sm" />
        </div>
      ))}
    </div>
  );
}

/** A label-over-value pair, the unit every dense readout on this page is built from. */
export function MetricCell({
  label,
  value,
  detail,
  valueClassName,
  className,
}: {
  label: string;
  value: React.ReactNode;
  detail?: React.ReactNode;
  valueClassName?: string;
  className?: string;
}) {
  return (
    <div className={cn("min-w-0", className)}>
      <p className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">{label}</p>
      <p className={cn("mt-1 truncate text-sm font-medium tabular-nums", valueClassName)}>
        {value}
      </p>
      {detail ? (
        <p className="text-2xs text-muted-foreground mt-0.5 truncate tabular-nums">{detail}</p>
      ) : null}
    </div>
  );
}

/**
 * The "show everything" affordance shared by the ranked panels. Ranked lists
 * lead with the rows that matter, but the long tail has to stay reachable.
 */
export function PanelExpandToggle({
  expanded,
  hiddenCount,
  noun,
  onToggle,
}: {
  expanded: boolean;
  hiddenCount: number;
  noun: string;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="text-2xs text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 ml-auto rounded-sm font-medium transition-colors outline-none focus-visible:ring-[3px]"
    >
      {expanded ? "Show fewer" : `Show ${hiddenCount} more ${noun}`}
    </button>
  );
}
