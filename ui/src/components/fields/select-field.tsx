import { cn } from "@/lib/utils";
import {
  DoubleClickSelectFieldProps,
  SelectFieldProps,
  SelectOption,
} from "@/types/fields";
import { CheckIcon } from "@radix-ui/react-icons";
import React, { useMemo, useState } from "react";
import { Controller, FieldValues } from "react-hook-form";
import Select, { ActionMeta } from "react-select";
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
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "../ui/tooltip";
import { FieldWrapper } from "./field-components";
import {
  ClearIndicator,
  DropdownIndicator,
  Group,
  InputComponent,
  LoadingMessage,
  MenuList,
  NoOptionsMessage,
  Option,
  SingleValueComponent,
  ValueContainer,
} from "./select-components";

type ReactSelectInputProps = Omit<SelectFieldProps<any>, "label" | "control">;

type GroupedOption = {
  label: string;
  options: SelectOption[];
};

const ReactSelectInput = React.forwardRef<any, ReactSelectInputProps>(
  (
    {
      options,
      placeholder = "Select an option...",
      className,
      isReadOnly,
      isDisabled,
      isInvalid,
      isMulti,
      onChange,
      value,
      isLoading,
      isFetchError,
      ...props
    },
    ref,
  ) => {
    const isError = isFetchError || isInvalid;

    const selectAllOptions = useMemo(
      () => ({
        label: "Select All",
        value: "<SELECT_ALL>",
        color: "#15083d",
      }),
      [],
    );

    const getOptions = useMemo(() => {
      if (!isMulti) return options;

      if (options.length > 0 && "options" in options[0]) {
        return [selectAllOptions, ...options];
      } else {
        return [selectAllOptions, ...options];
      }
    }, [isMulti, options, selectAllOptions]);

    const flattenOptions: SelectOption[] = useMemo(() => {
      return options.flatMap((item) => {
        if ("options" in item && Array.isArray(item.options)) {
          return item.options;
        }

        return item as SelectOption;
      });
    }, [options]);

    const getValue = () => {
      if (!isMulti) {
        const selectedValue = value;

        // Check if the value if a boolean
        if (typeof selectedValue === "boolean") {
          return (
            flattenOptions.find((option) => option.value === selectedValue) ||
            null
          );
        }
        return selectedValue
          ? flattenOptions.find((option) => option.value === selectedValue)
          : null;
      }

      if (Array.isArray(value)) {
        const selectedValues = value as (string | boolean)[];
        return flattenOptions.filter((option) =>
          selectedValues.includes(option.value as string | boolean),
        );
      }

      return [];
    };

    const handleChange = (
      newValue: any,
      actionMeta: ActionMeta<SelectOption>,
    ) => {
      if (!isMulti) {
        return newValue ? newValue.value : null;
      }

      if (!Array.isArray(newValue)) {
        return [];
      }

      const { action, option } = actionMeta;

      if (
        action === "select-option" &&
        option?.value === selectAllOptions.value
      ) {
        return flattenOptions.map((option) => option.value);
      } else if (
        action === "deselect-option" &&
        option?.value === selectAllOptions.value
      ) {
        return [];
      } else {
        return newValue.flatMap((item: GroupedOption | SelectOption) => {
          if (item && "options" in item && Array.isArray(item.options)) {
            // This is a group
            return item.options.map((option) => option?.value).filter(Boolean);
          } else if (item && "value" in item) {
            // This is a single option
            return item.value;
          } else {
            return [];
          }
        });
      }
    };

    return (
      <Select
        unstyled
        ref={ref}
        options={getOptions}
        placeholder={placeholder}
        className={className}
        value={getValue()}
        isLoading={isLoading}
        isDisabled={isDisabled || isReadOnly}
        onChange={(newValue, actionMeta) => {
          const transformedValue = handleChange(
            newValue,
            actionMeta as ActionMeta<SelectOption>,
          );
          onChange(transformedValue);
        }}
        styles={{
          control: () => ({
            cursor: "pointer",
            // minHeight: "2rem",
          }),
          menuList: (base) => ({
            ...base,
            display: "flex",
            flexDirection: "column",
            padding: "0.25rem",
            gap: "0.2rem",
            // Change scrollbar
            "::-webkit-scrollbar": {
              width: "0.3rem",
              background: "transparent",
            },
            "::-webkit-scrollbar-track": {
              background: "transparent",
            },
            "::-webkit-scrollbar-thumb": {
              background: "hsl(var(--border))",
            },
            "::-webkit-scrollbar-thumb:hover": {
              background: "transparent",
            },
          }),
        }}
        classNames={{
          control: (state) =>
            cn(
              "flex items-center h-7 w-full rounded-md border border-muted-foreground/20 px-2 py-1.5 bg-muted text-sm",
              "transition-[border-color,box-shadow] outline-hidden  duration-200 ease-in-out",
              state.isFocused && "border-blue-600 ring-4 ring-blue-600/20",
              // Invalid and focused state
              state.isFocused &&
                isError &&
                "ring-red-500 border-red-600 ring-4 ring-red-400/20",
              // Invalid state
              isError && "border-red-500 bg-red-500/20",
            ),
          placeholder: () =>
            cn("text-muted-foreground", isError && "text-red-500"),
          container: () =>
            cn(
              isReadOnly && "cursor-not-allowed opacity-60 pointer-events-none",
            ),
          valueContainer: () => cn("gap-1", isReadOnly && "cursor-not-allowed"),
          singleValue: () => "leading-7 ml-1",
          multiValue: () =>
            "bg-muted rounded items-center py-0.5 pl-2 pr-1 gap-0.5 h-6",
          multiValueLabel: () => "text-xs leading-4",
          multiValueRemove: () =>
            "hover:text-foreground/50 text-foreground rounded-md h-4 w-4",
          indicatorsContainer: () => cn(isReadOnly && "cursor-not-allowed"),
          clearIndicator: () => "text-foreground/50 hover:text-foreground",
          dropdownIndicator: () =>
            "p-1 text-foreground/50 rounded-md hover:text-foreground",
          menu: () => "mt-2 border rounded-md bg-popover shadow-lg",
          groupHeading: () => "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
          noOptionsMessage: () => "text-muted-foreground",
        }}
        components={{
          ClearIndicator: ClearIndicator,
          ValueContainer: ValueContainer,
          DropdownIndicator: DropdownIndicator,
          MenuList: MenuList,
          Option: Option,
          Input: InputComponent,
          NoOptionsMessage: NoOptionsMessage,
          SingleValue: SingleValueComponent,
          Group: Group,
          LoadingMessage: LoadingMessage,
        }}
        {...props}
      />
    );
  },
);

