import { Card } from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

type StatCardProps = {
  icon: LucideIcon;
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  tone?: "default" | "success" | "warning";
  isLoading?: boolean;
  onClick?: () => void;
};

const TONE_CLASS: Record<NonNullable<StatCardProps["tone"]>, string> = {
  default: "bg-muted text-muted-foreground",
  success: "bg-success/15 text-success-foreground",
  warning: "bg-warning/15 text-warning-foreground",
};

export function StatCard({
  icon: Icon,
  label,
  value,
  hint,
  tone = "default",
  isLoading = false,
  onClick,
}: StatCardProps) {
  const body = (
    <>
      <span
        className={cn(
          "flex size-9 shrink-0 items-center justify-center rounded-lg",
          TONE_CLASS[tone],
        )}
      >
        <Icon className="size-4" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-muted-foreground text-xs font-medium">{label}</p>
        {isLoading ? (
          <Skeleton className="mt-1 h-6 w-16" />
        ) : (
          <p className="text-xl font-semibold tabular-nums">{value}</p>
        )}
        {hint && !isLoading ? (
          <p className="text-muted-foreground mt-0.5 truncate text-xs">{hint}</p>
        ) : null}
      </div>
    </>
  );

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        className="border-border bg-card text-card-foreground hover:border-primary/40 hover:bg-muted/40 focus-visible:ring-ring/50 flex cursor-pointer flex-row items-center gap-3 rounded-xl border p-4 text-left text-sm transition-colors outline-none focus-visible:ring-[3px]"
      >
        {body}
      </button>
    );
  }

  return <Card className="flex flex-row items-center gap-3 p-4">{body}</Card>;
}
