import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

interface BorderBeamProps {
  duration?: number;
  colorFrom?: string;
  colorTo?: string;
  /**
   * A spectrum for the beam to carry instead of the one colour fading out:
   * the stops run head to tail along the beam, so the first is its leading
   * edge and the last is where it fades to nothing.
   */
  colors?: readonly string[];
  /** Lays a blurred copy of the beam under the sharp one, so it casts light. */
  glow?: boolean;
  className?: string;
  borderWidth?: number;
}

const BEAM_HEAD = 60;
const BEAM_TAIL = 100;

function beamGradient(stops: readonly string[]) {
  const span = (BEAM_TAIL - BEAM_HEAD) / (stops.length + 1);
  const run = stops.map((stop, index) => `${stop} ${(BEAM_HEAD + span * (index + 1)).toFixed(2)}%`);
  return `conic-gradient(from var(--border-beam-angle, 0deg), transparent ${BEAM_HEAD}%, ${run.join(", ")}, transparent ${BEAM_TAIL}%)`;
}

export function BorderBeam({
  className,
  duration = 4,
  colorFrom = "var(--brand)",
  colorTo = "color-mix(in oklch, var(--brand) 10%, transparent)",
  colors,
  glow = false,
  borderWidth = 1.5,
}: BorderBeamProps) {
  const stops = colors && colors.length > 0 ? colors : [colorFrom, colorTo];
  const beam: CSSProperties = {
    padding: borderWidth,
    background: beamGradient(stops),
    WebkitMask: "linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)",
    WebkitMaskComposite: "xor",
    maskComposite: "exclude",
    animationName: "border-beam-spin",
    animationDuration: `${duration}s`,
    animationTimingFunction: "linear",
    animationIterationCount: "infinite",
  };
  return (
    <>
      {glow && (
        <div
          aria-hidden
          className={cn(
            "pointer-events-none absolute inset-0 rounded-[inherit] opacity-70 blur-[6px]",
            className,
          )}
          style={{ ...beam, padding: borderWidth * 3 }}
        />
      )}
      <div
        className={cn("pointer-events-none absolute inset-0 rounded-[inherit]", className)}
        style={beam}
      />
    </>
  );
}
