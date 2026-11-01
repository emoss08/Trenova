import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import {
  fetchGraphQLSelectedOption,
  selectOptionFiltersFromSearchParams,
  type GraphQLSelectOptionsConfig,
} from "@/lib/graphql/select-options";
import { cn } from "@trenova/shared/lib/utils";
import { queryOptions, useQuery, useQueryClient } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useFieldValueFormat } from "@trenova/shared/lib/form-field-registry";
import React, { useCallback, useId, useMemo, useState } from "react";
import type { Control, Path, RegisterOptions } from "react-hook-form";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper, type FieldLayout } from "../field-components";
import { AutocompleteCommandContent } from "./autocomplete-content";
import { AutocompleteTrigger } from "./autocomplete-input";

export interface AutocompleteFormControlProps<T extends FieldValues> {
  name: Path<T>;
  control: Control<T>;
  rules?: RegisterOptions<T, Path<T>>;
}

export interface BaseAutocompleteFieldProps<
  TOption,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _TForm extends FieldValues,
> {
  /** The GraphQL select-options resource the picker searches and names its value from. */
  graphql: GraphQLSelectOptionsConfig;
  /** Preload all data ahead of time */
  preload?: boolean;
  /** Function to filter options */
  filterOption?: (option: TOption) => boolean;
  /** Function to render each option */
  renderOption: (option: TOption) => React.ReactNode;
  /** Function to get the value from an option */
  getOptionValue: (option: TOption) => string | number;
  /** Function to get the display value for the selected option */
  getDisplayValue: (option: TOption) => React.ReactNode;
  /** Custom not found message */
  notFound?: React.ReactNode;
  /** Currently selected value */
  value: string | null | undefined;
  /** Callback when selection changes */
  onChange: (...event: any[]) => void;
  /** Callback fired when the popover closes (used for form touched state) */
  onBlur?: () => void;
  /** Ref forwarded to the trigger button */
  ref?: React.Ref<HTMLButtonElement>;
  /** Label for the select field */
  label?: string;
  /** Placeholder text when no selection */
  placeholder?: string;
  /** Disable the entire select */
  disabled?: boolean;
  /** Custom width for the popover */
  width?: string | number;
  /** Custom class names */
  className?: string;
  /** Custom trigger button class names */
  triggerClassName?: string;
  /** Custom no results message */
  noResultsMessage?: string;
  /** Allow clearing the selection */
  clearable?: boolean;
  /** Whether the field is invalid */
  isInvalid?: boolean;
  /** Callback when an option is selected (Specific to AutocompleteField) */
  onOptionChange?: (option: TOption | null) => void;
  /** Extra search params to append to the query */
  extraSearchParams?: Record<string, string | string[]>;
  /** Popout link to open in a new window */
  popoutLink?: string;

  /** Initial limit for the query */
  initialLimit?: number;
  /**
   * Lets the person pick what they typed when no option has that value: the
   * option this returns is offered as "Use …" and selected like any other.
   */
  createOption?: (text: string) => TOption;
}

export type AutocompleteFieldProps<TOption, TForm extends FieldValues> = Omit<
  BaseAutocompleteFieldProps<TOption, TForm>,
  "onChange" | "value" | "onBlur" | "ref"
> &
  AutocompleteFormControlProps<TForm> & {
    description?: React.ReactNode;
    layout?: FieldLayout;
  };

type SelectedOptionSource = {
  graphql: GraphQLSelectOptionsConfig;
  filters?: Record<string, unknown>;
};

/**
 * The record behind a picker's value: one query per source and value, shared by the
 * field, which shows it, and a change review, which names it, so neither asks twice.
 */
