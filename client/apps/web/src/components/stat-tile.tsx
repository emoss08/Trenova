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
          "border-amber-200 bg-amber-50/50 dark:border-amber-900 dark:bg-amber-950/30",
        tone === "danger" && "border-red-200 bg-red-50/50 dark:border-red-900 dark:bg-red-950/30",
        tone === "info" && "border-blue-200 bg-blue-50/50 dark:border-blue-900 dark:bg-blue-950/30",
        !tone && "bg-muted/30",
        clickable && "hover:bg-muted/60 cursor-pointer transition-colors",
        active && "ring-brand ring-1",
      )}
    >
      <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
        {label}
      </p>
      <div className="mt-1 text-sm font-semibold">{value}</div>
      <p className="text-muted-foreground mt-0.5 text-[11px]">{sub}</p>
    </Comp>
  );
}
