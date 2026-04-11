import { cn } from "@/lib/utils";

type BillingRecordCardProps = {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  amount?: React.ReactNode;
  meta?: React.ReactNode;
  auxiliary?: React.ReactNode;
  isSelected: boolean;
  onClick: () => void;
};

export function BillingRecordCard({
  title,
  subtitle,
  amount,
  meta,
  auxiliary,
  isSelected,
  onClick,
}: BillingRecordCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full flex-col gap-1 rounded-lg border p-3 text-left transition-colors",
        "hover:bg-muted/40",
        isSelected ? "border-border bg-muted" : "border-border/60",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="truncate text-sm font-semibold">{title}</span>
          {auxiliary ? <span className="shrink-0">{auxiliary}</span> : null}
        </div>
        {amount ? (
          <span className="shrink-0 text-sm font-semibold tabular-nums">{amount}</span>
        ) : null}
      </div>
      {subtitle ? <span className="truncate text-xs text-muted-foreground">{subtitle}</span> : null}
      {meta ? <div className="mt-0.5">{meta}</div> : null}
    </button>
  );
}
