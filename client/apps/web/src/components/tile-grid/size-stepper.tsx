import { Button } from "@trenova/shared/components/ui/button";
import type { LucideIcon } from "lucide-react";

export type SizeStepperProps = {
  icon: LucideIcon;
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (next: number) => void;
};

/**
 * Resizing is a stepper rather than a drag handle: a narrow tile clips its own
 * overflow, which used to strand a shrunken tile with its "wider" control cut
 * off. The stepper lives in a popover that stays open, so a size can be nudged
 * repeatedly.
 */
export function SizeStepper({ icon: Icon, label, value, min, max, onChange }: SizeStepperProps) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="text-muted-foreground size-3.5 shrink-0" />
      <span className="text-muted-foreground flex-1 text-xs">{label}</span>
      <Button
        variant="outline"
        size="icon"
        className="size-6"
        aria-label={`Decrease ${label.toLowerCase()}`}
        disabled={value <= min}
        onClick={() => onChange(value - 1)}
      >
        −
      </Button>
      <span className="w-5 text-center text-xs tabular-nums">{value}</span>
      <Button
        variant="outline"
        size="icon"
        className="size-6"
        aria-label={`Increase ${label.toLowerCase()}`}
        disabled={value >= max}
        onClick={() => onChange(value + 1)}
      >
        +
      </Button>
    </div>
  );
}
