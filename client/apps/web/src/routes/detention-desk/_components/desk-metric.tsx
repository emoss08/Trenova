import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type DeskMetricProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  /** `lg` for the page rail, `md` inside the narrower claim file. */
  size?: "md" | "lg";
  valueClassName?: string;
  className?: string;
};

/**
 * One labelled number. Every figure on the desk — the floor rail and the claim
 * file alike — is set the same way, so a reader learns the shape once.
 * Renders `dt`/`dd`, so it belongs inside a `dl`.
 */
export function DeskMetric({
  label,
  value,
  sub,
  size = "lg",
  valueClassName,
  className,
}: DeskMetricProps) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      <dt className="text-2xs text-muted-foreground font-medium tracking-wide uppercase">
        {label}
      </dt>
      <dd
        className={cn(
          "leading-none font-medium tracking-tight tabular-nums",
          size === "lg" ? "text-2xl" : "text-lg",
          valueClassName,
        )}
      >
        {value}
      </dd>
      {sub ? <p className="text-muted-foreground truncate text-xs">{sub}</p> : null}
    </div>
  );
}
