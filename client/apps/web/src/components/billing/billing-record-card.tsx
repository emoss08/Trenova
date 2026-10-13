import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { cn } from "@trenova/shared/lib/utils";

type BillingRecordCardProps = {
  title: React.ReactNode;
  subtitle?: React.ReactNode;
  amount?: React.ReactNode;
  meta?: React.ReactNode;
  auxiliary?: React.ReactNode;
  isSelected: boolean;
  onClick: () => void;
  /**
   * Turns on bulk selection: a checkbox beside the card, separate from opening
   * it, so ticking a row never changes which record is open.
   */
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  checkLabel?: string;
};

export function BillingRecordCard({
  title,
  subtitle,
  amount,
  meta,
  auxiliary,
  isSelected,
  onClick,
  checked,
  onCheckedChange,
  checkLabel,
}: BillingRecordCardProps) {
  const card = (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex w-full min-w-0 flex-col gap-1 rounded-lg border p-3 text-left transition-colors",
        "hover:bg-muted/40",
        isSelected ? "border-border bg-muted" : "border-border/60",
        checked && !isSelected && "border-border bg-muted/40",
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
      {subtitle ? <span className="text-muted-foreground truncate text-xs">{subtitle}</span> : null}
      {meta ? <div className="mt-0.5">{meta}</div> : null}
    </button>
  );

  if (!onCheckedChange) {
    return card;
  }

  return (
    <div className="flex items-start gap-2">
      <Checkbox
        className="mt-3.5 shrink-0"
        checked={checked === true}
        onCheckedChange={(value) => onCheckedChange(value === true)}
        aria-label={checkLabel}
      />
      <div className="min-w-0 flex-1">{card}</div>
    </div>
  );
}
