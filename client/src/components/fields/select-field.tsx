import { cn } from "@/lib/utils";
import type { FormControlProps, SelectOption } from "@/types/fields";
import { ChevronDownIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Controller, useController, type FieldValues } from "react-hook-form";
import { Button } from "../ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandList,
  SelectCommandItem,
} from "../ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { FieldWrapper } from "./field-components";

export type BaseSelectFieldProps = {
  options: SelectOption[];
  label?: string;
  description?: string;
  isReadOnly?: boolean;
  isBoolean?: boolean;
  isInvalid?: boolean;
  className?: string;
  placeholder?: string;
  isClearable?: boolean;
};

type SelectFieldProps<T extends FieldValues> = BaseSelectFieldProps &
  FormControlProps<T>;

export function SelectField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  options,
  placeholder,
  isReadOnly,
  isClearable = false,
}: SelectFieldProps<T>) {
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const { field } = useController({ name, control });
  const [searchValue, setSearchValue] = useState("");

  const selectedOption = useMemo(
    () => options.find((option) => option.value === field.value) || null,
    [field.value, options],
  );

  const optionMap = useMemo(
    () => new Map(options.map((opt) => [opt.value.toLowerCase(), opt])),
    [options],
  );

  const color = selectedOption?.color;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
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
                    "group flex h-8 w-full items-center justify-between rounded-md border border-input bg-muted whitespace-nowrap hover:bg-muted/80",
                    "px-1.5 py-2 text-xs ring-offset-background outline-hidden placeholder:text-muted-foreground",
                    "data-pressed:border-brand data-pressed:ring-4 data-pressed:ring-brand/30",
                    "transition-[border-color,box-shadow] duration-200 ease-in-out",
                    "cursor-default disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1",
                    fieldState.invalid && "data-pressed:ring-destructive/20",
                    isReadOnly &&
                      "pointer-events-none cursor-not-allowed opacity-60",
                  )}
                >
                  <div
                    className={cn(
                      "flex min-w-0 flex-1 items-center gap-x-1.5 truncate font-normal text-foreground",
                      !selectedOption?.value && "text-muted-foreground",
                      fieldState.invalid && "text-destructive",
                    )}
                  >
                    {color ? (
                      <span
                        className="size-2 shrink-0 rounded-full"
                        style={{ backgroundColor: color }}
                      />
                    ) : null}
                    <span className="truncate">
                      {selectedOption?.label || placeholder}
                    </span>
                  </div>
                  <ChevronDownIcon className="ml-1 size-3 flex-shrink-0 opacity-50 transition-transform duration-200 ease-in-out group-data-pressed:rotate-180" />
                </Button>
              }
            />
            <PopoverContent
              className="border-input p-0"
              align="start"
              positionerClassName="min-w-(--anchor-width) rounded-lg"
            >
              <Command
                filter={(value, search) => {
                  const item = optionMap.get(value.toLowerCase());
                  if (!item) return 0;
                  if (!search) return 1;
                  return item.label.toLowerCase().includes(search.toLowerCase())
                    ? 1
                    : 0;
                }}
              >
                <CommandInput
                  placeholder={`Search ${label?.toLowerCase()}...`}
                  onValueChange={(value) => setSearchValue(value)}
                />
                <CommandList>
                  <CommandEmpty>No options found.</CommandEmpty>
                  <CommandGroup>
                    {options.map((option) => (
                      <SelectCommandItem
                        key={option.value}
                        value={option.value}
                        onSelect={(currentValue) => {
                          if (!isClearable && currentValue === field.value) {
                            return;
                          }
                          field.onChange(
                            currentValue === field.value ? "" : currentValue,
                          );
                          setIsOpen(false);
                        }}
                        color={option.color}
                        disabled={option.disabled}
                        checked={field.value === option.value}
                        icon={option.icon}
                        label={option.label}
                        description={option.description}
                        searchValue={searchValue}
                      />
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
        </FieldWrapper>
      )}
    />
  );
}
