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

import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { CaretSortIcon, CheckIcon, Cross2Icon } from "@radix-ui/react-icons";
import { AlertTriangle } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import {
  Path,
  PathValue,
  useController,
  UseControllerProps,
} from "react-hook-form";
import Select, {
  ClearIndicatorProps,
  components,
  DropdownIndicatorProps,
  GroupBase,
  IndicatorSeparatorProps,
  MenuListProps,
  OptionProps,
  OptionsOrGroups,
  Props,
  ValueContainerProps,
} from "react-select";
import { Label } from "./label";
import CreatableSelect, { CreatableProps } from "react-select/creatable";

/**
 * Option type for the SelectInput component.
 */
type SelectOption = {
  readonly label: string;
  readonly value: string | boolean;
};

/**
 * Option component for the SelectInput component.
 * @param props {OptionProps}
 * @constructor Option
 */
function Option({ ...props }: OptionProps) {
  return (
    <components.Option
      className="relative flex cursor-default select-none rounded-sm px-3 py-1.5 text-xs outline-none my-1 hover:bg-accent hover:cursor-pointer hover:rounded-sm"
      {...props}
    >
      {props.label}
      {props.isSelected && (
        <CheckIcon className="absolute top-1/2 right-3 transform -translate-y-1/2 h-4 w-4" />
      )}
    </components.Option>
  );
}

/**
 * DropdownIndicator component for the SelectInput component.
 * @param props {DropdownIndicatorProps}
 * @constructor DropdownIndicator
 */
function DropdownIndicator(props: DropdownIndicatorProps) {
  return (
    <components.DropdownIndicator {...props}>
      {props.selectProps["aria-invalid"] ? (
        <AlertTriangle size={15} className="text-red-500" />
      ) : (
        <CaretSortIcon className="h-4 w-4 shrink-0" />
      )}
    </components.DropdownIndicator>
  );
}

/**
 * IndicatorSeparator component for the SelectInput component.
 * @param props {IndicatorSeparatorProps}
 * @constructor IndicatorSeparator
 */
function IndicatorSeparator(props: IndicatorSeparatorProps) {
  return (
    <span
      className={cn(
        "bg-foreground/30 h-6 w-px",
        props.selectProps["aria-invalid"] && "bg-red-500",
      )}
    />
  );
}

/**
 * ClearIndicator component for the SelectInput component.
 * @param props {ClearIndicatorProps}
 * @constructor ClearIndicator
 */
function ClearIndicator(props: ClearIndicatorProps) {
  return (
    <components.ClearIndicator {...props}>
      <Cross2Icon className="h-4 w-4 shrink-0" />
    </components.ClearIndicator>
  );
}

/**
 * ValueContainer component for the SelectInput component.
 * @param children {React.ReactNode}
 * @param rest {ValueContainerProps}
 * @constructor ValueContainer
 */
function ValueContainer({ children, ...rest }: ValueContainerProps) {
  const selectedCount = rest.getValue().length;
  const conditional = selectedCount < 3;
  const { ValueContainer } = components;

  let firstChild: React.ReactNode[] | null = [];

  if (!conditional && Array.isArray(children)) {
    firstChild = [children[0].shift(), children[1]];
  }

  return (
    <ValueContainer {...rest}>
      {conditional ? children : firstChild}
      {!conditional && ` and ${selectedCount - 1} others`}
    </ValueContainer>
  );
}

/**
 * Description component for the SelectInput component.
 * @param description {string}
 * @constructor SelectDescription
 */
function SelectDescription({ description }: { description: string }) {
  return <p className="text-xs text-foreground/70">{description}</p>;
}

/**
 * MenuList component for the SelectInput component.
 * @param children {React.ReactNode}
 * @param props {MenuListProps}
 * @constructor MenuList
 */
function MenuList({
  children,
  ...props
}: MenuListProps & {
  selectProps?: {
    maxOptions?: number;
    formError?: string;
  };
}) {
  return (
    <components.MenuList {...props}>
      {Array.isArray(children)
        ? children.slice(0, props.selectProps?.maxOptions)
        : children}
    </components.MenuList>
  );
}

