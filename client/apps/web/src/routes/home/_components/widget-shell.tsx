import { SectionPanel } from "@/components/section-panel";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRowTotal } from "@trenova/shared/lib/row-total";
import { ArrowRightIcon, type IconComponent, Inbox01Icon } from "@trenova/shared/components/icons";
import { createContext, useContext, type ReactNode } from "react";
import { Link } from "react-router";

/**
 * True while the canvas is in edit mode. The canvas's edit bar becomes every
 * tile's header, so the shell drops its own header and footer rather than
 * stacking two title rows on one card.
 */
const WidgetEditingContext = createContext(false);

export const WidgetEditingProvider = WidgetEditingContext.Provider;

export function useWidgetEditing(): boolean {
  return useContext(WidgetEditingContext);
}

export type WidgetShellProps = {
  /** The widget's name in the reader's language, sentence case. */
  title: string;
  icon?: IconComponent;
  /** How many things the widget lists; drawn beside the title when above zero. */
  count?: number;
  /** False when `count` is the server's estimate of a list too large to count. */
  countExact?: boolean;
  /** A short muted note on the right of the header: an amount waiting, a window. */
  hint?: ReactNode;
  /** Controls that belong to the widget itself, on the right of the header. */
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

/** The tile draws the frame, so the panel inside it brings only its header and body. */
const PANEL_IN_TILE = "min-h-0 flex-1 rounded-none border-0 bg-transparent";

/**
 * The one chrome every home widget wears, and it is the app's own: the panel
 * header every dashboard uses (`SectionPanel`), a body, and exactly one way
 * through to the full view. Widgets supply content, never their own frame, so
 * a canvas of twenty cards reads as one surface and as part of the app.
 */
export function WidgetShell({
  title,
  icon: Icon,
  count,
  countExact = true,
  hint,
  actions,
  href,
  hrefLabel,
  scroll = true,
  className,
  bodyClassName,
  children,
}: WidgetShellProps) {
  const t = useT();
  const editing = useWidgetEditing();

  const body = (
    <div className={cn("flex min-h-0 flex-1 flex-col p-3", bodyClassName)}>{children}</div>
  );
  const content = scroll ? (
    <ScrollArea className="min-h-0 flex-1" maskVariant="card">
      {body}
    </ScrollArea>
  ) : (
    body
  );

  if (editing) {
    return <div className={cn("flex min-h-0 flex-1 flex-col", className)}>{content}</div>;
  }

  // A figure the server could not count is not drawn as one: it reads as an
  // estimate in words, beside whatever else the header notes.
  const estimate =
    count !== undefined && count > 0 && !countExact
      ? t("About {0}", formatRowTotal(count, false))
      : null;

  return (
    <SectionPanel
      title={title}
      icon={Icon ? <Icon /> : undefined}
      count={countExact ? count : undefined}
      hint={
        estimate && hint ? (
          <>
            {estimate} · {hint}
          </>
        ) : (
          (estimate ?? hint)
        )
      }
      action={actions}
      className={cn(PANEL_IN_TILE, className)}
    >
      {content}
      {href ? (
        <Link
          to={href}
          className="group/footer ui-inset-focus-ring flex h-8 shrink-0 items-center justify-between border-t border-border px-3 text-xs text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
        >
          <span>{hrefLabel ?? t("View all")}</span>
          <ArrowRightIcon className="size-3 transition-transform duration-200 group-hover/footer:translate-x-0.5" />
        </Link>
      ) : null}
    </SectionPanel>
  );
}

/**
 * An empty state that says what the emptiness means. "No unassigned loads" is
 * information; "No data" is an apology.
 */
export function WidgetEmpty({
  icon: Icon = Inbox01Icon,
  children,
}: {
  icon?: IconComponent;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 py-4 text-center">
      <Icon aria-hidden className="size-4 text-foreground-subtle" />
      <p className="max-w-56 text-xs text-muted-foreground">{children}</p>
    </div>
  );
}

/**
 * A tile before its widget is there: while it is still below the fold, and while
 * the widget's code is on its way. It is the shape every widget shares, a title
 * over a few rows, so the tile does not jump when the real one lands.
 */
export function WidgetTileSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy
      aria-label={t("Loading widget")}
      className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-3"
    >
      <Skeleton className="h-4 w-32 rounded-md" />
      <WidgetSkeleton rows={3} />
    </div>
  );
}

export function WidgetSkeleton({ rows = 3 }: { rows?: number }) {
  return (
    <div className="flex flex-col gap-1.5">
      {Array.from({ length: rows }, (_, index) => (
        <Skeleton key={index} className="h-6 w-full rounded-md" />
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
      <p className="max-w-56 text-xs text-muted-foreground">{message}</p>
      {actionLabel && onAction && (
        <Button variant="outline" size="xs" onClick={onAction}>
          {actionLabel}
        </Button>
      )}
    </div>
  );
}
