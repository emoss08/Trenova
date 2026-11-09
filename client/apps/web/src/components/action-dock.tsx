import { cn } from "@trenova/shared/lib/utils";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { m, useReducedMotion } from "motion/react";
import type { ReactNode, Ref } from "react";

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
  /** The status block on the left of the dock; omit to show actions alone. */
  indicator?: ReactNode;
  /** Classes for the pill itself, such as a fixed width its content truncates within. */
  pillClassName?: string;
  /** The pill's element, for a popover that should sit over the whole dock. */
  pillRef?: Ref<HTMLDivElement>;
  children: ReactNode;
};

const ENTER = { duration: 0.24, ease: [0.16, 1, 0.3, 1] } as const;
const LEAVE = { duration: 0.16, ease: [0.4, 0, 1, 1] } as const;

/**
 * The floating pill that carries a surface's primary actions — the same shell
 * the form save dock uses, so every dock in the app sits in the same place and
 * wears the same surface. It slides up from below the viewport edge when it
 * mounts and back down when it leaves (a fade when the person asks for reduced
 * motion); the leave plays when the caller renders it inside `AnimatePresence`.
 */
export function ActionDock({
  position = "center",
  width,
  className,
  indicator,
  pillClassName,
  pillRef,
  children,
}: ActionDockProps) {
  const reduceMotion = useReducedMotion();
  const containerClassName = cn("fixed bottom-6 z-50", POSITION_CLASSES[position], className);
  const offstage = reduceMotion ? { opacity: 0 } : { opacity: 0, y: "150%" };
  const pill = (
    <div
      ref={pillRef}
      className={cn(
        "bg-foreground flex w-fit min-w-112.5 items-center gap-x-10 rounded-lg p-2",
        pillClassName,
      )}
    >
      {indicator}
      <div className="ml-auto flex shrink-0 items-center space-x-2">{children}</div>
    </div>
  );

  // The animation has to live on the fixed element itself. A transform on any
  // wrapper becomes the containing block for `position: fixed`, which pins the
  // dock wherever that wrapper sits in the document flow until the transform
  // clears — the dock renders high on the page, then jumps down.
  return (
    <m.div
      className={containerClassName}
      style={{ width }}
      initial={offstage}
      animate={{ opacity: 1, y: 0, transition: ENTER }}
      exit={{ ...offstage, transition: LEAVE }}
    >
      {pill}
    </m.div>
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
      {icon ?? (
        <AlertCircleIcon className="rounded-full bg-warning-subtle text-warning-foreground" />
      )}
      <div className="flex flex-col">
        <span className="text-background text-sm font-medium">{title}</span>
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
  // design-tokens-ignore: a glass overlay tints the surface behind it, so it is a translucent white and black rather than a theme colour.
  "border-none bg-white/20 text-background hover:bg-white/30 hover:text-background dark:bg-black/20 dark:hover:bg-black/30";