/**
 * Gets the label of the option by its value.
 * @param value {PathValue<T, Path<T>>}
 * @param options {OptionsOrGroups<SelectOption, GroupBase<SelectOption>>}
 */
function getLabelByValue<T extends Record<string, unknown>>(
  value: PathValue<T, Path<T>>,
  options: OptionsOrGroups<SelectOption, GroupBase<SelectOption>>,
) {
  const option = options.find((opt) => opt.value === value);
  return option?.label || value;
}

/**
 * Processes the value of the SelectInput component.
 * @param value {PathValue<T, Path<T>>}
 * @param options {OptionsOrGroups<SelectOption, GroupBase<SelectOption>>}
 * @param isMulti {boolean}
 * @constructor ValueProcessor
 */
function ValueProcessor<T extends Record<string, unknown>>(
  value: PathValue<T, Path<T>>,
  options: OptionsOrGroups<SelectOption, GroupBase<SelectOption>>,
  isMulti?: boolean,
) {
  if (isMulti) {
    const valuesArray = Array.isArray(value) ? value : [];
    return valuesArray.map((val) => ({
      value: val,
      label: getLabelByValue(val, options),
    }));
  }

  if (!value) return undefined;

  return {
    value: value,
    label: getLabelByValue(value, options),
  };
}

/**
 * Error message component for the SelectInput component.
 * @param isFetchError {boolean}
 * @param formError {string}
 * @constructor ErrorMessage
 */
function ErrorMessage({
  isFetchError,
  formError,
}: {
  isFetchError?: boolean;
  formError?: string;
}) {
  return (
    <p className="text-xs text-red-500">
      {isFetchError
        ? "An error has occurred! Please try again later."
        : formError}
    </p>
  );
}

/**
 * Props for the ContextMenu component.
 * @param key {string}
 * @param icon {React.ReactNode}
 * @param title {string}
 * @param onClick {() => void}
 * @constructor ContextMenuProps
 */
type ContextMenuProps = {
  key: string;
  icon?: React.ReactNode;
  title: string;
  onClick: () => void;
};

/**
 * Context Menu component used in the SelectInput component.
 * @param menuItems {ContextMenuProps[]}
 * @constructor ContextMenu
 * @returns {React.ReactElement}
 */
function ContextMenu({ menuItems }: { menuItems: ContextMenuProps[] }) {
  return (
    <Command className="rounded-lg border shadow-md max-h-[200px]">
      <CommandList>
        <CommandGroup heading="Actions">
          {menuItems.map((item) => (
            <CommandItem key={item.key} onClick={item.onClick}>
              {item.icon}
              <span>{item.title}</span>
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </Command>
  );
}

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
  contextMenuItems?: ContextMenuProps[];
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
  const ref = useRef<HTMLDivElement>(null);

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
    hasContextMenu = false,
    contextMenuItems = [],
    ...controllerProps
  } = props;

  const dataLoading = props.isLoading || props.isDisabled;
  const errorOccurred = props.isFetchError || !!fieldState.error?.message;
  const processedValue = ValueProcessor(field.value, options, isMulti);

  const [visible, setVisible] = useState(false);
  const [contextMenuVisible, setContextMenuVisible] = useState(false);

  const handleClickOutside = useCallback((event: MouseEvent) => {
    if (ref.current && !ref.current.contains(event.target as Node)) {
      setVisible(false);
      setContextMenuVisible(false);
    }
  }, []);

  useEffect(() => {
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [handleClickOutside]);

  const handleContextMenu = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (hasContextMenu) {
        e.preventDefault();
        setContextMenuVisible(true);
      }
    },
    [hasContextMenu],
  );

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
      <div
        className="relative"
        ref={ref}
        onClick={() => setVisible(true)}
        onContextMenu={handleContextMenu}
      >
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
          menuIsOpen={visible}
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
      {hasContextMenu && contextMenuVisible && (
        <ContextMenu menuItems={contextMenuItems} />
      )}
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
