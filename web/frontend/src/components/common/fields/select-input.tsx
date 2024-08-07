/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { cn } from "@/lib/utils";
import { Controller, UseControllerProps, useController } from "react-hook-form";
import Select, { GroupBase, OptionsOrGroups, Props } from "react-select";

import { useMemo } from "react";
import CreatableSelect, { CreatableProps } from "react-select/creatable";
import { FieldDescription } from "./components";
import { Label } from "./label";
import {
  ClearIndicator,
  DropdownIndicator,
  ErrorMessage,
  Group,
  IndicatorSeparator,
  InputComponent,
  MenuList,
  NoOptionsMessage,
  Option,
  SelectOption,
  SingleValueComponent,
  ValueContainer,
  ValueProcessor,
} from "./select-components";

/**
 * Props for the SelectInput component.
 * @param T The type of the form object.
 * @param K The type of the created option.
 */
interface SelectInputProps<T extends Record<string, unknown>>
  extends UseControllerProps<T>,
    Omit<
      Props<SelectOption, boolean, GroupBase<SelectOption>>,
      "defaultValue" | "name"
    > {
  label?: string;
  description?: string;
  options: OptionsOrGroups<SelectOption, GroupBase<SelectOption>>;
  hasContextMenu?: boolean;
  maxOptions?: number;
  isFetchError?: boolean;
  hasPopoutWindow?: boolean; // Set to true to open the popout window
  popoutLink?: string; // Link to the popout page
  popoutLinkLabel?: string; // Label for the popout link
  isReadOnly?: boolean;
}

type GroupedOption = {
  label: string;
  options: SelectOption[];
};

/**
 * A wrapper around react-select's Select component.
 * @param props {SelectInputProps}
 * @constructor SelectInput
 */
