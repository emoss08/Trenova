import { XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps, CSSProperties, ReactNode } from "react";

const DEFAULT_WIDTH = 372;

type FloatingPanelProps = Omit<ComponentProps<"aside">, "aria-label"> & {
  /** What the panel is, for assistive technology: "Brief", "Brief and activity". */
  label: string;
  /** Its width in pixels where there is room; it never takes more than its host leaves. */
  width?: number;
};

/**
 * A panel that floats over a workspace's right side: the shipment board's brief and
 * activity, the freight outlook's brief and its areas. It sits on top of the board instead
 * of beside it, so opening or closing it never resizes the board: nothing under it is laid
 * out or drawn again, and a board that is costly to draw does not stutter as the panel
 * arrives. It wears a stronger border and no shadow.
 *
 * The workspace it floats over is its host and must be `relative @container`: the panel is
 * placed against the host's edges, and in a host 820px wide or less it takes the whole
 * width. Render it only while it is open; it enters with `animate-panel-in`.
 */
export function FloatingPanel({
  label,
  width = DEFAULT_WIDTH,
  className,
  style,
  children,
  ...props
}: FloatingPanelProps) {
  return (
    <aside
      data-slot="floating-panel"
      aria-label={label}
      className={cn(
        "absolute top-2 right-3 bottom-3 z-30 flex w-[min(var(--floating-panel-width),calc(100%-28px))] animate-panel-in flex-col overflow-hidden rounded-lg border border-border-strong bg-raised motion-reduce:animate-none @max-[820px]:inset-x-2 @max-[820px]:w-auto",
        className,
      )}
      style={{ "--floating-panel-width": `${width}px`, ...style } as CSSProperties}
      {...props}
    >
      {children}
    </aside>
  );
}

/**
 * The panel's top row: what it is showing on the left and the button that closes it on the
 * right, on one centre line whatever the left side holds (a title, tabs, a way back).
 */
export function FloatingPanelHeader({
  onClose,
  className,
  children,
}: {
  onClose: () => void;
  className?: string;
  children: ReactNode;
}) {
  const t = useT();
  return (
    <div
      data-slot="floating-panel-header"
      className={cn(
        "flex min-h-11 shrink-0 items-center gap-2 border-b border-border px-3 py-2",
        className,
      )}
    >
      <div className="flex min-w-0 flex-1 items-center gap-2">{children}</div>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={t("Close panel")}
        className="shrink-0"
        onClick={onClose}
      >
        <XCloseIcon className="size-4" />
      </Button>
    </div>
  );
}
