import { useId } from "react";
import type { Box } from "./tour-geometry";

const RADIUS = 8;

/**
 * The dimmed page behind a tour step, with the step's part left clear. It covers the whole
 * window and takes the pointer, so nothing on the page is clicked by accident mid-tour. With
 * nothing lit the whole page is dimmed evenly.
 */
export function TourSpotlight({ lit }: { lit: Box | null }) {
  const maskId = useId();

  return (
    <svg
      aria-hidden
      data-slot="tour-spotlight"
      className="fixed inset-0 z-40 size-full animate-in fade-in-0 motion-reduce:animate-none"
    >
      <mask id={maskId}>
        <rect width="100%" height="100%" fill="white" />
        {lit ? (
          <rect
            x={lit.x}
            y={lit.y}
            width={lit.width}
            height={lit.height}
            rx={RADIUS}
            fill="black"
          />
        ) : null}
      </mask>
      <rect width="100%" height="100%" className="fill-scrim" mask={`url(#${maskId})`} />
      {lit ? (
        <rect
          x={lit.x}
          y={lit.y}
          width={lit.width}
          height={lit.height}
          rx={RADIUS}
          className="fill-none stroke-brand"
          strokeWidth={1.5}
        />
      ) : null}
    </svg>
  );
}