export function SelectInput<T extends Record<string, unknown>>(
  props: SelectInputProps<T>,
) {
  const { field, fieldState } = useController(props);

  const {
    label,
    description,
    isFetchError,
    isLoading,
    rules,
    isClearable,
    isMulti,
    placeholder,
    options,
    maxOptions = 10,
    menuPlacement = "auto",
    menuPosition = "absolute",
    hideSelectedOptions = false,
    hasPopoutWindow = false,
    popoutLink,
    popoutLinkLabel,
    isReadOnly,
    name,
    menuIsOpen,
    isDisabled,
    control,
    ...controllerProps
  } = props;

  const dataLoading = isLoading || isDisabled;
  const errorOccurred = isFetchError || fieldState.invalid;
  const menuOpen = isReadOnly ? false : menuIsOpen;

  const selectAllOption: SelectOption = useMemo(
    () => ({
      label: "Select All",
      value: "<SELECT_ALL>" as string,
      color: "#15803d",
    }),
    [],
  );

  const getOptionsWithSelectAll = useMemo(() => {
    if (!isMulti) return options;

    if (options.length > 0 && "options" in options[0]) {
      // Grouped options
      return [selectAllOption, ...options];
    } else {
      // Flat options
      return [selectAllOption, ...(options as SelectOption[])];
    }
  }, [isMulti, options, selectAllOption]);

  const flattenOptions = useMemo(() => {
    return options.flatMap((item) => {
      if ("options" in item && Array.isArray(item.options)) {
        return item.options;
      }
      return item as SelectOption;
    });
  }, [options]);

  const getValue = () => {
    if (!isMulti) {
      const selectedValue = field.value as string | undefined;
      return selectedValue
        ? flattenOptions.find((opt) => opt.value === selectedValue) || null
        : null;
    }

    if (Array.isArray(field.value)) {
      const selectedValues = field.value as string[];
      return flattenOptions.filter((opt) =>
        selectedValues.includes(opt.value as string),
      );
    }

    return [];
  };

  const handleChange = (newValue: any, actionMeta: any) => {
    if (!isMulti) {
      return newValue ? newValue.value : null;
    }

    if (!Array.isArray(newValue)) {
      return [];
    }

    const { action, option } = actionMeta;

    if (action === "select-option" && option?.value === selectAllOption.value) {
      return flattenOptions.map((opt) => opt.value);
    } else if (
      action === "deselect-option" &&
      option?.value === selectAllOption.value
    ) {
      return [];
    } else {
      return newValue.flatMap((item: GroupedOption | SelectOption) => {
        if (item && "options" in item && Array.isArray(item.options)) {
          // This is a group
          return item.options.map((opt) => opt?.value).filter(Boolean);
        } else if (item && "value" in item) {
          // This is an individual option
          return item.value;
        } else {
          return [];
        }
      });
    }
  };

  return (
    <>
      <span className="space-x-1">
        {label && <Label className="text-sm font-medium">{label}</Label>}
        {rules?.required && <span className="text-red-500">*</span>}
      </span>
      <div className="relative">
        <Controller
          name={name}
          control={control}
          render={({ field: { onChange, ...restField } }) => (
            <Select
              unstyled
              aria-invalid={errorOccurred}
              aria-labelledby={controllerProps.id}
              inputId={controllerProps.id}
              closeMenuOnSelect={!isMulti}
              hideSelectedOptions={hideSelectedOptions}
              popoutLinkLabel={popoutLinkLabel}
              options={getOptionsWithSelectAll}
              isMulti={isMulti}
              isLoading={isLoading}
              hasPopoutWindow={hasPopoutWindow}
              popoutLink={popoutLink}
              isDisabled={dataLoading || isFetchError}
              isClearable={isClearable}
              maxOptions={maxOptions}
              placeholder={placeholder}
              isFetchError={isFetchError}
              formError={fieldState.error?.message}
              noOptionsMessage={() => "No options available..."}
              maxMenuHeight={200}
              menuPlacement={menuPlacement}
              menuPosition={menuPosition}
              menuIsOpen={menuOpen}
              styles={{
                input: (base) => ({
                  ...base,
                  "input:focus": {
                    boxShadow: "none",
                  },
                }),
                control: (base) => ({
                  ...base,
                  transition: "none",
                  minHeight: "2.25rem",
                }),
              }}
              components={{
                ClearIndicator: ClearIndicator,
                ValueContainer: ValueContainer,
                DropdownIndicator: DropdownIndicator,
                IndicatorSeparator: IndicatorSeparator,
                MenuList: MenuList,
                Option: Option,
                Input: InputComponent,
                NoOptionsMessage: NoOptionsMessage,
                SingleValue: SingleValueComponent,
                Group: Group,
              }}
              classNames={{
                control: ({ isFocused }) =>
                  cn(
                    isFocused
                      ? "flex h-9 w-full rounded-md border border-border bg-background text-sm sm:text-sm sm:leading-6 ring-1 ring-inset ring-foreground"
                      : "flex h-9 w-full rounded-md border border-border bg-background text-sm sm:text-sm sm:leading-6",
                    errorOccurred &&
                      "ring-1 ring-inset ring-red-500 bg-red-500 bg-opacity-20",
                  ),
                placeholder: () =>
                  cn(
                    "text-muted-foreground pl-1 py-0.5 truncate",
                    errorOccurred && "text-red-500",
                  ),
                input: () => "pl-1 py-0.5",
                container: () =>
                  cn(isReadOnly && "cursor-not-allowed opacity-50"),
                valueContainer: () =>
                  cn("p-1 gap-1", isReadOnly && "cursor-not-allowed"),
                singleValue: () => "leading-7 ml-1",
                multiValue: () =>
                  "bg-accent rounded items-center py-0.5 pl-2 pr-1 gap-0.5 h-6",
                multiValueLabel: () => "text-xs leading-4",
                multiValueRemove: () =>
                  "hover:text-foreground/50 text-foreground rounded-md h-4 w-4",
                indicatorsContainer: () =>
                  cn("p-1 gap-1", isReadOnly && "cursor-not-allowed"),
                clearIndicator: () =>
                  "text-foreground/50 p-1 hover:text-foreground",
                dropdownIndicator: () =>
                  "p-1 text-foreground/50 rounded-md hover:text-foreground",
                menu: () => "mt-2 p-1 border rounded-md bg-popover shadow-lg",
                groupHeading: () =>
                  "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
                noOptionsMessage: () => "text-muted-foreground",
              }}
              {...restField}
              value={getValue()}
              onChange={(newValue, actionMeta) => {
                const transformedValue = handleChange(newValue, actionMeta);
                onChange(transformedValue);
              }}
            />
          )}
        />
        {errorOccurred ? (
          <ErrorMessage
            isFetchError={isFetchError}
            formError={fieldState.error?.message}
          />
        ) : (
          <FieldDescription description={description!} />
        )}
      </div>
    </>
  );
}

/**
 * Props for the CreatableSelectField component.
 * @param T The type of the form object.
 * @param K The type of the created option.
 */
interface CreatableSelectFieldProps<T extends Record<string, unknown>, K>
  extends UseControllerProps<T>,
    Omit<
      CreatableProps<SelectOption, boolean, GroupBase<SelectOption>>,
      "defaultValue" | "name"
    > {
  label: string;
  withAsterisk?: boolean;
  description?: string;
  isFetchError?: boolean;
  isLoading?: boolean;
  isDisabled?: boolean;
  isClearable?: boolean;
  isMulti?: boolean;
  placeholder?: string;
  formError?: string;
  options: OptionsOrGroups<SelectOption, GroupBase<SelectOption>>;
  onCreate: (inputValue: string) => Promise<K>;
}

