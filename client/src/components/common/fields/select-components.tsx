/*
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

import { Button } from "@/components/ui/button";
import { cn, PopoutWindow } from "@/lib/utils";
import {
  CaretSortIcon,
  CheckIcon,
  Cross2Icon,
  PlusIcon,
} from "@radix-ui/react-icons";
import { AlertTriangle } from "lucide-react";
import React from "react";
import { Path, PathValue } from "react-hook-form";
import {
  ClearIndicatorProps,
  components,
  DropdownIndicatorProps,
  GroupBase,
  IndicatorSeparatorProps,
  InputProps,
  MenuListProps,
  NoticeProps,
  OptionProps,
  OptionsOrGroups,
  SingleValueProps,
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
    <components.Option {...props}>
      <div className="hover:bg-accent relative my-1 flex cursor-default select-none rounded-sm px-3 py-1.5 text-xs outline-none hover:cursor-pointer hover:rounded-sm">
        {props.label}
        {props.isSelected && (
          <CheckIcon className="absolute right-3 top-1/2 size-4 -translate-y-1/2" />
        )}
      </div>
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
        <CaretSortIcon className="size-4 shrink-0" />
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
      <Cross2Icon className="size-4 shrink-0" />
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
  return <p className="text-foreground/70 text-xs">{description}</p>;
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
    popoutLink?: string;
    popoutLinkLabel?: string;
    hasPopoutWindow?: boolean;
  };
}) {
  return (
    <components.MenuList {...props}>
      {Array.isArray(children)
        ? children
            .slice(0, props.selectProps?.maxOptions)
            .map((child, index) => {
              if (
                index === 0 &&
                props.selectProps?.popoutLink &&
                props.selectProps?.hasPopoutWindow
              ) {
                return (
                  <React.Fragment key={index}>
                    <AddNewButton
                      label={props.selectProps?.popoutLinkLabel as string}
                      popoutLink={props.selectProps.popoutLink as string}
                    />
                    {child}
                  </React.Fragment>
                );
              } else {
                return child;
              }
            })
        : children}
    </components.MenuList>
  );
}

export function NoOptionsMessage({
  children,
  ...props
}: NoticeProps & {
  selectProps?: {
    maxOptions?: number;
    formError?: string;
    popoutLink?: string;
    popoutLinkLabel?: string;
    hasPopoutWindow?: boolean;
  };
}) {
  const { popoutLink, hasPopoutWindow } = props.selectProps || {};

  return (
    <components.NoOptionsMessage {...props}>
      <div className="my-1 flex flex-col items-center justify-center">
        <p className="text-accent-foreground my-1 text-sm">
          {children || "No options available..."}
        </p>
        {popoutLink && hasPopoutWindow && (
          <AddNewButton
            label={props.selectProps?.popoutLinkLabel as string}
            popoutLink={props.selectProps.popoutLink as string}
          />
        )}
      </div>
    </components.NoOptionsMessage>
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

export function InputComponent(
  props: InputProps & { selectProps: { isReadOnly?: boolean } },
) {
  return (
    <components.Input {...props} readOnly={props.selectProps.isReadOnly} />
  );
}

export function SingleValueComponent(props: SingleValueProps<any>) {
  const { selectProps, data, children } = props;

  // Find the option that matches the selected value
  const selectedOption = selectProps.options.find(
    (option) => option.value === data.value,
  );

  // Extract color from the selected option
  const color = selectedOption ? selectedOption.color : null;

  console.info("Selected option color", color);

  return (
    <components.SingleValue {...props}>
      <div className="flex items-center">
        {/* Display colored dot if color is available */}
        {color && (
          <span
            style={{
              backgroundColor: color,
              height: "10px",
              width: "10px",
              borderRadius: "50%",
              display: "inline-block",
              marginRight: "8px",
            }}
          ></span>
        )}
        <span>{children}</span>
      </div>
    </components.SingleValue>
  );
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
    <div className="mt-2 inline-block rounded bg-red-50 px-2 py-1 text-xs leading-tight text-red-500 dark:bg-red-300 dark:text-red-800 ">
      {isFetchError
        ? "An error has occurred! Please try again later."
        : formError}
    </div>
  );
}

/**
 * Popout window component for the SelectInput component.
 * @param popoutLink {string}
 * @param event
 */
function openPopoutWindow(
  popoutLink: string,
  event: React.MouseEvent<HTMLButtonElement>,
) {
  event.preventDefault();
  event.stopPropagation();

  PopoutWindow(popoutLink, {
    hideHeader: true,
  });
}

function AddNewButton({
  label,
  popoutLink,
}: {
  label: string;
  popoutLink: string;
}) {
  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    openPopoutWindow(popoutLink, event);
  };

  return (
    <Button
      className="bg-background text-foreground hover:bg-accent hover:text-foreground/90 flex w-full items-center justify-between rounded-sm py-3.5 pl-3 text-xs font-normal"
      size="xs"
      onClick={(event) => handleClick(event)}
    >
      <span className="mr-2">{label} Entry</span>
      <PlusIcon className="size-4" />
    </Button>
  );
}
