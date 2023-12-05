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
import { CaretSortIcon, CheckIcon, Cross2Icon } from "@radix-ui/react-icons";
import { AlertTriangle } from "lucide-react";
import React from "react";
import { Path, PathValue } from "react-hook-form";
import {
  ClearIndicatorProps,
  components,
  DropdownIndicatorProps,
  GroupBase,
  IndicatorSeparatorProps,
  MenuListProps,
  OptionProps,
  OptionsOrGroups,
  ValueContainerProps,
} from "react-select";

/**
 * Option type for the SelectInput component.
 */
export type SelectOption = {
  readonly label: string;
  readonly value: string | boolean | number;
};

/**
 * Option component for the SelectInput component.
 * @param props {OptionProps}
 * @constructor Option
 */
export function Option({ ...props }: OptionProps) {
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
export function DropdownIndicator(props: DropdownIndicatorProps) {
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
export function IndicatorSeparator(props: IndicatorSeparatorProps) {
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
export function ClearIndicator(props: ClearIndicatorProps) {
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
export function ValueContainer({ children, ...rest }: ValueContainerProps) {
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
export function SelectDescription({ description }: { description: string }) {
  return <p className="text-xs text-foreground/70">{description}</p>;
}

/**
 * MenuList component for the SelectInput component.
 * @param children {React.ReactNode}
 * @param props {MenuListProps}
 * @constructor MenuList
 */
export function MenuList({
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
export function getLabelByValue<T extends Record<string, unknown>>(
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
export function ValueProcessor<T extends Record<string, unknown>>(
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
export function ErrorMessage({
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
