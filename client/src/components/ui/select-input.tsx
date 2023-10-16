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

import React from "react";
import Select, {
  ActionMeta,
  ClearIndicatorProps,
  components,
  DropdownIndicatorProps,
  GroupBase,
  MenuListProps,
  OptionProps,
  Props,
  ValueContainerProps,
} from "react-select";
import { Label } from "./label";
import { cn } from "@/lib/utils";
import { CaretSortIcon, CheckIcon, Cross2Icon } from "@radix-ui/react-icons";
import { AlertTriangle } from "lucide-react";
import CreatableSelect, { CreatableProps } from "react-select/creatable";
import { StateManagerAdditionalProps } from "react-select/dist/declarations/src/useStateManager";

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

function DropdownIndicator(
  props: DropdownIndicatorProps & {
    selectProps?: {
      isFetchError?: boolean;
      formError?: string;
    };
  },
) {
  const errorOccurred =
    props.selectProps?.isFetchError || props.selectProps?.formError;

  return (
    <components.DropdownIndicator {...props}>
      {errorOccurred ? (
        <AlertTriangle size={15} className="text-red-500" />
      ) : (
        <CaretSortIcon className="h-4 w-4 shrink-0" />
      )}
    </components.DropdownIndicator>
  );
}

function ClearIndicator(props: ClearIndicatorProps) {
  return (
    <components.ClearIndicator {...props}>
      <Cross2Icon className="h-4 w-4 shrink-0" />
    </components.ClearIndicator>
  );
}

function CustomValueContainer({ children, ...rest }: ValueContainerProps) {
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

export function SelectInput({
  ...props
}: Props & {
  label: string;
  withAsterisk?: boolean;
  description?: string;
  maxOptions?: number;
  isFetchError?: boolean;
  formError?: string;
}) {
  const {
    label,
    withAsterisk,
    description,
    placeholder,
    maxOptions = 10,
    isClearable = true,
    isMulti,
    isLoading,
    isDisabled,
    isFetchError,
    formError,
  } = props;

  const dataLoading = isLoading || isDisabled;

  const errorOccurred = isFetchError || !!formError;

  return (
    <>
      {label && (
        <Label
          className={cn("text-sm font-medium", withAsterisk && "required")}
          htmlFor={props.id}
        >
          {label}
        </Label>
      )}
      <div className="relative">
        <Select
          closeMenuOnSelect={!isMulti}
          hideSelectedOptions={false}
          unstyled
          isMulti={isMulti}
          isLoading={isLoading}
          isDisabled={dataLoading || errorOccurred}
          isClearable={isClearable}
          maxOptions={maxOptions}
          placeholder={placeholder || "Select"}
          isFetchError={isFetchError}
          formError={formError}
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
            ValueContainer: CustomValueContainer,
            DropdownIndicator: DropdownIndicator,
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
            placeholder: () => "text-muted-foreground pl-1 py-0.5 truncate",
            input: () => "pl-1 py-0.5",
            valueContainer: () => "p-1 gap-1",
            singleValue: () => "leading-7 ml-1",
            multiValue: () =>
              "bg-accent rounded items-center py-0.5 pl-2 pr-1 gap-1.5 h-6",
            multiValueLabel: () => "text-xs leading-4",
            multiValueRemove: () =>
              "hover:text-foreground/50 text-foreground rounded-md h-4 w-4",
            indicatorsContainer: () => "p-1 gap-1",
            clearIndicator: () =>
              "text-foreground/50 p-1 hover:text-foreground",
            indicatorSeparator: () => "bg-foreground/20 mt-1 h-6 w-px",
            dropdownIndicator: () =>
              "p-1 text-foreground/50 rounded-md hover:text-foreground",
            menu: () => "mt-2 p-1 border rounded-md bg-background shadow-lg",
            groupHeading: () => "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
            noOptionsMessage: () =>
              "text-muted-foreground p-2 bg-background rounded-sm",
          }}
          {...props}
        />
        {errorOccurred && (
          <p className="text-xs text-red-500">
            {isFetchError
              ? "An error has occured! Please try again later."
              : formError}
          </p>
        )}
        {description && !errorOccurred && (
          <p className="text-xs text-foreground/70">{description}</p>
        )}
      </div>
    </>
  );
}
export type Option = {
  readonly label: string;
  readonly value: string;
};

type isMulti = boolean;

export function CreatableSelectField({
  ...props
}: CreatableProps<Option, isMulti, GroupBase<Option>> & {
  onCreateOption: (inputValue: string) => void;
  onChange: (newValue: unknown, actionMeta: ActionMeta<unknown>) => void;
  label: string;
  withAsterisk?: boolean;
  description?: string;
  isFetchError?: boolean;
  formError?: string;
}) {
  const errorOccurred = props.isFetchError || !!props.formError;
  const dataLoading = props.isLoading || props.isDisabled;

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.withAsterisk && "required",
          )}
          htmlFor={props.id}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        <CreatableSelect
          isMulti={props.isMulti}
          isLoading={props.isLoading}
          isDisabled={dataLoading || errorOccurred}
          isClearable={props.isClearable}
          placeholder={props.placeholder || "Select"}
          closeMenuOnSelect={!props.isMulti}
          isFetchError={props.isFetchError}
          formError={props.formError}
          unstyled
          components={{
            ClearIndicator: ClearIndicator,
            ValueContainer: CustomValueContainer,
            DropdownIndicator: DropdownIndicator,
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
            placeholder: () => "text-muted-foreground pl-1 py-0.5 truncate",
            input: () => "pl-1 py-0.5",
            valueContainer: () => "p-1 gap-1",
            singleValue: () => "leading-7 ml-1",
            multiValue: () =>
              "bg-accent rounded items-center py-0.5 pl-2 pr-1 gap-1.5 h-6",
            multiValueLabel: () => "text-xs leading-4 text-foreground",
            multiValueRemove: ({ isFocused }) =>
              cn(
                isFocused
                  ? "bg-accent pr-1 h-6"
                  : "bg-accent hover:text-foreground/50 p-0",
              ),
            indicatorsContainer: () => "p-1 gap-1",
            clearIndicator: () =>
              "text-foreground/50 p-1 hover:text-foreground",
            indicatorSeparator: () => "bg-foreground/20 mt-1 h-6 w-px",
            dropdownIndicator: () =>
              "p-1 text-foreground/50 rounded-md hover:text-foreground",
            menu: () => "mt-2 p-1 border rounded-md bg-background shadow-lg",
            groupHeading: () => "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
            noOptionsMessage: () =>
              "text-muted-foreground p-2 bg-background rounded-sm",
          }}
          onChange={props.onChange}
          onCreateOption={props.onCreateOption}
          options={props.options}
          value={props.value}
        />
        {errorOccurred && (
          <p className="text-xs text-red-500">
            {props.isFetchError
              ? "An error has occured! Please try again later."
              : props.formError}
          </p>
        )}
        {props.description && !errorOccurred && (
          <p className="text-xs text-foreground/70">{props.description}</p>
        )}
      </div>
    </>
  );
}
