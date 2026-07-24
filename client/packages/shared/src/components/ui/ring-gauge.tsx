import { cn } from "@trenova/shared/lib/utils";

export type RingGaugeTone = "brand" | "warning" | "critical" | "success" | "muted";

const TONE_CLASS: Record<RingGaugeTone, string> = {
  brand: "text-brand",
  warning: "text-warning",
  critical: "text-destructive",
  success: "text-success",
  muted: "text-muted-foreground",
};

export interface RingGaugeProps {
  value: number;
  size?: number;
  strokeWidth?: number;
  tone?: RingGaugeTone;
  trackClassName?: string;
  className?: string;
  children?: React.ReactNode;
  "aria-label"?: string;
}

export function RingGauge({
  value,
  size = 96,
  strokeWidth = 6,
  tone = "brand",
  trackClassName,
  className,
  children,
  "aria-label": ariaLabel,
}: RingGaugeProps) {
  const clamped = Math.min(1, Math.max(0, value));
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference * (1 - clamped);

  return (
    <div
      role="meter"
      aria-label={ariaLabel}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.round(clamped * 100)}
      className={cn("relative inline-flex items-center justify-center", className)}
      style={{ width: size, height: size }}
    >
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        className="-rotate-90"
        aria-hidden
      >
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          className={cn("stroke-current text-border/80", trackClassName)}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className={cn(
            "stroke-current transition-[stroke-dashoffset,color] duration-700 ease-out motion-reduce:transition-none",
            TONE_CLASS[tone],
          )}
        />
      </svg>
      {children ? (
        <div className="absolute inset-0 flex items-center justify-center">{children}</div>
      ) : null}
    </div>
  );
}
