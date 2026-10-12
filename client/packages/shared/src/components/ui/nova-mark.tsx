import { NovaRingIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import { useId } from "react";

/**
 * Nova's mark: the ring from the custom icon set, stroked with a gradient
 * across three categorical accents so it reads as Nova's own and never as a
 * tone.
 *
 * It appears in exactly three places — the Briefing item in Triage's rail,
 * "Drafted by Nova" on a reply, and Nova's card in the Briefing deck. Machine
 * suggestions anywhere else keep `AssistMark`, and the Desk never wears it.
 *
 * `spin` turns it: "hover" while the pointer is on the nearest `group`
 * ancestor (the rail item), "always" while Nova is working. Reduced motion
 * keeps it still.
 */
export function NovaMark({
  className,
  spin = "none",
  label,
}: {
  /** Size it with a class (`size-3.5`); the stroke scales with it. */
  className?: string;
  spin?: "none" | "hover" | "always";
  /** Names the mark for assistive technology; omitted, the mark is decoration. */
  label?: string;
}) {
  const gradientId = `nova-${useId().replace(/[^a-zA-Z0-9_-]/g, "")}`;

  return (
    <span
      role={label === undefined ? undefined : "img"}
      aria-label={label}
      aria-hidden={label === undefined ? true : undefined}
      className={cn("relative inline-flex size-4 shrink-0", className)}
    >
      <svg aria-hidden className="absolute size-0" focusable="false">
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" style={{ stopColor: "var(--accent-indigo)" }} />
            <stop offset="0.5" style={{ stopColor: "var(--accent-violet)" }} />
            <stop offset="1" style={{ stopColor: "var(--accent-rose)" }} />
          </linearGradient>
        </defs>
      </svg>
      <NovaRingIcon
        color={`url(#${gradientId})`}
        strokeWidth={4}
        className={cn(
          "size-full",
          spin === "hover" && "group-hover:animate-nova-spin",
          spin === "always" && "animate-nova-spin",
        )}
      />
    </span>
  );
}
