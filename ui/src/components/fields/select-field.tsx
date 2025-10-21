import { cn } from "@/lib/utils";
import { DoubleClickSelectFieldProps, SelectFieldProps } from "@/types/fields";
import { CheckIcon, ChevronDownIcon } from "@radix-ui/react-icons";
import { useMemo, useState } from "react";
import { Controller, FieldValues, useController } from "react-hook-form";
import { Button } from "../ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  SelectCommandItem,
} from "../ui/command";
import { Icon } from "../ui/icons";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { FieldWrapper } from "./field-components";

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
  const [isOpen, setIsOpen] = useState(false);
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

  const renderIcon = () => {
    if (
      typeof selectedOption?.icon === "object" &&
      selectedOption?.icon !== null &&
      "icon" in selectedOption.icon
    ) {
      return <Icon icon={selectedOption?.icon} />;
    }
    return selectedOption?.icon;
  };
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
            <PopoverTrigger className="w-full" asChild>
              <Button
                variant="outline"
                className={cn(
                  "group bg-primary/5 hover:bg-primary/10 flex h-7 w-full items-center justify-between whitespace-nowrap rounded-md border border-muted-foreground/20",
                  "px-1.5 py-2 text-xs ring-offset-background placeholder:text-muted-foreground outline-hidden",
                  "data-[state=open]:border-foreground data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-foreground/20",
                  "focus-visible:border-foreground focus-visible:outline-hidden focus-visible:ring-4 focus-visible:ring-foreground/20",
                  "transition-[border-color,box-shadow] duration-200 ease-in-out",
                  "disabled:opacity-50 [&>span]:line-clamp-1 cursor-pointer disabled:cursor-not-allowed",
                  fieldState.invalid &&
                    "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20",
                  isReadOnly &&
                    "cursor-not-allowed opacity-60 pointer-events-none",
                )}
              >
                <div
                  className={cn(
                    "flex flex-1 min-w-0 items-center gap-x-1.5 truncate font-normal text-foreground [&_svg]:size-3 [&_svg]:shrink-0",
                    !selectedOption?.value && "text-muted-foreground",
                  )}
                >
                  {color ? (
                    <span
                      className="block size-2 rounded-full"
                      style={{ backgroundColor: color }}
                    />
                  ) : (
                    renderIcon()
                  )}
                  <span className="truncate">
                    {selectedOption?.label || placeholder}
                  </span>
                </div>
                <ChevronDownIcon className="group-data-[state=open]:rotate-180 transition-transform duration-200 ease-in-out size-3 opacity-50 flex-shrink-0 ml-1" />
              </Button>
            </PopoverTrigger>
            <PopoverContent
              className="border-input max-w-[var(--radix-popover-trigger-width)] p-0"
              align="start"
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
                          // If not clearable and trying to deselect the current value, do nothing
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

export function DoubleClickSelectField<T extends FieldValues>({
  name,
  control,
  rules,
  placeholder,
  options,
  isClearable = false,
}: DoubleClickSelectFieldProps<T>) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        return (
          <Popover open={isOpen} onOpenChange={setIsOpen}>
            <PopoverTrigger>
              <span className="flex flex-col text-left text-xs">
                <div
                  className={cn(
                    "cursor-text",
                    fieldState.invalid && "text-red-500",
                  )}
                >
                  {field.value}
                </div>
                {isOpen ? (
                  <span
                    onClick={() => setIsOpen(false)}
                    className="cursor-pointer select-none text-xs text-muted-foreground"
                  >
                    Cancel
                  </span>
                ) : (
                  <span className="cursor-pointer select-none text-xs text-muted-foreground">
                    Click to edit
                  </span>
                )}
              </span>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0">
              <Command>
                <CommandInput className="h-7" placeholder={placeholder} />
                <CommandList>
                  <CommandEmpty>No results found</CommandEmpty>
                  <CommandGroup>
                    {options.map((option) => (
                      <CommandItem
                        key={option.value as string}
                        value={option.value as string}
                        onSelect={() => {
                          // If not clearable and trying to deselect the current value, do nothing
                          if (!isClearable && field.value === option.value) {
                            return;
                          }
                          field.onChange(option.value);
                          setIsOpen(false);
                        }}
                      >
                        {option.label}
                        <CheckIcon
                          className={cn(
                            "ml-auto",
                            field.value === option.value
                              ? "opacity-100"
                              : "opacity-0",
                          )}
                        />
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
        );
      }}
    />
  );
}
