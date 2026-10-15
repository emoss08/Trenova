import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { CardIcon } from "./record-card";

/** One step of a record's way: behind it, where it stands, ahead, or struck off with the rest. */
export type CardTrackStep = {
  key: string;
  label: string;
  state: "done" | "now" | "next" | "off";
};

/**
 * A record's way from its first state to its last as one track of segments,
 * each naming its step: the steps behind it ticked, the one it stands at
 * filled in ink, the ones ahead left open. A record that left the way (an
 * order canceled, a settlement voided) leaves the whole track struck through.
 * The line beneath says where it stands in words: which step and since when,
 * or what happens next.
 */
export function CardTrack({
  steps,
  label,
  off = false,
  note,
}: {
  steps: readonly CardTrackStep[];
  /** What the way is, for a screen reader: an order's lifecycle, a settlement's. */
  label: string;
  off?: boolean;
  note: ReactNode;
}) {
  const last = steps.length - 1;
  return (
    <div className={cn("dk-rec-track", off && "dk-off")}>
      <ol aria-label={label}>
        {steps.map((step, index) => (
          <li
            key={step.key}
            className={`dk-${step.state}`}
            aria-current={step.state === "now" ? "step" : undefined}
          >
            {(step.state === "done" || (step.state === "now" && index === last)) && (
              <CardIcon name="check" size={11} stroke={2.8} />
            )}
            <span>{step.label}</span>
          </li>
        ))}
      </ol>
      <p>{note}</p>
    </div>
  );
}
