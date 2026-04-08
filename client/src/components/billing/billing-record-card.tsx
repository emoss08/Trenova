import { cn } from "@/lib/utils";

type BillingRecordCardProps = {
  accentColor: string;
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  amount?: React.ReactNode;
  meta?: React.ReactNode;
  auxiliary?: React.ReactNode;
  isSelected: boolean;
  onClick: () => void;
};

export function BillingRecordCard({
  accentColor,
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
        "flex w-full items-stretch gap-2 rounded-md border p-2.5 text-left transition-colors",
        "hover:bg-accent/50",
        isSelected ? "border-border bg-muted" : "border-border",
      )}
    >
      <div className="w-[3px] shrink-0 rounded-full" style={{ backgroundColor: accentColor }} />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1.5">
            <div className="truncate text-sm font-semibold">{title}</div>
            {auxiliary ? <div className="shrink-0">{auxiliary}</div> : null}
          </div>
          {amount ? (
            <div className="shrink-0 text-xs font-medium tabular-nums">{amount}</div>
          ) : null}
        </div>
        {subtitle ? <div className="truncate text-xs text-muted-foreground">{subtitle}</div> : null}
        {meta ? <div>{meta}</div> : null}
      </div>
    </button>
  );
}