/**
 * A wrapper around react-select's Creatable component.
 * @param props {CreatableSelectFieldProps}
 * @constructor CreatableSelectField
 */
export function CreatableSelectField<T extends Record<string, unknown>, K>(
  props: CreatableSelectFieldProps<T, K>,
) {
  const {
    label,
    withAsterisk,
    description,
    isFetchError,
    isLoading,
    isDisabled,
    isClearable,
    isMulti,
    placeholder,
    formError,
    options,
    onCreate,
    ...controllerProps
  } = props;

  const { field, fieldState } = useController(controllerProps);
  const errorOccurred = isFetchError || !!formError;
  const dataLoading = isLoading || isDisabled;
  const processedValue = ValueProcessor(field.value, options, isMulti);

  return (
    <>
      {label && (
        <Label
          className={cn("text-sm font-medium", withAsterisk && "required")}
          htmlFor={controllerProps.id}
        >
          {label}
        </Label>
      )}
      <div className="relative">
        <Controller
          name={controllerProps.name}
          control={controllerProps.control}
          render={({ field }) => (
            <CreatableSelect
              unstyled
              aria-invalid={fieldState.invalid || isFetchError}
              isMulti={isMulti}
              isLoading={isLoading}
              isDisabled={dataLoading || isFetchError}
              isClearable={isClearable}
              placeholder={placeholder || "Select"}
              closeMenuOnSelect={!isMulti}
              options={options}
              value={processedValue}
              onCreateOption={async (inputValue) => {
                const newOption = await onCreate(inputValue);
                const currentValues = Array.isArray(processedValue)
                  ? processedValue
                  : [];
                const updatedValues = [...currentValues, newOption];
                field.onChange(
                  (updatedValues as Array<{ value: string }>).map(
                    (opt) => opt.value,
                  ),
                );
              }}
              onChange={(selected) => {
                if (isMulti) {
                  const values = (selected as SelectOption[]).map(
                    (opt) => opt.value,
                  );
                  field.onChange(values);
                } else {
                  field.onChange((selected as SelectOption).value);
                }
              }}
              styles={{
                input: (base) => ({
                  ...base,
                  "input:focus": {
                    boxShadow: "none",
                  },
                }),
                control: (base) => ({
                  ...base,
                  transition: "none",
                  minHeight: "2.25rem",
                }),
              }}
              components={{
                ClearIndicator: ClearIndicator,
                ValueContainer: ValueContainer,
                DropdownIndicator: DropdownIndicator,
                IndicatorSeparator: IndicatorSeparator,
                MenuList: MenuList,
                Option: Option,
                NoOptionsMessage: NoOptionsMessage,
              }}
              classNames={{
                control: ({ isFocused }) =>
                  cn(
                    isFocused
                      ? "flex h-9 w-full rounded-md border border-border bg-background text-sm sm:text-sm sm:leading-6 ring-1 ring-inset ring-foreground"
                      : "flex h-9 w-full rounded-md border border-border bg-background text-sm sm:text-sm sm:leading-6 disabled:cursor-not-allowed disabled:opacity-50",
                    errorOccurred && "ring-1 ring-inset ring-red-500",
                  ),
                placeholder: () =>
                  cn(
                    "text-muted-foreground pl-1 py-0.5 truncate",
                    errorOccurred && "text-red-500",
                  ),
                input: () => "pl-1 py-0.5",
                valueContainer: () => "p-1 gap-1",
                singleValue: () => "leading-7 ml-1",
                multiValue: () =>
                  "bg-accent rounded items-center py-0.5 pl-2 pr-1 gap-0.5 h-6",
                multiValueLabel: () => "text-xs leading-4",
                multiValueRemove: () =>
                  "hover:text-foreground/50 text-foreground rounded-md h-4 w-4",
                indicatorsContainer: () => "p-1 gap-1",
                clearIndicator: () =>
                  "text-foreground/50 p-1 hover:text-foreground",
                dropdownIndicator: () =>
                  "p-1 text-foreground/50 rounded-md hover:text-foreground",
                menu: () => "mt-2 p-1 border rounded-md bg-popover shadow-lg",
                groupHeading: () =>
                  "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
                noOptionsMessage: () =>
                  "text-muted-foreground p-2 bg-popover rounded-sm",
              }}
            />
          )}
        />

        {errorOccurred ? (
          <ErrorMessage
            isFetchError={isFetchError}
            formError={fieldState.error?.message}
          />
        ) : (
          <FieldDescription description={description!} />
        )}
      </div>
    </>
  );
}
