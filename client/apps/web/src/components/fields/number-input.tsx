import { cn } from "@trenova/shared/lib/utils";
import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";
import { NumericFormat, type NumericFormatProps } from "react-number-format";

type NumberInputProps = Omit<
  NumericFormatProps,
  "onValueChange" | "value" | "defaultValue" | "type"
> & {
  value?: string | number | null;
  onValueChange: (value: string) => void;
  sideText?: string;
  min?: number;
  max?: number;
  step?: number;
};

export function NumberInput({
  className,
  value,
  onValueChange,
  sideText,
  decimalScale = 0,
  fixedDecimalScale = false,
  allowNegative = false,
  thousandSeparator,
  prefix,
  suffix,
  min,
  max,
  step = 1,
  disabled,
  readOnly,
  ...props
}: NumberInputProps) {
  const currentValue =
    typeof value === "number" ? value : Number(value && value !== "" ? value : 0);

  const applyNumericValue = (nextValue: number) => {
    let boundedValue = nextValue;

    if (min !== undefined) {
      boundedValue = Math.max(boundedValue, min);
    }
    if (max !== undefined) {
      boundedValue = Math.min(boundedValue, max);
    }

    if (fixedDecimalScale && decimalScale !== undefined) {
      onValueChange(boundedValue.toFixed(decimalScale));
      return;
    }

    onValueChange(String(boundedValue));
  };

  return (
    <div className="relative w-full">
      <NumericFormat
        value={value ?? ""}
        onValueChange={(values) => {
          onValueChange(values.value);
        }}
        decimalScale={decimalScale}
        fixedDecimalScale={fixedDecimalScale}
        allowNegative={allowNegative}
        thousandSeparator={thousandSeparator}
        prefix={prefix}
        suffix={suffix}
        disabled={disabled}
        readOnly={readOnly}
        className={cn(
          "border-input bg-muted flex h-7 w-full min-w-0 rounded-md border px-2 py-0.5 text-base outline-none md:text-sm",
          "placeholder:text-muted-foreground",
          "focus-visible:border-brand focus-visible:ring-brand/20 focus-visible:ring-4 focus-visible:outline-hidden",
          "disabled:bg-input/50 disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
          "transition-[border-color,box-shadow] duration-200 ease-in-out",
          sideText ? "pr-16" : "pr-12",
          className,
        )}
        {...props}
      />

      <div className="absolute top-px right-px bottom-px flex h-6 items-center gap-1 pr-0">
        {sideText ? (
          <div className="text-muted-foreground pointer-events-none mr-2 text-xs select-none">
            {sideText}
          </div>
        ) : null}
        <div className="border-muted-foreground/20 absolute top-px right-px bottom-px flex h-6 flex-col items-stretch rounded-r-md border-l bg-transparent">
          <button
            type="button"
            aria-label="Increment"
            className="border-muted-foreground/20 text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground inline-flex w-6 flex-1 items-center justify-center border-b disabled:opacity-50"
            disabled={disabled || readOnly}
            onClick={() => applyNumericValue(currentValue + step)}
          >
            <ChevronUpIcon className="h-3 w-3" />
          </button>
          <button
            type="button"
            aria-label="Decrement"
            className="text-muted-foreground hover:bg-muted-foreground/10 hover:text-foreground inline-flex w-6 flex-1 items-center justify-center disabled:opacity-50"
            disabled={disabled || readOnly}
            onClick={() => applyNumericValue(currentValue - step)}
          >
            <ChevronDownIcon className="h-3 w-3" />
          </button>
        </div>
      </div>
    </div>
  );
}
