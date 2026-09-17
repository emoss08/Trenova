import { cn } from "@trenova/shared/lib/utils";
import { m } from "motion/react";

/**
 * The assistant's own mark: a four-point spark cut by the road it follows, with
 * a single travelling dot. Lucide's sparkles is the icon every product uses for
 * this; a transportation assistant should look like it belongs to this one.
 */
export function TrenovaSpark({ className, animated = false }: TrenovaSparkProps) {
  return (
    <svg
      className={cn("size-5", className)}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
      focusable="false"
    >
      <path
        d="M12 2.5 13.7 8.1a4 4 0 0 0 2.2 2.2l5.6 1.7-5.6 1.7a4 4 0 0 0-2.2 2.2L12 21.5l-1.7-5.6a4 4 0 0 0-2.2-2.2L2.5 12l5.6-1.7a4 4 0 0 0 2.2-2.2Z"
        fill="currentColor"
      />
      {animated ? (
        <m.circle
          r="1.6"
          fill="currentColor"
          className="opacity-70"
          animate={{ cx: [4.4, 12, 19.6, 12, 4.4], cy: [19.6, 22.2, 19.6, 17, 19.6] }}
          transition={{ duration: 6, repeat: Infinity, ease: "linear" }}
        />
      ) : (
        <circle cx="19.6" cy="19.6" r="1.6" fill="currentColor" className="opacity-70" />
      )}
    </svg>
  );
}

type TrenovaSparkProps = {
  className?: string;
  /** Sends the trailing dot around the mark. Used on hover, never at rest. */
  animated?: boolean;
};
