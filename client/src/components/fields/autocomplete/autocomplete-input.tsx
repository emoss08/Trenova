import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ChevronDownIcon, XIcon } from "lucide-react";
import React from "react";

type AutocompleteTriggerProps<TOption> = {
  open: boolean;
  disabled: boolean;
  triggerClassName: string | undefined;
  clearable: boolean;
  currentValue: string | null | undefined;
  selectedOption: TOption | null;
  getDisplayValue: (option: TOption) => React.ReactNode;
  placeholder: string;
  handleClear: () => void;
  listboxId: string;
  isInvalid?: boolean;
} & React.ComponentProps<"button">;

export function AutocompleteTrigger<TOption>({
  open,
  disabled,
  isInvalid,
  triggerClassName,
  clearable,
  currentValue,
  selectedOption,
  getDisplayValue,
  placeholder,
  handleClear,
  listboxId,
  ...props
}: AutocompleteTriggerProps<TOption>) {
  return (
    <Button
      type="button"
      variant="outline"
      role="combobox"
      aria-expanded={open}
      aria-controls={listboxId}
      className={cn(
        "h-7 w-full gap-2 rounded border-muted-foreground/20 bg-muted px-1.5 text-xs font-normal",
        "data-pressed:border-brand data-pressed:ring-4 data-pressed:ring-brand/30 data-pressed:outline-hidden",
        "cursor-default justify-between hover:bg-muted-foreground/10 dark:hover:bg-muted-foreground/30 [&_svg]:size-3",
        "transition-[border-color,box-shadow] duration-200 ease-in-out",
        disabled && "cursor-not-allowed opacity-50",
        isInvalid &&
          "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 hover:border-red-500 hover:bg-red-500/20 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20 data-pressed:border-red-500 data-pressed:bg-red-500/20 data-pressed:ring-red-500/20",
        triggerClassName,
      )}
      disabled={disabled}
      {...props}
    >
      <AutocompleteInputInner
        selectedOption={selectedOption}
        getDisplayValue={getDisplayValue}
        isInvalid={isInvalid}
        placeholder={placeholder}
      />
      <AutocompleteInputActions
        clearable={clearable}
        currentValue={currentValue}
        handleClear={handleClear}
        open={open}
      />
    </Button>
  );
}

export function AutocompleteInputInner<TOption>({
  selectedOption,
  getDisplayValue,
  placeholder,
  isInvalid,
}: {
  selectedOption: TOption | null;
  getDisplayValue: (option: TOption) => React.ReactNode;
  placeholder: string;
  isInvalid?: boolean;
}) {
  const displayValue = selectedOption ? (
    getDisplayValue(selectedOption)
  ) : (
    <p className={cn("text-muted-foreground", isInvalid && "text-red-500")}>{placeholder}</p>
  );

  return <div className="truncate">{displayValue}</div>;
}

export function AutocompleteInputActions({
  clearable,
  currentValue,
  handleClear,
  open,
}: {
  clearable: boolean;
  currentValue: string | null | undefined;
  handleClear: () => void;
  open: boolean;
}) {
  return (
    <div className="ml-auto flex items-center gap-1">
      {clearable && currentValue && (
        <span
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            handleClear();
          }}
          className="flex size-5 cursor-pointer items-center justify-center rounded-md text-muted-foreground transition-colors duration-200 ease-in-out hover:bg-muted-foreground/30 hover:text-foreground [&>svg]:size-3"
        >
          <span className="sr-only">Clear</span>
          <XIcon className="size-4" />
        </span>
      )}
      <ChevronDownIcon
        className={cn(
          "size-7 opacity-50 transition-all duration-200 ease-in-out",
          open && "-rotate-180",
        )}
      />
    </div>
  );
}
