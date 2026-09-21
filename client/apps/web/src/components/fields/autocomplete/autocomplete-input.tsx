import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronDownIcon, XIcon } from "lucide-react";
import React from "react";
import { fieldInvalidClass, fieldTriggerClass } from "@trenova/shared/lib/variants/field";

type AutocompleteTriggerProps<TOption> = {
  open: boolean;
  disabled: boolean;
  triggerClassName: string | undefined;
  clearable: boolean;
  currentValue: string | null | undefined;
  selectedOption: TOption | null;
  isLoadingSelected?: boolean;
  isErrorSelected?: boolean;
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
  isLoadingSelected,
  isErrorSelected,
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
        fieldTriggerClass,
        "h-7 w-full gap-2 px-1.5 text-xs font-normal",
        "cursor-default justify-between [&_svg]:size-3",
        disabled && "cursor-not-allowed opacity-50",
        isInvalid && fieldInvalidClass,
        triggerClassName,
      )}
      disabled={disabled}
      {...props}
    >
      <AutocompleteInputInner
        selectedOption={selectedOption}
        currentValue={currentValue}
        isLoadingSelected={isLoadingSelected}
        isErrorSelected={isErrorSelected}
        getDisplayValue={getDisplayValue}
        isInvalid={isInvalid}
        placeholder={placeholder}
      />
      <AutocompleteInputActions
        clearable={clearable}
        currentValue={currentValue}
        handleClear={handleClear}
        disabled={disabled}
        open={open}
      />
    </Button>
  );
}

export function AutocompleteInputInner<TOption>({
  selectedOption,
  currentValue,
  isLoadingSelected,
  isErrorSelected,
  getDisplayValue,
  placeholder,
  isInvalid,
}: {
  selectedOption: TOption | null;
  currentValue: string | null | undefined;
  isLoadingSelected?: boolean;
  isErrorSelected?: boolean;
  getDisplayValue: (option: TOption) => React.ReactNode;
  placeholder: string;
  isInvalid?: boolean;
}) {
  const t = useT();

  if (selectedOption) {
    return <div className="truncate">{getDisplayValue(selectedOption)}</div>;
  }

  if (currentValue && isLoadingSelected) {
    return (
      <div className="truncate">
        <span className="text-muted-foreground">{t("Loading...")}</span>
      </div>
    );
  }

  if (currentValue && isErrorSelected) {
    return (
      <div className="truncate">
        <span className="text-muted-foreground">{currentValue}</span>
      </div>
    );
  }

  return (
    <div className="truncate">
      <p className={cn("text-muted-foreground", isInvalid && "text-danger-foreground")}>
        {placeholder}
      </p>
    </div>
  );
}

export function AutocompleteInputActions({
  clearable,
  currentValue,
  handleClear,
  disabled,
  open,
}: {
  clearable: boolean;
  currentValue: string | null | undefined;
  handleClear: () => void;
  disabled?: boolean;
  open: boolean;
}) {
  const t = useT();

  return (
    <div className="ml-auto flex items-center gap-1">
      {clearable && currentValue && !disabled && (
        <span
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            handleClear();
          }}
          className="text-muted-foreground hover:bg-muted-foreground/30 hover:text-foreground flex size-5 cursor-pointer items-center justify-center rounded-md transition-colors duration-200 ease-in-out [&>svg]:size-3"
        >
          <span className="sr-only">{t("Clear")}</span>
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
