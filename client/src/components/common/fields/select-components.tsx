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
import { IconProp } from "@fortawesome/fontawesome-svg-core";
import { faTriangleExclamation } from "@fortawesome/pro-regular-svg-icons";
import { faPlus } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { CaretSortIcon, CheckIcon, Cross2Icon } from "@radix-ui/react-icons";
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
 * @type {SelectOption}
 */
export type SelectOption = {
  /**
   * Label to be displayed in the option.
   * @type {string}
   */
  readonly label: string;

  /**
   * Value to be displayed in the option.
   * @type {string | boolean | number}
   */
  readonly value: string | boolean | number;

  /**
   * Color to be displayed in the option.
   * @type {string}
   * @default undefined
   * @example "#FF0000"
   */
  readonly color?: string;

  /**
   * Description to be displayed in the option.
   * @type {string}
   * @default undefined
   * @example "This is a description"
   */
  readonly description?: string;

  /**
   * Icon to be displayed in the option.
   * @type {IconProp}
   * @default undefined
   * @example <FontAwesomeIcon icon={faPlus} />
   */
  readonly icon?: IconProp;
};

/**
 * Option component for the SelectInput component.
 * @param props {OptionProps}
 * @constructor Option
 */
export function Option({ ...props }: OptionProps) {
  const { isSelected, label } = props;
  const data = props.data as SelectOption;

  return (
    <components.Option {...props}>
      <div
        className={`group relative my-1 flex cursor-default select-none items-center gap-x-3 rounded-sm px-3 py-1.5 text-xs outline-none ${
          isSelected ? "bg-accent" : "hover:bg-accent"
        }`}
      >
        {data.icon ? (
          <FontAwesomeIcon
            icon={data.icon}
            className={`size-4 ${
              isSelected
                ? "text-foreground"
                : "text-muted-foreground group-hover:text-foreground"
            }`}
          />
        ) : data.color ? (
          <span
            className="block size-2 rounded-full"
            style={{ backgroundColor: data.color }}
          />
        ) : null}
        <div className="flex flex-1 flex-col justify-center overflow-hidden">
          <span className="truncate">{label}</span>
          {data.description && (
            <span className="text-foreground/70 truncate text-xs">
              {data.description}
            </span>
          )}
        </div>
        {isSelected && (
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
        <FontAwesomeIcon
          icon={faTriangleExclamation}
          className="text-red-500"
        />
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
 * MenuList component for the SelectInput component.
 * @param children {React.ReactNode}
 * @param props {MenuListProps}
 * @example <MenuList selectProps={{ maxOptions: 5 }} />
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

/**
 * LoadingMessage component for the SelectInput component.
 * @param children {React.ReactNode}
 * @param props {NoticeProps}
 * @example <LoadingMessage />
 */
export function LoadingMessage({ children, ...props }: NoticeProps) {
  return (
    <components.LoadingMessage {...props}>
      <div className="my-1 flex flex-col items-center justify-center">
        <p className="text-accent-foreground text-xs">
          {children || "Loading..."}
        </p>
      </div>
    </components.LoadingMessage>
  );
}

/**
 * NoOptionsMessage component for the SelectInput component.
 * @param children {React.ReactNode}
 * @param props {NoticeProps}
 * @example <NoOptionsMessage />
 */
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
        <p className="text-accent-foreground p-2 text-xs">{children}</p>
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
 * @returns {string}
 * @example getLabelByValue("value", options)
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
 * Input component for the SelectInput component.
 * @param props {InputProps & { selectProps: { isReadOnly?: boolean } }}
 * @constructor InputComponent
 */
export function InputComponent(
  props: InputProps & { selectProps: { isReadOnly?: boolean } },
) {
  return (
    <components.Input
      {...props}
      autoComplete="nope"
      readOnly={props.selectProps.isReadOnly}
    />
  );
}

/**
 * SingleValue component for the SelectInput component.
 * @param props {SingleValueProps<any>}
 * @constructor SingleValueComponent
 */
export function SingleValueComponent(props: SingleValueProps<any>) {
  const { selectProps, data, children } = props;

  // Find the option that matches the selected value
  const selectedOption = selectProps.options.find(
    (option) => option.value === data.value,
  );

  // Extract color from the selected option
  const color = selectedOption ? selectedOption.color : null;

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
 * @example <ErrorMessage isFetchError={true} />
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
        : formError || "An error has occurred! Please try again later."}
    </div>
  );
}

/**
 * Popout window component for the SelectInput component.
 * @param popoutLink {string}
 * @param event
 * @example openPopoutWindow("/add", event)
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

/**
 * Add new button component for the SelectInput component.
 * @param label {string}
 * @param popoutLink {string}
 * @example <AddNewButton label="Add" popoutLink="/add" />
 */
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
      className="text-foreground hover:bg-accent hover:text-foreground/90 flex w-full items-center justify-between rounded-sm bg-transparent py-3.5 pl-3 text-xs font-normal shadow-none"
      size="xs"
      onClick={(event) => handleClick(event)}
    >
      <span className="mr-2">{label} Entry</span>
      <FontAwesomeIcon icon={faPlus} className="size-3" />
    </Button>
  );
}
