import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

export function StatTile({
  label,
  value,
  sub,
  hint,
  tone,
  clickable,
  onClick,
  active,
}: {
  label: string;
  value: ReactNode;
  sub: ReactNode;
  hint: string;
  tone?: "warn" | "info" | "danger";
  clickable?: boolean;
  onClick?: () => void;
  active?: boolean;
}) {
  const Comp = clickable ? "button" : "div";
  return (
    <Comp
      type={clickable ? "button" : undefined}
      onClick={onClick}
      title={hint}
      className={cn(
        "rounded-lg border p-3 text-left",
        tone === "warn" &&
          "border-warning-border bg-warning-subtle/50 dark:border-warning-border dark:bg-warning-subtle/30",
        tone === "danger" && "border-danger-border bg-danger-subtle/50 dark:border-danger-border dark:bg-danger-subtle/30",
        tone === "info" && "border-info-border bg-info-subtle/50 dark:border-info-border dark:bg-info-subtle/30",
        !tone && "bg-muted/30",
        clickable && "hover:bg-muted/60 cursor-pointer transition-colors",
        active && "ring-brand ring-1",
      )}
    >
      <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
        {label}
      </p>
      <div className="mt-1 text-sm font-semibold">{value}</div>
      <p className="text-muted-foreground mt-0.5 text-xs">{sub}</p>
    </Comp>
  );
}
