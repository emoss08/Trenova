import { cn } from "@/lib/utils";
import {
  DoubleClickSelectFieldProps,
  SelectFieldProps,
  type SelectOption,
} from "@/types/fields";
import { CheckIcon } from "@radix-ui/react-icons";
import { useState } from "react";
import { Controller, FieldValues, useController } from "react-hook-form";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";
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
}: SelectFieldProps<T>) {
  const [isOpen, setIsOpen] = useState(false);
  const { field } = useController({ name, control });

  const [selectedOption, setSelectedOption] = useState<SelectOption | null>(
    options.find((option) => option.value === field.value) || null,
  );

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
          <Select
            open={isOpen}
            onOpenChange={setIsOpen}
            onValueChange={(value) => {
              field.onChange(value);
              // * Update the selected option
              setSelectedOption(
                options.find((option) => option.value === value) || null,
              );
            }}
            defaultValue={field.value}
          >
            <SelectTrigger>
              <SelectValue
                placeholder={placeholder}
                color={selectedOption?.color}
              />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {options.map((option) => (
                  <SelectItem
                    key={String(option.value)}
                    value={String(option.value)}
                    icon={option.icon}
                    color={option.color}
                  >
                    {option.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
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
