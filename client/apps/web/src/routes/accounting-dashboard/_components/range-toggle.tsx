import { Button } from "@trenova/shared/components/ui/button";

export const TREND_RANGE_OPTIONS = [
  { label: "13w", value: 13 },
  { label: "26w", value: 26 },
  { label: "52w", value: 52 },
] as const;

export function RangeToggle({
  value,
  onChange,
}: {
  value: number;
  onChange: (value: number) => void;
}) {
  return (
    <div className="flex gap-1">
      {TREND_RANGE_OPTIONS.map((option) => (
        <Button
          key={option.value}
          type="button"
          size="sm"
          variant={value === option.value ? "secondary" : "ghost"}
          onClick={() => onChange(option.value)}
          className="h-7 px-2 text-xs"
        >
          {option.label}
        </Button>
      ))}
    </div>
  );
}
