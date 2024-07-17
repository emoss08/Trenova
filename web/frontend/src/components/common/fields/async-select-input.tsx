/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { cn } from "@/lib/utils";
import { API_ENDPOINTS } from "@/types/server";
import axios from "axios";
import debounce from "lodash-es/debounce";
import { useCallback, useEffect, useRef, useState } from "react";
import { Controller, UseControllerProps, useController } from "react-hook-form";
import { GroupBase } from "react-select";
import AsyncSelect, { AsyncProps } from "react-select/async";
import { FieldDescription } from "./components";
import { Label } from "./label";
import {
  ClearIndicator,
  DropdownIndicator,
  ErrorMessage,
  IndicatorSeparator,
  LoadingMessage,
  MenuList,
  NoOptionsMessage,
  Option,
  SelectOption,
  ValueContainer,
} from "./select-components";

interface AsyncSelectProps<T extends Record<string, unknown>>
  extends UseControllerProps<T>,
    Omit<
      AsyncProps<SelectOption, boolean, GroupBase<SelectOption>>,
      "defaultValue" | "name"
    > {
  valueKey?: string;
  label?: string;
  description?: string;
  maxOptions?: number;
  hasPopoutWindow?: boolean;
  popoutLink?: string;
  popoutLinkLabel?: string;
  isReadOnly?: boolean;
  link: API_ENDPOINTS;
}

const fetchOptions = async (
  link: string,
  inputValue: string,
  page: number,
  valueKey?: string,
): Promise<{ options: SelectOption[]; hasMore: boolean }> => {
  const limit = 50;
  const offset = (page - 1) * limit;
  try {
    // Modified API call to include exact match parameter
    const { data } = await axios.get(link, {
      params: {
        search: inputValue,
        limit,
        offset,
        exact_match: inputValue, // Add this parameter to request an exact match
      },
    });

    const options =
      data.results?.map((result: any) => ({
        value: result.id,
        label: result[valueKey || "name"],
      })) || [];

    // Check if exact match is in the results
    const hasExactMatch = options.some(
      (option: SelectOption) =>
        option.label.toLowerCase() === inputValue.toLowerCase() ||
        option.value.toString() === inputValue,
    );

    // If no exact match in results and we have an exact match from backend, add it
    if (!hasExactMatch && data.exact_match) {
      options.unshift({
        value: data.exact_match.id,
        label: data.exact_match[valueKey || "name"],
      });
    }

    return {
      options,
      hasMore: !!data.next,
    };
  } catch (error) {
    console.error("Error fetching options:", error);
    return { options: [], hasMore: false };
  }
};

const fetchInitialValue = async (
  link: string,
  id: string | number,
  valueKey?: string,
): Promise<SelectOption> => {
  try {
    const { data } = await axios.get(`${link}${id}/`);
    return { label: data[valueKey || "name"], value: data.id };
  } catch (error) {
    console.error("Error fetching initial value:", error);
    return { label: "Error fetching value", value: id };
  }
};

export function AsyncSelectInput<T extends Record<string, unknown>>({
  label,
  description,
  placeholder,
  menuPlacement = "auto",
  menuPosition = "absolute",
  hideSelectedOptions = false,
  hasPopoutWindow = false,
  popoutLink,
  popoutLinkLabel,
  isMulti,
  rules,
  link,
  valueKey,
  ...props
}: AsyncSelectProps<T>) {
  const { field, fieldState } = useController(props);
  const [defaultValue, setDefaultValue] = useState<SelectOption | null>(null);
  const [inputValue, setInputValue] = useState("");
  const initialFetchRef = useRef(false);

  useEffect(() => {
    const fetchInitial = async () => {
      if (
        (typeof field.value === "string" || typeof field.value === "number") &&
        !initialFetchRef.current
      ) {
        initialFetchRef.current = true;
        try {
          const initialValue = await fetchInitialValue(
            link,
            field.value,
            valueKey,
          );
          setDefaultValue(initialValue);
        } catch (error) {
          console.error("Error fetching initial value:", error);
        }
      }
    };
    fetchInitial();
  }, [field.value, link, valueKey]);

  const debouncedFetchOptions = useCallback(
    debounce(
      async (
        inputValue: string,
        callback: (options: SelectOption[]) => void,
      ) => {
        const { options } = await fetchOptions(link, inputValue, 1, valueKey);
        callback(options);
      },
      300,
    ),
    [link, valueKey],
  );
  const promiseOptions = (inputValue: string) =>
    new Promise<SelectOption[]>((resolve) => {
      debouncedFetchOptions(inputValue, resolve);
    });

  return (
    <>
      <span className="space-x-1">
        {label && <Label className="text-sm font-medium">{label}</Label>}
        {rules?.required && <span className="text-red-500">*</span>}
      </span>
      <div className="relative">
        <Controller
          name={props.name}
          control={props.control}
          render={({ field }) => (
            <AsyncSelect
              {...field}
              unstyled
              cacheOptions
              defaultOptions
              loadOptions={promiseOptions}
              value={defaultValue || field.value}
              inputValue={inputValue}
              onInputChange={(newValue) => setInputValue(newValue)}
              onChange={(selected) => {
                const newValue = isMulti
                  ? (selected as SelectOption[]).map((opt) => opt.value)
                  : (selected as SelectOption)?.value;
                field.onChange(newValue);
                setDefaultValue(isMulti ? null : (selected as SelectOption));
              }}
              isMulti={isMulti}
              placeholder={placeholder}
              isClearable
              isDisabled={props.isDisabled}
              menuPlacement={menuPlacement}
              menuPosition={menuPosition}
              hideSelectedOptions={hideSelectedOptions}
              hasPopoutWindow={hasPopoutWindow}
              popoutLink={popoutLink}
              popoutLinkLabel={popoutLinkLabel}
              components={{
                ClearIndicator,
                ValueContainer,
                DropdownIndicator,
                IndicatorSeparator,
                LoadingMessage,
                MenuList,
                Option,
                NoOptionsMessage,
              }}
              classNames={{
                control: ({ isFocused }) =>
                  cn(
                    isFocused
                      ? "flex h-9 w-full rounded-md border border-border bg-background text-sm sm:text-sm sm:leading-6 ring-1 ring-inset ring-foreground"
                      : "flex h-9 w-full rounded-md border border-border bg-background text-sm sm:text-sm sm:leading-6 disabled:cursor-not-allowed disabled:opacity-50",
                    fieldState.invalid &&
                      "ring-1 ring-inset ring-red-500 focus:ring-red-500 bg-red-500 bg-opacity-20",
                  ),
                placeholder: () =>
                  cn(
                    "text-muted-foreground pl-1 py-0.5 truncate",
                    fieldState.invalid && "text-red-500",
                  ),
                input: () => "pl-1 py-0.5",
                valueContainer: () => "p-1 gap-1",
                singleValue: () => "leading-7 ml-1",
                menuList: () => "p-1",
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
                menu: () => "mt-2 border rounded-md bg-popover shadow-lg",
                groupHeading: () =>
                  "ml-3 mt-2 mb-1 text-muted-foreground text-sm",
                noOptionsMessage: () =>
                  "text-muted-foreground p-2 bg-popover rounded-sm",
              }}
              name={field.name}
              ref={field.ref}
            />
          )}
        />
        {fieldState.invalid ? (
          <ErrorMessage formError={fieldState.error?.message} />
        ) : (
          <FieldDescription description={description!} />
        )}
      </div>
    </>
  );
}
