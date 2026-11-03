import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";

interface BorderBeamProps {
  duration?: number;
  colorFrom?: string;
  colorTo?: string;
  className?: string;
  borderWidth?: number;
}

/**
 * A beam of colour that runs round the parent's border while it works. The ring
 * is masked once and the gradient behind it turns (`ui-spin-ring`), so the
 * browser composites the beam rather than repainting it each frame.
 */
export function BorderBeam({
  className,
  duration = 4,
  colorFrom = "var(--brand)",
  colorTo = "color-mix(in oklch, var(--brand) 10%, transparent)",
  borderWidth = 1.5,
}: BorderBeamProps) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "ui-spin-ring pointer-events-none absolute inset-0 rounded-[inherit]",
        className,
      )}
      style={
        {
          padding: borderWidth,
          "--spin-gradient": `conic-gradient(transparent 60%, ${colorFrom} 78%, ${colorTo} 92%, transparent 100%)`,
          "--spin-duration": `${duration}s`,
          WebkitMask: "linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)",
          WebkitMaskComposite: "xor",
          maskComposite: "exclude",
        } as CSSProperties
      }
    />
  );
}
