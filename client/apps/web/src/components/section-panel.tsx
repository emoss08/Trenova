import { InfoPopover } from "@/components/info-popover";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type SectionPanelProps = {
  title: string;
  icon?: ReactNode;
  /** A short muted note beside the title: a count, a window, a unit. */
  hint?: ReactNode;
  /** Shown only when above zero, so an empty panel does not announce "0". */
  count?: number;
  /** Controls on the right of the header: a button, a segmented control. */
  action?: ReactNode;
  /** A plain-words note behind an info icon: what the panel counts, what it leaves out. */
  help?: ReactNode;
  className?: string;
  children: ReactNode;
};

/**
 * The bordered card every HR console is built from: a one-line header and a
 * body that supplies its own padding, so a list can run edge to edge with
 * dividers while prose sits inside a padded block.
 */
export function SectionPanel({
  title,
  icon,
  hint,
  count,
  action,
  help,
  className,
  children,
}: SectionPanelProps) {
  return (
    <section
      aria-label={title}
      className={cn("bg-card flex flex-col overflow-hidden rounded-lg border", className)}
    >
      <header className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
        <div className="flex min-w-0 items-center gap-2">
          {icon ? (
            <span className="text-muted-foreground shrink-0 [&>svg]:size-3.5">{icon}</span>
          ) : null}
          <h3 className="truncate text-sm font-medium">{title}</h3>
          {help ? <InfoPopover title={title}>{help}</InfoPopover> : null}
          {count !== undefined && count > 0 ? (
            <span className="text-muted-foreground text-xs tabular-nums">{count}</span>
          ) : null}
        </div>
        <div className="flex min-w-0 shrink-0 items-center gap-2">
          {hint ? <span className="text-muted-foreground truncate text-xs">{hint}</span> : null}
          {action}
        </div>
      </header>
      {children}
    </section>
  );
}

/** One quiet line for a panel with nothing to list. */
export function SectionPanelQuiet({ children }: { children: ReactNode }) {
  return <p className="text-muted-foreground px-3 py-3 text-xs">{children}</p>;
}
