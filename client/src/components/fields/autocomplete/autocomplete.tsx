import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { API_BASE_URL } from "@/lib/constants";
import {
  fetchGraphQLSelectedOption,
  selectOptionFiltersFromSearchParams,
  type GraphQLSelectOptionsConfig,
} from "@/lib/graphql/select-options";
import { cn } from "@/lib/utils";
import type { API_ENDPOINTS, SELECT_OPTIONS_ENDPOINTS } from "@/types/server";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useCallback, useId, useMemo, useState } from "react";
import type { Control, Path, RegisterOptions } from "react-hook-form";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "../field-components";
import { AutocompleteCommandContent } from "./autocomplete-content";
import { AutocompleteTrigger } from "./autocomplete-input";

const optionRequestQueueByLink = new Map<string, Promise<void>>();
let optionRequestSequence = 0;

function logOptionRequestDebug(event: string, details: Record<string, unknown>) {
  if (!import.meta.env.DEV) return;
  console.debug("[AutocompleteOption]", event, details);
}

async function fetchOptionQueued(url: string, link: string): Promise<Response> {
  const previousRequest = optionRequestQueueByLink.get(link) ?? Promise.resolve();
  const requestId = ++optionRequestSequence;
  const queuedAt = Date.now();

  let releaseQueue!: () => void;
  const currentRequest = new Promise<void>((resolve) => {
    releaseQueue = resolve;
  });

  optionRequestQueueByLink.set(
    link,
    previousRequest.then(() => currentRequest),
  );

  logOptionRequestDebug("queued", {
    requestId,
    link,
    url,
  });

  await previousRequest;
  const startedAt = Date.now();
  logOptionRequestDebug("started", {
    requestId,
    link,
    url,
    queuedMs: startedAt - queuedAt,
  });

  try {
    const response = await fetch(url, {
      credentials: "include",
    });
    logOptionRequestDebug("response", {
      requestId,
      link,
      url,
      ok: response.ok,
      status: response.status,
      statusText: response.statusText,
      durationMs: Date.now() - startedAt,
    });
    return response;
  } catch (error) {
    logOptionRequestDebug("network_error", {
      requestId,
      link,
      url,
      durationMs: Date.now() - startedAt,
      isAbortError: error instanceof DOMException && error.name === "AbortError",
      error: error instanceof Error ? { name: error.name, message: error.message } : String(error),
    });
    throw error;
  } finally {
    releaseQueue();
    logOptionRequestDebug("finished", {
      requestId,
      link,
      url,
      totalMs: Date.now() - queuedAt,
    });
    if (optionRequestQueueByLink.get(link) === currentRequest) {
      optionRequestQueueByLink.delete(link);
    }
  }
}

type SelectedValueLookupCandidate = {
  lookupLink: string;
  url: string;
};

function getFallbackLookupLink(link: string): API_ENDPOINTS | null {
  if (!link.endsWith("select-options/")) {
    return null;
  }
  return link.replace("select-options/", "") as API_ENDPOINTS;
}

function appendSelectedValueLookupCandidate(
  candidates: SelectedValueLookupCandidate[],
  seenURLs: Set<string>,
  lookupLink: string,
  value: string,
) {
  const url = new URL(`${API_BASE_URL}${lookupLink}${value}`, window.location.origin);

  if (!seenURLs.has(url.href)) {
    candidates.push({ lookupLink, url: url.href });
    seenURLs.add(url.href);
  }

  if (url.pathname.endsWith("/")) {
    return;
  }

  url.pathname = `${url.pathname}/`;
  if (!seenURLs.has(url.href)) {
    candidates.push({ lookupLink, url: url.href });
    seenURLs.add(url.href);
  }
}

export function buildSelectedValueLookupCandidates(
  lookupLink: string,
  value: string,
): SelectedValueLookupCandidate[] {
  const candidates: SelectedValueLookupCandidate[] = [];
  const seenURLs = new Set<string>();

  appendSelectedValueLookupCandidate(candidates, seenURLs, lookupLink, value);

  const fallbackLookupLink = getFallbackLookupLink(lookupLink);
  if (fallbackLookupLink) {
    appendSelectedValueLookupCandidate(candidates, seenURLs, fallbackLookupLink, value);
  }

  return candidates;
}

function isAuthFailure(response: Response): boolean {
  return response.status === 401 || response.status === 403;
}

