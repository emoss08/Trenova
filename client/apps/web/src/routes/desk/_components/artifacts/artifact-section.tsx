import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * The one rhythm every artifact body is set in, so a plan and a rate ledger
 * read as two pages of one document rather than two widgets.
 *
 * The body scrolls as a whole and is padded once; what it holds is blocks —
 * a titled section bordered by a hairline, a sunken well for prose — set a
 * gap apart. A table is the exception, bleeding edge to edge with a toolbar
 * over it and a footer under it, because a table's rows are its own rhythm.
 */
export function ArtifactScroll({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      data-slot="artifact-scroll"
      className={cn(
        "scrollbar-overlay flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4",
        className,
      )}
    >
      {children}
    </div>
  );
}

/**
 * A titled block inside an artifact: one quiet header line and a body. The
 * body supplies its own padding when it is a list that wants to run to the
 * edges, so `inset` is the default for prose and figures and off for rows.
 */
export function ArtifactSection({
  title,
  hint,
  action,
  inset = true,
  className,
  children,
}: {
  title: string;
  /** A short muted note beside the title: a count, a unit. */
  hint?: ReactNode;
  /** A control on the right of the header. */
  action?: ReactNode;
  inset?: boolean;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section
      aria-label={title}
      data-slot="artifact-section"
      className={cn(
        "border-border-subtle flex shrink-0 flex-col overflow-hidden rounded-lg border",
        className,
      )}
    >
      <header className="border-border-subtle bg-sunken/60 flex h-8 shrink-0 items-center gap-2 border-b px-3">
        <h4 className="truncate text-xs font-medium">{title}</h4>
        {hint !== undefined && hint !== null && hint !== "" && (
          <span className="text-foreground-subtle ml-auto truncate text-xs tabular-nums">
            {hint}
          </span>
        )}
        {action !== undefined && (
          <span className={cn("flex shrink-0 items-center", hint === undefined && "ml-auto")}>
            {action}
          </span>
        )}
      </header>
      <div className={cn("flex min-w-0 flex-col", inset && "px-3 py-2.5")}>{children}</div>
    </section>
  );
}

/** A sunken well for a stretch of prose: a message body, a summary. */
export function ArtifactWell({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      data-slot="artifact-well"
      className={cn(
        "bg-sunken rounded-lg px-4 py-3 text-sm leading-relaxed whitespace-pre-wrap",
        className,
      )}
    >
      {children}
    </div>
  );
}

/** The row of controls over a table or a document: a filter on the left, actions on the right. */
export function ArtifactToolbar({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      data-slot="artifact-toolbar"
      className={cn(
        "border-border-subtle flex h-9 shrink-0 items-center gap-1.5 border-b px-2",
        className,
      )}
    >
      {children}
    </div>
  );
}

/** The line under a table that says how much there is and what was asked. */
export function ArtifactFooter({ children }: { children: ReactNode }) {
  return (
    <p
      data-slot="artifact-footer"
      className="text-foreground-subtle border-border-subtle bg-sunken flex h-8 shrink-0 items-center gap-2 border-t px-3 text-xs tabular-nums"
    >
      {children}
    </p>
  );
}
