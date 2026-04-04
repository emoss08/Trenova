import { cn } from "@/lib/utils";

interface BorderBeamProps {
  duration?: number;
  colorFrom?: string;
  colorTo?: string;
  className?: string;
  borderWidth?: number;
}

export function BorderBeam({
  className,
  duration = 4,
  colorFrom = "oklch(0.55 0.22 263)",
  colorTo = "oklch(0.55 0.22 263 / 0.1)",
  borderWidth = 1.5,
}: BorderBeamProps) {
  return (
    <div
      className={cn(
        "pointer-events-none absolute inset-0 rounded-[inherit]",
        className,
      )}
      style={{
        padding: borderWidth,
        background: `conic-gradient(from var(--border-beam-angle, 0deg), transparent 60%, ${colorFrom} 78%, ${colorTo} 92%, transparent 100%)`,
        WebkitMask:
          "linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0)",
        WebkitMaskComposite: "xor",
        maskComposite: "exclude",
        animationName: "border-beam-spin",
        animationDuration: `${duration}s`,
        animationTimingFunction: "linear",
        animationIterationCount: "infinite",
      }}
    />
  );
}
