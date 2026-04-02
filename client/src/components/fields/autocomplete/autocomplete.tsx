import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { API_BASE_URL } from "@/lib/constants";
import { cn } from "@/lib/utils";
import type { API_ENDPOINTS, SELECT_OPTIONS_ENDPOINTS } from "@/types/server";
import { useQuery } from "@tanstack/react-query";
import React, { useCallback, useId, useMemo, useState } from "react";
import type { Control, Path, RegisterOptions } from "react-hook-form";
import { Controller, type FieldValues } from "react-hook-form";
import { FieldWrapper } from "../field-components";
import { AutocompleteCommandContent } from "./autocomplete-content";
import { AutocompleteTrigger } from "./autocomplete-input";

const optionRequestQueueByLink = new Map<string, Promise<void>>();
let optionRequestSequence = 0;

function logOptionRequestDebug(
  event: string,
  details: Record<string, unknown>,
) {
  if (!import.meta.env.DEV) return;
  console.debug("[AutocompleteOption]", event, details);
}

async function fetchOptionQueued(url: string, link: string): Promise<Response> {
  const previousRequest =
    optionRequestQueueByLink.get(link) ?? Promise.resolve();
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
      isAbortError:
        error instanceof DOMException && error.name === "AbortError",
      error:
        error instanceof Error
          ? { name: error.name, message: error.message }
          : String(error),
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

function getFallbackLookupLink(link: string): API_ENDPOINTS | null {
  if (!link.endsWith("select-options/")) {
    return null;
  }
  return link.replace("select-options/", "") as API_ENDPOINTS;
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
  filterFn?: (option: TOption, query: string) => boolean;
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
}

export type AutocompleteFieldProps<TOption, TForm extends FieldValues> = Omit<
  BaseAutocompleteFieldProps<TOption, TForm>,
  "onChange" | "value"
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
  disabled = false,
  className,
  triggerClassName,
  noResultsMessage,
  onOptionChange,
  isInvalid,
  clearable = false,
  extraSearchParams,
  popoutLink,
  initialLimit,
}: BaseAutocompleteFieldProps<TOption, TForm>) {
  const [open, setOpen] = useState(false);
  const listboxId = useId();
  const [userSelectedOptionState, setUserSelectedOptionState] = useState<{
    option: TOption;
    value: string;
  } | null>(null);

  const valueLookupLink = selectedValueLink ?? link;

  const { data: fetchedOption } = useQuery({
    queryKey: ["autocomplete-option", link, valueLookupLink, value],
    queryFn: async () => {
      if (!value) return null;
      const fetchURL = new URL(
        `${API_BASE_URL}${valueLookupLink}${value}`,
        window.location.origin,
      );

      let response: Response;
      try {
        response = await fetchOptionQueued(fetchURL.href, valueLookupLink);
      } catch (error) {
        const fallbackLookupLink = getFallbackLookupLink(valueLookupLink);
        if (!fallbackLookupLink) {
          throw error;
        }

        const fallbackURL = new URL(
          `${API_BASE_URL}${fallbackLookupLink}${value}`,
          window.location.origin,
        );
        logOptionRequestDebug("fallback_retry", {
          value,
          primaryLink: valueLookupLink,
          primaryUrl: fetchURL.href,
          fallbackLink: fallbackLookupLink,
          fallbackUrl: fallbackURL.href,
          error:
            error instanceof Error
              ? { name: error.name, message: error.message }
              : String(error),
        });
        response = await fetchOptionQueued(
          fallbackURL.href,
          fallbackLookupLink,
        );
      }

      if (!response.ok) {
        logOptionRequestDebug("not_ok", {
          link,
          valueLookupLink,
          value,
          url: fetchURL.href,
          status: response.status,
          statusText: response.statusText,
        });
        throw new Error("Failed to fetch option");
      }

      const data = await response.json();

      return data;
    },
    enabled:
      !!value &&
      (!userSelectedOptionState || userSelectedOptionState.value !== value),
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

  return (
    <div className="relative">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <AutocompleteTrigger
              open={open}
              disabled={disabled}
              isInvalid={isInvalid}
              triggerClassName={triggerClassName}
              clearable={clearable}
              currentValue={value}
              selectedOption={selectedOption}
              getDisplayValue={getDisplayValue}
              placeholder={placeholder}
              handleClear={handleClear}
              listboxId={listboxId}
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
            setSelectedOption={(option) =>
              setUserSelectedOptionState(
                option
                  ? { option, value: getOptionValue(option).toString() }
                  : null,
              )
            }
            selectedOption={selectedOption}
            onOptionChange={onOptionChange}
            onChange={onChange}
            clearable={clearable}
            value={value}
            noResultsMessage={noResultsMessage}
            extraSearchParams={extraSearchParams}
            initialLimit={initialLimit}
            popoutLink={popoutLink}
            listboxId={listboxId}
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
  placeholder,
  initialLimit,
  selectedValueLink,
  ...props
}: AutocompleteFieldProps<TOption, TForm>) {
  return (
    <Controller<TForm>
      name={name}
      control={control}
      rules={rules}
      render={({ field: { onChange, value, disabled }, fieldState }) => {
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
              label={label}
              initialLimit={initialLimit}
              selectedValueLink={selectedValueLink}
              placeholder={placeholder}
              value={value}
              onChange={onChange}
              disabled={disabled}
              isInvalid={fieldState.invalid}
              {...props}
            />
          </FieldWrapper>
        );
      }}
    />
  );
}