function selectedOptionQuery<TOption>(source: SelectedOptionSource, value: string) {
  return queryOptions({
    queryKey: [
      "autocomplete-option",
      value,
      source.graphql.resource,
      source.graphql.filters,
      source.filters,
    ],
    queryFn: async ({ signal }): Promise<TOption | null> => {
      if (!value) return null;
      return (await fetchGraphQLSelectedOption(source.graphql.resource, value, source.filters, {
        signal,
      })) as TOption | null;
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    retry: 1,
  });
}

/** A record's name in a change review, read from the picker's own query. */
function RecordName<TOption>({
  query,
  getDisplayValue,
  fallback,
}: {
  query: ReturnType<typeof selectedOptionQuery<TOption>>;
  getDisplayValue: (option: TOption) => React.ReactNode;
  fallback: string;
}) {
  const { data, isPending } = useQuery(query);
  if (isPending) {
    return <Skeleton className="inline-block h-3 w-20 align-middle" />;
  }
  return <>{data ? getDisplayValue(data) : fallback}</>;
}

export function Autocomplete<TOption, TForm extends FieldValues>({
  preload = false,
  renderOption,
  getOptionValue,
  getDisplayValue,
  label,
  placeholder = "Select...",
  value,
  onChange,
  onBlur,
  ref,
  disabled = false,
  className,
  triggerClassName,
  noResultsMessage,
  onOptionChange,
  isInvalid,
  clearable = false,
  extraSearchParams,
  graphql,
  popoutLink,
  initialLimit,
  filterOption,
  createOption,
}: BaseAutocompleteFieldProps<TOption, TForm>) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const listboxId = useId();
  const [userSelectedOptionState, setUserSelectedOptionState] = useState<{
    option: TOption;
    value: string;
  } | null>(null);

  const graphQLFilters = useMemo(
    () => ({
      ...selectOptionFiltersFromSearchParams(extraSearchParams),
      ...graphql.filters,
    }),
    [extraSearchParams, graphql.filters],
  );
  const normalizedGraphQLFilters = useMemo(
    () => (Object.keys(graphQLFilters).length > 0 ? graphQLFilters : undefined),
    [graphQLFilters],
  );

  const optionSource = useMemo<SelectedOptionSource>(
    () => ({ graphql, filters: normalizedGraphQLFilters }),
    [graphql, normalizedGraphQLFilters],
  );

  const getOptionQueryKey = useCallback(
    (optionValue: string) => selectedOptionQuery<TOption>(optionSource, optionValue).queryKey,
    [optionSource],
  );

  const {
    data: fetchedOption,
    isLoading: isSelectedOptionLoading,
    isError: isSelectedOptionError,
  } = useQuery({
    ...selectedOptionQuery<TOption>(optionSource, value ?? ""),
    enabled: !!value && (!userSelectedOptionState || userSelectedOptionState.value !== value),
  });

  useFieldValueFormat((optionValue) =>
    optionValue === null || optionValue === undefined || optionValue === "" ? undefined : (
      <RecordName
        query={selectedOptionQuery<TOption>(optionSource, String(optionValue))}
        getDisplayValue={getDisplayValue}
        fallback={String(optionValue)}
      />
    ),
  );

  const selectedOption = useMemo(() => {
    if (!value) return null;
    if (userSelectedOptionState?.value === value) {
      return userSelectedOptionState.option;
    }
    return fetchedOption ?? null;
  }, [value, userSelectedOptionState, fetchedOption]);

  const handleClear = useCallback(() => {
    setUserSelectedOptionState(null);
    onChange(null);
    if (onOptionChange) {
      onOptionChange(null);
    }
  }, [onChange, onOptionChange]);

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (nextOpen && disabled) return;
      setOpen(nextOpen);
      if (!nextOpen) {
        onBlur?.();
      }
    },
    [disabled, onBlur],
  );

  const handleTriggerKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLButtonElement>) => {
      if (!clearable || disabled || !value) return;
      if (e.key === "Backspace" || e.key === "Delete") {
        e.preventDefault();
        handleClear();
      }
    },
    [clearable, disabled, value, handleClear],
  );

  const handleUserSelectedOption = useCallback(
    (option: TOption | null) => {
      if (!option) {
        setUserSelectedOptionState(null);
        return;
      }
      const optionValue = getOptionValue(option).toString();
      setUserSelectedOptionState({ option, value: optionValue });
      queryClient.setQueryData(getOptionQueryKey(optionValue), option);
    },
    [getOptionValue, getOptionQueryKey, queryClient],
  );

  return (
    <div className="relative">
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger
          render={
            <AutocompleteTrigger
              ref={ref}
              open={open}
              disabled={disabled}
              isInvalid={isInvalid}
              triggerClassName={triggerClassName}
              clearable={clearable}
              currentValue={value}
              selectedOption={selectedOption}
              isLoadingSelected={isSelectedOptionLoading}
              isErrorSelected={isSelectedOptionError}
              getDisplayValue={getDisplayValue}
              placeholder={placeholder}
              handleClear={handleClear}
              listboxId={listboxId}
              onKeyDown={handleTriggerKeyDown}
            />
          }
        />
        <PopoverContent
          sideOffset={7}
          className={cn("w-(--anchor-width) rounded-md p-0", className)}
        >
          <AutocompleteCommandContent
            open={open}
            preload={preload}
            label={label}
            getOptionValue={getOptionValue}
            renderOption={renderOption}
            setOpen={setOpen}
            setSelectedOption={handleUserSelectedOption}
            selectedOption={selectedOption}
            onOptionChange={onOptionChange}
            onChange={onChange}
            clearable={clearable}
            value={value}
            noResultsMessage={noResultsMessage}
            extraSearchParams={extraSearchParams}
            graphql={graphql}
            initialLimit={initialLimit}
            popoutLink={popoutLink}
            listboxId={listboxId}
            filterOption={filterOption}
            createOption={createOption}
            onClear={() => {
              setUserSelectedOptionState(null);
              if (onOptionChange) {
                onOptionChange(null);
              }
            }}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}

export function AutocompleteField<TOption, TForm extends FieldValues>({
  label,
  name,
  control,
  rules,
  className,
  preload,
  renderOption,
  description,
  getOptionValue,
  getDisplayValue,
  onOptionChange,
  clearable,
  extraSearchParams,
  graphql,
  placeholder,
  initialLimit,
  filterOption,
  layout,
  ...props
}: AutocompleteFieldProps<TOption, TForm>) {
  return (
    <Controller<TForm>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => {
        return (
          <FieldWrapper
            name={name}
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
            layout={layout}
          >
            <Autocomplete<TOption, TForm>
              preload={preload}
              renderOption={renderOption}
              getOptionValue={getOptionValue}
              getDisplayValue={getDisplayValue}
              onOptionChange={onOptionChange}
              clearable={clearable}
              extraSearchParams={extraSearchParams}
              graphql={graphql}
              label={label}
              initialLimit={initialLimit}
              filterOption={filterOption}
              placeholder={placeholder}
              value={field.value}
              onChange={field.onChange}
              onBlur={field.onBlur}
              ref={field.ref}
              disabled={field.disabled}
              isInvalid={fieldState.invalid}
              {...props}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