ReactSelectInput.displayName = "ReactSelectInput";

export function SelectField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  options,
  isReadOnly,
  isMulti,
  isLoading,
  isClearable,
  isFetchError,
  placeholder,
  menuPlacement,
}: Omit<SelectFieldProps<T>, "onChange">) {
  const [isOpen, setIsOpen] = useState(false);

  const inputId = `select-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({
        field: { onChange, value, onBlur, ref, disabled },
        fieldState,
      }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
        >
          <ReactSelectInput
            isDisabled={disabled}
            menuPlacement={menuPlacement}
            id={inputId}
            ref={ref}
            name={name}
            isMulti={isMulti}
            isClearable={isClearable}
            onChange={onChange}
            placeholder={placeholder}
            onBlur={onBlur}
            // onFocus={() => setIsOpen(true)}
            menuIsOpen={isOpen}
            onMenuOpen={() => setIsOpen(true)}
            onMenuClose={() => setIsOpen(false)}
            isFetchError={isFetchError}
            isReadOnly={isReadOnly}
            value={value}
            options={options}
            aria-describedby={cn(description && descriptionId, errorId)}
            aria-invalid={fieldState.invalid}
            isInvalid={fieldState.invalid}
            isLoading={isLoading}
          />
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
              <TooltipProvider>
                <Tooltip delayDuration={0}>
                  <TooltipTrigger asChild>
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
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{placeholder}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0">
              <Command>
                <CommandInput className="h-8" placeholder={placeholder} />
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
