import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { AsyncSelectFieldProps, SelectOption } from "@/types/fields";
import debounce from "debounce";
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
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

async function fetchOptions(
  link: string,
  inputValue: string,
  page: number,
  valueKey?: string | string[],
  signal?: AbortSignal,
): Promise<{ options: SelectOption[]; hasMore: boolean }> {
  const limit = 10;
  const offset = (page - 1) * limit;

  try {
    const { data } = await http.get<{
      results: SelectOption[];
      next: string;
      query: string;
    }>(link, {
      params: {
        query: inputValue,
        limit: limit.toString(),
        offset: offset.toString(),
      },
      signal,
    });

    const formatLabel = (result: any) =>
      Array.isArray(valueKey)
        ? valueKey
            .map((key) => result[key])
            .filter(Boolean)
            .join(" ")
        : result[valueKey || "name"];

    const options =
      data.results?.map((result: any) => ({
        value: result.id,
        label: formatLabel(result),
        color: result?.color,
      })) || [];

    return {
      options,
      hasMore: !!data.next,
    };
  } catch (error) {
    if (error instanceof Error && error.name === "AbortError") {
      return { options: [], hasMore: false };
    }
    console.error("Error fetching options:", error);
    return { options: [], hasMore: false };
  }
}

const fetchInitialValue = async (
  link: string,
  id: string | number,
  valueKey?: string | string[],
): Promise<SelectOption> => {
  try {
    const { data } = await http.get<any>(`${link}${id}/`);

    let label: string;
    if (Array.isArray(valueKey)) {
      label = valueKey
        .map((key) => data[key])
        .filter(Boolean)
        .join(" ");
    } else {
      label = data[valueKey || "name"];
    }

    return { label, value: data.id, color: data?.color };
  } catch (error) {
    console.error("Error fetching initial value:", error);
    return { label: "Error fetching value", value: id };
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
      valueKey,
      // ...rest
    },
    ref,
  ) => {
    const isError = useMemo(
      () => isFetchError || isInvalid,
      [isFetchError, isInvalid],
    );
    const [selectedOption, setSelectedOption] = useState<
      SelectOption | SelectOption[] | null
    >(null);
    const [inputValue, setInputValue] = useState("");
    const previousValueRef = useRef<any>(value);
    const optionsCache = useRef<SelectOption[]>([]);
    const mountedRef = useRef(false);

    useEffect(() => {
      const abortController = new AbortController();

      // Skip if the value hasn't actually changed and component is already mounted
      if (value === previousValueRef.current && mountedRef.current) {
        return;
      }

      const fetchAndSetOption = async () => {
        // If value is explicitly null/undefined/empty, just clear and exit
        if (
          value === null ||
          value === undefined ||
          (value as unknown as string) === ""
        ) {
          setSelectedOption(null);
          previousValueRef.current = value;
          return;
        }

        try {
          if (!value) {
            setSelectedOption(null);
            return;
          }

          const { options } = await fetchOptions(link, "", 1, valueKey);

          if (isMulti) {
            const selected = options.filter((o) =>
              (value as unknown as (string | number | boolean)[]).includes(
                o.value as string | number | boolean,
              ),
            );

            if (selected.length > 0) {
              setSelectedOption(selected);
            } else if ((value as unknown as string[]).length > 0) {
              const missingOptions = await Promise.all(
                (value as unknown as string[]).map((v) =>
                  fetchInitialValue(link, v, valueKey),
                ),
              );
              setSelectedOption(missingOptions);
            }
          } else {
            const selected = options.find(
              (o) =>
                o.value === (value as unknown as string | number | boolean),
            );

            if (selected) {
              setSelectedOption(selected);
            } else if (value) {
              const option = await fetchInitialValue(
                link,
                value as unknown as string | number,
                valueKey,
              );
              setSelectedOption(option);
            }
          }

          previousValueRef.current = value;
        } catch (error) {
          console.error("Error fetching options:", error);
          setSelectedOption(null);
          previousValueRef.current = value;
        }
      };

      fetchAndSetOption();
      mountedRef.current = true;

      return () => {
        abortController.abort();
        mountedRef.current = false; // Reset mounted state on cleanup
      };
    }, [value, isMulti, link, valueKey]);

    const handleChange = useCallback(
      (selected: any) => {
        if (isMulti) {
          const newValue = selected
            ? selected.map((opt: SelectOption) => opt.value)
            : [];
          onChange(newValue);
          setSelectedOption(selected);
          previousValueRef.current = newValue;
        } else {
          const newValue = selected ? selected.value : null;
          onChange(newValue);
          setSelectedOption(selected);
          previousValueRef.current = newValue;
        }
      },
      [isMulti, onChange],
    );

    const debouncedFetchOptions = useMemo(
      () =>
        debounce(
          (inputValue: string, callback: (options: SelectOption[]) => void) => {
            // Allow empty input for initial load, but require 2+ chars for search
            if (inputValue && inputValue.length < 2) {
              callback([]);
              return;
            }

            fetchOptions(link, inputValue, 1, valueKey)
              .then(({ options }) => callback(options))
              .catch((error) => {
                console.error("Error in debouncedFetchOptions:", error);
                callback([]);
              });
          },
          300,
          { immediate: false },
        ),
      [link, valueKey],
    );

    // Fetch initial options on mount
    useEffect(() => {
      fetchOptions(link, "", 1, valueKey)
        .then(({ options }) => {
          optionsCache.current = options;
        })
        .catch((error) => {
          console.error("Error fetching initial options:", error);
        });
    }, [link, valueKey]);

    const promiseOptions = useCallback(
      (inputValue: string) =>
        new Promise<SelectOption[]>((resolve) => {
          // For initial load (empty input), use cached options if available
          if (!inputValue && optionsCache.current.length > 0) {
            resolve(optionsCache.current);
            return;
          }

          debouncedFetchOptions(inputValue, resolve);
        }),
      [debouncedFetchOptions],
    );

    const handleInputChange = useCallback((inputValue: string) => {
      setInputValue(inputValue);
    }, []);

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
  valueKey,
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
            valueKey={valueKey}
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
