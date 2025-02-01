import { popoutWindowManager } from "@/hooks/popout-window/popout-window";
import { cn } from "@/lib/utils";
import { SelectOption } from "@/types/fields";
import { ExtendedOption } from "@/types/react-select-extension";
import { faCheck, faPlus, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { ChevronDownIcon } from "@radix-ui/react-icons";
import React from "react";
import { Path, PathValue } from "react-hook-form";
import {
  ClearIndicatorProps,
  components,
  DropdownIndicatorProps,
  GroupBase,
  GroupProps,
  IndicatorSeparatorProps,
  InputProps,
  MenuListProps,
  NoticeProps,
  OptionProps,
  OptionsOrGroups,
  SingleValueProps,
  ValueContainerProps,
} from "react-select";
import Highlight from "../ui/highlight";
import { Icon } from "../ui/icons";

interface AddNewButtonProps {
  label: string;
  popoutLink: string;
}

interface SelectProps {
  maxOptions?: number;
  popoutLink?: string;
  hasPopoutWindow?: boolean;
  hasPermission?: boolean;
  popoutLinkLabel?: string;
  inputValue?: string;
}

// Utility Functions
function getLabelByValue<T extends Record<string, unknown>>(
  value: PathValue<T, Path<T>>,
  options: OptionsOrGroups<SelectOption, GroupBase<SelectOption>>,
): string {
  const option = options.find((opt) => opt?.value === value);
  return option?.label || (value as string);
}

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

function openPopoutWindow(
  popoutLink: string,
  event: React.MouseEvent<HTMLButtonElement>,
) {
  event.preventDefault();
  event.stopPropagation();

  popoutWindowManager.openWindow(
    popoutLink,
    {},
    {
      modal: "create",
      width: 800,
      height: 800,
    },
  );
}

function useAddNewButton(selectProps: SelectProps | undefined) {
  const { popoutLink, hasPopoutWindow, hasPermission, popoutLinkLabel } =
    selectProps || {};

  const showAddNewButton = Boolean(
    hasPermission && popoutLink && hasPopoutWindow && popoutLinkLabel,
  );

  const addNewButton = showAddNewButton ? (
    <AddNewButton
      key="add-new-button"
      label={popoutLinkLabel as string}
      popoutLink={popoutLink as string}
    />
  ) : null;

  return { showAddNewButton, addNewButton, selectProps };
}

// Component Functions
export function Option<
  Option = ExtendedOption,
  IsMulti extends boolean = boolean,
  Group extends GroupBase<Option> = GroupBase<Option>,
>(props: OptionProps<Option, IsMulti, Group>) {
  const { isSelected, label, isFocused } = props;
  const data = props.data as ExtendedOption;
  const inputValue = props.selectProps.inputValue || "";

  return (
    <components.Option {...props}>
      <div
        className={cn(
          "group relative flex cursor-pointer select-none items-center gap-x-3 rounded-sm px-3 py-1.5 text-xs outline-hidden",
          isSelected && "bg-accent",
          isFocused && "bg-accent",
        )}
      >
        {data.icon ? (
          <Icon
            icon={data.icon}
            className={cn(
              "size-3",
              isSelected
                ? "text-foreground"
                : "text-muted-foreground group-hover:text-foreground",
            )}
          />
        ) : data.color ? (
          <span
            className="block size-2 rounded-full"
            style={{ backgroundColor: data.color }}
          />
        ) : null}
        <div className="flex flex-1 flex-col justify-center overflow-hidden">
          <span className="truncate">
            <Highlight text={label as string} highlight={inputValue} />
          </span>
          {data.description && (
            <span className="text-wrap text-xs text-foreground/70">
              <Highlight text={data.description} highlight={inputValue} />
            </span>
          )}
        </div>
        {isSelected && (
          <Icon
            icon={faCheck}
            className="absolute right-3 top-1/2 size-3 -translate-y-1/2"
          />
        )}
      </div>
    </components.Option>
  );
}

export function DropdownIndicator(props: DropdownIndicatorProps) {
  const { selectProps } = props;
  return (
    <components.DropdownIndicator {...props}>
      <ChevronDownIcon
        className={cn(
          "size-3 text-muted-foreground duration-200 ease-in-out transition-all",
          selectProps.menuIsOpen && "-rotate-180",
        )}
      />
    </components.DropdownIndicator>
  );
}

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

export function ClearIndicator(props: ClearIndicatorProps) {
  return (
    <components.ClearIndicator {...props}>
      <Icon icon={faXmark} className="size-3" />
    </components.ClearIndicator>
  );
}

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

export function LoadingMessage({ children, ...props }: NoticeProps) {
  return (
    <components.LoadingMessage {...props}>
      <div className="my-1 flex flex-col items-center justify-center">
        <p className="text-xs text-accent-foreground">
          {children || "Loading..."}
        </p>
      </div>
    </components.LoadingMessage>
  );
}

export function Group({ ...props }: GroupProps) {
  return (
    <div>
      <div className="px-3 pt-1 text-xs text-muted-foreground">
        {props.label}
      </div>
      {props.children}
    </div>
  );
}

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

export function SingleValueComponent(props: SingleValueProps) {
  const { selectProps, data, children } = props;

  const selectedOption = selectProps.options.find(
    (option: SelectOption | GroupBase<SelectOption>) =>
      option.value === (data as SelectOption).value,
  );

  const color = selectedOption ? (selectedOption as SelectOption).color : null;

  return (
    <components.SingleValue {...props}>
      <div className="flex items-center">
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

function AddNewButton({ label, popoutLink }: AddNewButtonProps) {
  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    openPopoutWindow(popoutLink, event);
  };

  return (
    <span
      className="flex h-7 w-full cursor-pointer items-center justify-between rounded-sm bg-transparent px-2 py-3.5 pl-3 text-xs font-normal text-foreground shadow-none hover:bg-accent hover:text-foreground/90"
      onClick={handleClick}
    >
      <span className="mr-2">{label} Entry</span>
      <Icon icon={faPlus} className="size-3" />
    </span>
  );
}

export function MenuList({ children, ...props }: MenuListProps<any, false>) {
  const { maxOptions } = props.selectProps || {};
  const { showAddNewButton, addNewButton } = useAddNewButton(props.selectProps);

  const renderChildren = () => {
    if (!Array.isArray(children)) return children;

    const slicedChildren = children.slice(0, maxOptions);

    return showAddNewButton
      ? [addNewButton, ...slicedChildren]
      : slicedChildren;
  };

  return (
    <components.MenuList {...props}>{renderChildren()}</components.MenuList>
  );
}

export function NoOptionsMessage({
  ...props
}: NoticeProps & { selectProps: SelectProps }) {
  const { addNewButton } = useAddNewButton(props.selectProps);

  return (
    <components.NoOptionsMessage {...props}>
      <div className="my-1 flex flex-col items-center justify-center">
        <p className="p-1 text-xs text-accent-foreground">
          No Options available...
        </p>
        {addNewButton}
      </div>
    </components.NoOptionsMessage>
  );
}
