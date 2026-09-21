import { ASSIST_MARK_DIAMOND } from "@trenova/shared/components/ui/assist-mark";
import { cn } from "@trenova/shared/lib/utils";
import { m } from "motion/react";

/**
 * The assistant's own mark: the advisory diamond of AssistMark, drawn here
 * rather than reused so its centre point can answer the pointer.
 */
export function AssistantMark({ className, animated = false }: AssistantMarkProps) {
  return (
    <svg
      className={cn("size-5", className)}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      <path d={ASSIST_MARK_DIAMOND} />
      {animated ? (
        <m.circle
          cx="12"
          cy="12"
          fill="currentColor"
          initial={{ r: 2 }}
          animate={{ r: [2, 3.4, 2] }}
          transition={{ duration: 1.6, repeat: Infinity, ease: "easeInOut" }}
        />
      ) : (
        <circle cx="12" cy="12" r="2" fill="currentColor" />
      )}
    </svg>
  );
}

type AssistantMarkProps = {
  className?: string;
  /** Breathes the centre point. Used on hover, never at rest. */
  animated?: boolean;
};
