import { cn } from "@trenova/shared/lib/utils";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

/** The house settle curve (`ease-settle`), for motion's own transitions. */
const SETTLE = [0.16, 1, 0.3, 1] as const;

/**
 * Settings that are there only while something else is on. They open by growing to their
 * height and close by giving it back, so the rows beneath move once instead of jumping; a
 * person who asked for reduced motion gets the cut. Nothing is rendered while closed.
 */
export function SettingsReveal({
  show,
  className,
  children,
}: {
  show: boolean;
  className?: string;
  children: ReactNode;
}) {
  const reduceMotion = useReducedMotion();

  return (
    <AnimatePresence initial={false}>
      {show ? (
        <m.div
          key="revealed"
          data-slot="settings-reveal"
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: "auto", opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={reduceMotion ? { duration: 0 } : { duration: 0.3, ease: SETTLE }}
          className={cn("overflow-hidden", className)}
        >
          {children}
        </m.div>
      ) : null}
    </AnimatePresence>
  );
}
