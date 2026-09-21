import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormControlProps,
  SelectOption,
  SelectOptionGroup,
  WarningProps,
} from "@trenova/shared/types/fields";
import { CheckIcon, ChevronDownIcon, XIcon } from "lucide-react";
import React, { useMemo, useState } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  SelectCommandItem,
} from "@trenova/shared/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { FieldWrapper } from "./field-components";
import { fieldInvalidClass, fieldTriggerClass } from "@trenova/shared/lib/variants/field";

export type BaseSelectFieldProps = {
  options?: SelectOption[];
  groups?: SelectOptionGroup[];
  label?: string;
  description?: string;
  isReadOnly?: boolean;
  isBoolean?: boolean;
  isInvalid?: boolean;
  className?: string;
  placeholder?: string;
  isClearable?: boolean;
  renderOption?: (option: SelectOption, searchValue: string) => React.ReactNode;
  onValueChange?: (value: string) => void;
  warning?: WarningProps;
};

type SelectFieldProps<T extends FieldValues> = BaseSelectFieldProps & FormControlProps<T>;

export function SelectField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  options,
  groups,
  warning,
  placeholder,
  isReadOnly,
  isClearable = false,
  renderOption,
  onValueChange,
}: SelectFieldProps<T>) {
  const t = useT();

  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [searchValue, setSearchValue] = useState("");

  const allOptions = useMemo(
    () => (groups ? groups.flatMap((g) => g.options) : (options ?? [])),
    [options, groups],
  );

  const optionMap = useMemo(
    () => new Map(allOptions.map((opt) => [String(opt.value).toLowerCase(), opt])),
    [allOptions],
  );

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        const selectedOption = allOptions.find((option) => option.value === field.value) || null;
        const color = selectedOption?.color;

        const handleClear = () => {
          field.onChange("");
          onValueChange?.("");
          setSearchValue("");
        };

        const handleSelect = (currentValue: string) => {
          if (!isClearable && currentValue === field.value) {
            return;
          }
          const nextValue = currentValue === field.value ? "" : currentValue;
          field.onChange(nextValue);
          onValueChange?.(nextValue);
          setIsOpen(false);
        };

        const renderOptions = (items: SelectOption[]) =>
          items.map((option) =>
            renderOption ? (
              <CommandItem
                key={String(option.value)}
                value={String(option.value)}
                onSelect={handleSelect}
                disabled={option.disabled}
              >
                {field.value === option.value && <CheckIcon className="size-3.5 shrink-0" />}
                {renderOption(option, searchValue)}
              </CommandItem>
            ) : (
              <SelectCommandItem
                key={String(option.value)}
                value={String(option.value)}
                onSelect={handleSelect}
                color={option.color}
                disabled={option.disabled}
                checked={field.value === option.value}
                icon={option.icon}
                label={t(option.label)}
                description={t(option.description)}
                searchValue={searchValue}
              />
            ),
          );

        return (
          <FieldWrapper
            label={label}
            description={description}
            warning={warning}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <Popover open={isOpen} onOpenChange={setIsOpen}>
              <PopoverTrigger
                className="w-full"
                render={
                  <Button
                    variant="outline"
                    aria-invalid={fieldState.invalid}
                    className={cn(
                      fieldTriggerClass,
                      "group flex h-7 w-full items-center justify-between whitespace-nowrap",
                      "ring-offset-background placeholder:text-muted-foreground px-1.5 py-1 text-xs outline-hidden select-none",
                      "cursor-default disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1",
                      fieldState.invalid && fieldInvalidClass,
                      isReadOnly && "pointer-events-none cursor-not-allowed opacity-60",
                      warning?.show &&
                        "ui-focus-ring border-warning bg-warning-subtle ring-0 ring-warning placeholder:text-warning-foreground focus:outline-hidden",
                    )}
                  >
                    <div
                      className={cn(
                        "text-foreground flex min-w-0 flex-1 items-center gap-x-1.5 truncate font-normal",
                        !selectedOption?.value && "text-muted-foreground",
                        fieldState.invalid && "text-destructive",
                      )}
                    >
                      {/* An option that carries a mark shows it here too. Seeing the
                          vendor's logo on the closed field is the whole reason the
                          list had one; a dot is the fallback for a plain choice. */}
                      {selectedOption?.icon ? (
                        <span className="flex size-4 shrink-0 items-center justify-center [&_svg]:size-full">
                          {selectedOption.icon}
                        </span>
                      ) : color ? (
                        <span
                          className="size-2 shrink-0 rounded-full"
                          style={{ backgroundColor: color }}
                        />
                      ) : null}
                      <span className="truncate">
                        {selectedOption ? t(selectedOption.label) : placeholder}
                      </span>
                    </div>
                    <SelectInputActions
                      clearable={isClearable}
                      currentValue={field.value}
                      handleClear={handleClear}
                      open={isOpen}
                    />
                  </Button>
                }
              />
              <PopoverContent
                className="border-input p-0"
                align="start"
                positionerClassName="min-w-(--anchor-width) rounded-lg dark"
              >
                <Command
                  filter={(value, search) => {
                    const item = optionMap.get(value.toLowerCase());
                    if (!item) return 0;
                    if (!search) return 1;
                    return item.label.toLowerCase().includes(search.toLowerCase()) ? 1 : 0;
                  }}
                >
                  <CommandInput
                    placeholder={`Search ${label?.toLowerCase()}...`}
                    onValueChange={(value) => setSearchValue(value)}
                  />
                  <CommandList>
                    <CommandEmpty>{t("No options found.")}</CommandEmpty>
                    {groups ? (
                      groups.map((group, idx) => (
                        <React.Fragment key={group.label}>
                          {idx > 0 && <CommandSeparator />}
                          <CommandGroup heading={t(group.label)}>
                            {renderOptions(group.options)}
                          </CommandGroup>
                        </React.Fragment>
                      ))
                    ) : (
                      <CommandGroup>{renderOptions(allOptions)}</CommandGroup>
                    )}
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          </FieldWrapper>
        );
      }}
    />
  );
}

function SelectInputActions({
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
  const t = useT();

  return (
    <div className="ml-auto flex items-center gap-1">
      {clearable && currentValue && (
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
          "size-3 opacity-50 transition-all duration-200 ease-in-out",
          open && "-rotate-180",
        )}
      />
    </div>
  );
}