function isRouteStyleLookupFailure(response: Response): boolean {
  return response.status === 404 || response.status === 405;
}

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
  /** Link to fetch options */
  link: SELECT_OPTIONS_ENDPOINTS;
  /** Optional link to fetch selected value details by id */
  selectedValueLink?: API_ENDPOINTS;
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
  /** Optional GraphQL select-options resource. REST link remains the compatibility fallback. */
  graphql?: GraphQLSelectOptionsConfig;
  /** Popout link to open in a new window */
  popoutLink?: string;

  /** Initial limit for the query */
  initialLimit?: number;
}

export type AutocompleteFieldProps<TOption, TForm extends FieldValues> = Omit<
  BaseAutocompleteFieldProps<TOption, TForm>,
  "onChange" | "value" | "onBlur" | "ref"
> &
  AutocompleteFormControlProps<TForm> & {
    description?: string;
  };

export function Autocomplete<TOption, TForm extends FieldValues>({
  link,
  selectedValueLink,
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
}: BaseAutocompleteFieldProps<TOption, TForm>) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const listboxId = useId();
  const [userSelectedOptionState, setUserSelectedOptionState] = useState<{
    option: TOption;
    value: string;
  } | null>(null);

  const valueLookupLink = selectedValueLink ?? link;
  const graphQLFilters = useMemo(
    () => ({
      ...selectOptionFiltersFromSearchParams(extraSearchParams),
      ...graphql?.filters,
    }),
    [extraSearchParams, graphql?.filters],
  );
  const normalizedGraphQLFilters = useMemo(
    () => (Object.keys(graphQLFilters).length > 0 ? graphQLFilters : undefined),
    [graphQLFilters],
  );

  const getOptionQueryKey = useCallback(
    (optionValue: string) => [
      "autocomplete-option",
      link,
      valueLookupLink,
      optionValue,
      graphql,
      graphql?.resource,
      normalizedGraphQLFilters,
    ],
    [link, valueLookupLink, graphql, normalizedGraphQLFilters],
  );

  const {
    data: fetchedOption,
    isLoading: isSelectedOptionLoading,
    isError: isSelectedOptionError,
  } = useQuery({
    queryKey: [
      "autocomplete-option",
      link,
      valueLookupLink,
      value,
      graphql,
      graphql?.resource,
      normalizedGraphQLFilters,
    ],
    queryFn: async () => {
      if (!value) return null;
      if (graphql) {
        return (await fetchGraphQLSelectedOption(
          graphql.resource,
          value,
          normalizedGraphQLFilters,
        )) as TOption | null;
      }

      const candidates = buildSelectedValueLookupCandidates(valueLookupLink, value);

      for (const [index, candidate] of candidates.entries()) {
        let response: Response;

        try {
          response = await fetchOptionQueued(candidate.url, candidate.lookupLink);
        } catch (error) {
          if (index === candidates.length - 1) {
            throw error;
          }

          const nextCandidate = candidates[index + 1];
          logOptionRequestDebug("fallback_retry", {
            value,
            primaryLink: candidate.lookupLink,
            primaryUrl: candidate.url,
            fallbackLink: nextCandidate.lookupLink,
            fallbackUrl: nextCandidate.url,
            error:
              error instanceof Error ? { name: error.name, message: error.message } : String(error),
          });
          continue;
        }

        if (response.ok) {
          return await response.json();
        }

        logOptionRequestDebug("not_ok", {
          link,
          valueLookupLink,
          value,
          url: candidate.url,
          status: response.status,
          statusText: response.statusText,
        });

        if (
          isAuthFailure(response) ||
          !isRouteStyleLookupFailure(response) ||
          index === candidates.length - 1
        ) {
          throw new Error("Failed to fetch option");
        }

        const nextCandidate = candidates[index + 1];
        logOptionRequestDebug("fallback_retry", {
          value,
          primaryLink: candidate.lookupLink,
          primaryUrl: candidate.url,
          fallbackLink: nextCandidate.lookupLink,
          fallbackUrl: nextCandidate.url,
          status: response.status,
          statusText: response.statusText,
        });
      }

      return null;
    },
    enabled: !!value && (!userSelectedOptionState || userSelectedOptionState.value !== value),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    retry: 1,
  });

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
          className={cn("dark w-(--anchor-width) rounded-md p-0", className)}
        >
          <AutocompleteCommandContent
            open={open}
            link={link}
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
  link,
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
  selectedValueLink,
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
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <Autocomplete<TOption, TForm>
              link={link}
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
              selectedValueLink={selectedValueLink}
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
