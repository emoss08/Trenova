import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { AsyncSelectFieldProps, SelectOption } from "@/types/fields";
import debounce from "debounce";
import React, { useEffect, useMemo, useState } from "react";
import { Controller, FieldValues } from "react-hook-form";
import AsyncSelect from "react-select/async";
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

type ReactAsyncSelectInputProps = Omit<
  AsyncSelectFieldProps<any>,
  "label" | "control"
>;

const fetchOptions = async (link: string, inputValue: string, page: number) => {
  const limit = 50;
  const offset = (page - 1) * limit;

  try {
    const response = await http.get<{
      results: SelectOption[];
      next: string;
    }>(`${link}/select-options/`, {
      params: {
        search: inputValue,
        limit: limit.toString(),
        offset: offset.toString(),
      },
    });

    const { results, next } = response.data;

    return {
      options: results || [],
      hasMore: !!next,
    };
  } catch (error: any) {
    console.error("Error fetching options", error);
    return { options: [], hasMore: false };
  }
};

const ReactAsyncSelect = React.forwardRef<any, ReactAsyncSelectInputProps>(
  (
    {
      link,
      isReadOnly,
      isFetchError,
      isInvalid,
      isMulti,
      value,
      onChange,
      placeholder,
      isClearable,
      hasPopoutWindow,
      popoutLink,
      popoutLinkLabel,
      hasPermission,
      ...rest
    },
    ref,
  ) => {
    const isError = isFetchError || isInvalid;
    const [selectedOption, setSelectedOption] = useState<
      SelectOption | SelectOption[] | null
    >(null);
    const [inputValue, setInputValue] = useState("");

    useEffect(() => {
      // If value exists fetch the corresponding option
      if (value) {
        fetchOptions(link, "", 1).then(({ options }) => {
          if (isMulti) {
            const selected = options.filter((o) =>
              (value as unknown as (string | number | boolean)[]).includes(
                o.value as string | number | boolean,
              ),
            );
            setSelectedOption(selected);
          } else {
            const selected = options.find(
              (o) =>
                o.value === (value as unknown as string | number | boolean),
            );
            setSelectedOption(selected as SelectOption);
          }
        });
      }
    }, [value, isMulti, link]);

    const debouncedFetchOptions = useMemo(
      () =>
        debounce(
          (inputValue: string, callback: (options: SelectOption[]) => void) => {
            fetchOptions(link, inputValue, 1)
              .then(({ options }) => callback(options))
              .catch((error) => {
                console.error("Error in debouncedFetchOptions:", error);
                callback([]);
              });
          },
          300,
        ),
      [link],
    );

    const promiseOptions = (inputValue: string) =>
      new Promise<SelectOption[]>((resolve) => {
        debouncedFetchOptions(inputValue, resolve);
      });

    const handleChange = (selected: any) => {
      if (isMulti) {
        const newValue = (selected as SelectOption[]).map((opt) => opt.value);
        onChange(newValue);
        setSelectedOption(selected);
      } else {
        const newValue = (selected as SelectOption)?.value;
        onChange(newValue);
        setSelectedOption(selected);
      }
    };

    const handleInputChange = (inputValue: string) => {
      setInputValue(inputValue);
    };

    return (
      <AsyncSelect
        unstyled
        cacheOptions
        defaultOptions
        onInputChange={handleInputChange}
        onChange={handleChange}
        value={selectedOption}
        loadOptions={promiseOptions}
        inputValue={inputValue}
        isClearable={isClearable}
        placeholder={placeholder}
        hasPopoutWindow={hasPopoutWindow}
        popoutLink={popoutLink}
        popoutLinkLabel={popoutLinkLabel}
        hasPermission={hasPermission}
        styles={{
          control: () => ({
            cursor: "pointer",
            minHeight: "2rem",
          }),
          menuList: (base) => ({
            ...base,
            display: "flex",
            flexDirection: "column",
            padding: "0.25rem",
            gap: "0.1rem",
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
              "flex h-8 w-full rounded-md border border-muted-foreground/20 px-2 py-1.5 bg-muted text-sm",
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
          container: () => cn(isReadOnly && "cursor-not-allowed opacity-50"),
          valueContainer: () => cn("gap-1", isReadOnly && "cursor-not-allowed"),
          singleValue: () => "leading-7 ml-1",
          multiValue: () =>
            "bg-accent rounded items-center py-0.5 pl-2 pr-1 gap-0.5 h-6",
          multiValueLabel: () => "text-xs leading-4",
          multiValueRemove: () =>
            "hover:text-foreground/50 text-foreground rounded-md h-4 w-4",
          indicatorsContainer: () =>
            cn("gap-1", isReadOnly && "cursor-not-allowed"),
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
        ref={ref}
      />
    );
  },
);

ReactAsyncSelect.displayName = "ReactAsyncSelect";

export function AsyncSelectField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  isReadOnly,
  isMulti,
  isLoading,
  isFetchError,
  placeholder,
  link,
  isClearable,
  hasPopoutWindow,
  popoutLink,
  popoutLinkLabel,
  hasPermission,
}: Omit<AsyncSelectFieldProps<T>, "onChange" | "id" | "options">) {
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
          <ReactAsyncSelect
            link={link}
            isDisabled={disabled}
            id={inputId}
            ref={ref}
            name={name}
            isMulti={isMulti}
            onChange={onChange}
            isClearable={isClearable}
            placeholder={placeholder}
            onBlur={onBlur}
            onFocus={() => setIsOpen(true)}
            menuIsOpen={isOpen}
            onMenuOpen={() => setIsOpen(true)}
            onMenuClose={() => setIsOpen(false)}
            isFetchError={isFetchError}
            isReadOnly={isReadOnly}
            value={value}
            aria-describedby={cn(description && descriptionId, errorId)}
            aria-invalid={fieldState.invalid}
            isInvalid={fieldState.invalid}
            isLoading={isLoading}
            hasPopoutWindow={hasPopoutWindow}
            popoutLink={popoutLink}
            popoutLinkLabel={popoutLinkLabel}
            hasPermission={hasPermission}
          />
        </FieldWrapper>
      )}
    />
  );
}
