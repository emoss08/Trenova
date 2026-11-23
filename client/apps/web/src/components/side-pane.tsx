import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps, CSSProperties, ReactNode, Ref } from "react";

export type SidePaneProps = Omit<ComponentProps<"aside">, "children"> & {
  open: boolean;
  /** The open width, as a CSS length. The content is laid out at this width while it folds. */
  width?: string;
  /** Receives the fixed-width box the content sits in. */
  innerRef?: Ref<HTMLDivElement>;
  innerClassName?: string;
  children?: ReactNode;
};

/**
 * A pane beside a list or a page that opens and closes by width, so what sits beside it gives up
 * and takes back the room rather than being covered. The content keeps its open width while the
 * pane folds, so nothing in it reflows; the content brings its own entrance (`animate-panel-in`,
 * keyed by what it shows). Closed, the pane is `inert` and hidden from assistive technology.
 */
export function SidePane({
  open,
  width = "25rem",
  innerRef,
  innerClassName,
  className,
  style,
  children,
  ...props
}: SidePaneProps) {
  return (
    <aside
      {...props}
      aria-hidden={!open}
      inert={!open}
      data-open={open || undefined}
      style={{ ...style, "--side-pane-width": width } as CSSProperties}
      className={cn(
        "min-h-0 shrink-0 overflow-hidden border-l transition-[width,opacity,border-color] duration-320 ease-settle motion-reduce:transition-none",
        open
          ? "w-(--side-pane-width) border-border opacity-100"
          : "w-0 border-transparent opacity-0",
        className,
      )}
    >
      <div
        ref={innerRef}
        className={cn("flex h-full w-(--side-pane-width) flex-col", innerClassName)}
      >
        {children}
      </div>
    </aside>
  );
}
