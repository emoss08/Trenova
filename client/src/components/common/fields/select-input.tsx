/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import { UseControllerProps, useController } from "react-hook-form";
import Select, { GroupBase, OptionsOrGroups, Props } from "react-select";
import CreatableSelect, { CreatableProps } from "react-select/creatable";
import { Label } from "./label";
import {
  ClearIndicator,
  DropdownIndicator,
  ErrorMessage,
  IndicatorSeparator,
  MenuList,
  Option,
  SelectDescription,
  SelectOption,
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
  label: string;
  description?: string;
  options: OptionsOrGroups<SelectOption, GroupBase<SelectOption>>;
  hasContextMenu?: boolean;
}

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
    isClearable,
    isMulti,
    placeholder,
    options,
    maxOptions,
    menuPlacement = "auto",
    menuPosition = "absolute",
    hideSelectedOptions = false,
    ...controllerProps
  } = props;

  const dataLoading = props.isLoading || props.isDisabled;
  const errorOccurred = props.isFetchError || !!fieldState.error?.message;
  const processedValue = ValueProcessor(field.value, options, isMulti);

  return (
    <>
      {label && (
        <Label
          className={cn(
            "text-sm font-medium",
            controllerProps.rules?.required && "required",
          )}
          htmlFor={controllerProps.id}
        >
          {label}
        </Label>
      )}
      <div className="relative">
        <Select
          aria-invalid={fieldState.invalid || isFetchError}
          closeMenuOnSelect={!isMulti}
          hideSelectedOptions={hideSelectedOptions}
          unstyled
          options={options}
          isMulti={isMulti}
          isLoading={isLoading}
          isDisabled={dataLoading || isFetchError}
          isClearable={isClearable}
          maxOptions={maxOptions}
          placeholder={placeholder}
          isFetchError={isFetchError}
          formError={fieldState.error?.message}
          maxMenuHeight={200}
          menuPlacement={menuPlacement}
          menuPosition={menuPosition}
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
            }),
          }}
          components={{
            ClearIndicator: ClearIndicator,
            ValueContainer: ValueContainer,
            DropdownIndicator: DropdownIndicator,
            IndicatorSeparator: IndicatorSeparator,
            MenuList: MenuList,
            Option: Option,
          }}
          classNames={{
            control: ({ isFocused }) =>
              cn(
                isFocused
                  ? "flex h-10 w-full rounded-md border border-input bg-background text-sm sm:text-sm sm:leading-6 ring-1 ring-inset ring-foreground"
                  : "flex h-10 w-full rounded-md border border-input bg-background text-sm sm:text-sm sm:leading-6 disabled:cursor-not-allowed disabled:opacity-50",
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
            menu: () => "mt-2 p-1 border rounded-md bg-background shadow-lg",
            groupHeading: () => "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
            noOptionsMessage: () =>
              "text-muted-foreground p-2 bg-background rounded-sm",
          }}
          {...field}
          value={processedValue}
          onChange={(selected) => {
            if (isMulti) {
              const values = (selected as SelectOption[]).map(
                (opt) => opt.value,
              );
              field.onChange(values);
            } else {
              field.onChange(
                selected ? (selected as SelectOption).value : undefined,
              );
            }
          }}
        />
        {errorOccurred ? (
          <ErrorMessage
            isFetchError={isFetchError}
            formError={fieldState.error?.message}
          />
        ) : (
          <SelectDescription description={description!} />
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
          components={{
            ClearIndicator: ClearIndicator,
            ValueContainer: ValueContainer,
            DropdownIndicator: DropdownIndicator,
            IndicatorSeparator: IndicatorSeparator,
            MenuList: MenuList,
            Option: Option,
          }}
          classNames={{
            control: ({ isFocused }) =>
              cn(
                isFocused
                  ? "flex h-10 w-full rounded-md border border-input bg-background text-sm sm:text-sm sm:leading-6 ring-1 ring-inset ring-foreground"
                  : "flex h-10 w-full rounded-md border border-input bg-background text-sm sm:text-sm sm:leading-6 disabled:cursor-not-allowed disabled:opacity-50",
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
            menu: () => "mt-2 p-1 border rounded-md bg-background shadow-lg",
            groupHeading: () => "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
            noOptionsMessage: () =>
              "text-muted-foreground p-2 bg-background rounded-sm",
          }}
        />
        {errorOccurred ? (
          <ErrorMessage
            isFetchError={isFetchError}
            formError={fieldState.error?.message}
          />
        ) : (
          <SelectDescription description={description!} />
        )}
      </div>
    </>
  );
}
