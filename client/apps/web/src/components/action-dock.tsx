import { cn } from "@trenova/shared/lib/utils";
import { CircleAlert } from "lucide-react";
import { m } from "motion/react";
import type { ReactNode } from "react";

export type ActionDockPosition = "center" | "left" | "right";

const POSITION_CLASSES: Record<ActionDockPosition, string> = {
  center: "left-1/2 -translate-x-1/2",
  left: "left-20",
  right: "right-35",
};

export type ActionDockProps = {
  position?: ActionDockPosition;
  width?: string;
  className?: string;
  /** Slide the dock up from below the viewport edge when it mounts. */
  animated?: boolean;
  /** The status block on the left of the dock; omit to show actions alone. */
  indicator?: ReactNode;
  children: ReactNode;
};

/**
 * The floating pill that carries a surface's primary actions — the same shell
 * the form save dock uses, so every dock in the app sits in the same place and
 * wears the same surface.
 */
export function ActionDock({
  position = "center",
  width,
  className,
  animated = false,
  indicator,
  children,
}: ActionDockProps) {
  const containerClassName = cn("fixed bottom-6 z-50", POSITION_CLASSES[position], className);
  const pill = (
    <div className="flex w-fit min-w-112.5 items-center gap-x-10 rounded-lg bg-foreground p-2 shadow-lg">
      {indicator}
      <div className="ml-auto flex items-center space-x-2">{children}</div>
    </div>
  );

  // The entrance animation has to live on the fixed element itself. A transform
  // on any wrapper becomes the containing block for `position: fixed`, which
  // pins the dock wherever that wrapper sits in the document flow until the
  // transform clears — the dock renders high on the page, then jumps down.
  if (animated) {
    return (
      <m.div
        className={containerClassName}
        style={{ width }}
        initial={{ opacity: 0, y: "150%" }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: "150%" }}
        transition={{ duration: 0.25, ease: "easeOut" }}
      >
        {pill}
      </m.div>
    );
  }

  return (
    <div className={containerClassName} style={{ width }}>
      {pill}
    </div>
  );
}

export function ActionDockIndicator({
  icon,
  title,
  description,
}: {
  icon?: ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-center gap-x-3">
      {icon ?? <CircleAlert className="rounded-full bg-amber-400/10 text-amber-400 dark:text-amber-600" />}
      <div className="flex flex-col">
        <span className="text-sm font-medium text-background">{title}</span>
        <span className="text-2xs text-background/80">{description}</span>
      </div>
    </div>
  );
}

/**
 * The muted secondary-action style that reads against the dock's inverted
 * surface, where the standard outline button would vanish.
 */
export const ACTION_DOCK_SECONDARY_BUTTON =
  "border-none bg-white/20 text-background hover:bg-white/30 hover:text-background dark:bg-black/20 dark:hover:bg-black/30";
