import { useDebounce } from "@/hooks/use-debounce";
import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import {
  AutocompleteFieldProps,
  BaseAutocompleteFieldProps,
} from "@/types/fields";
import { LimitOffsetResponse } from "@/types/server";
import { faCheck } from "@fortawesome/pro-regular-svg-icons";
import { CaretSortIcon } from "@radix-ui/react-icons";
import { Controller, FieldValues } from "react-hook-form";
import { Icon } from "../ui/icons";
import { PulsatingDots } from "../ui/pulsating-dots";
import { FieldWrapper } from "./field-components";

export interface Option {
  value: string;
  label: string;
  disabled?: boolean;
  description?: string;
  icon?: React.ReactNode;
}

async function fetchOptions<T>(
  link: string,
  inputValue: string,
  page: number,
  extraSearchParams?: Record<string, string>,
): Promise<LimitOffsetResponse<T>> {
  const limit = 10;
  const offset = (page - 1) * limit;

  const { data } = await http.get<LimitOffsetResponse<T>>(link, {
    params: {
      query: inputValue,
      limit: limit.toString(),
      offset: offset.toString(),
      ...extraSearchParams,
    },
  });

  return data;
}

async function fetchOptionById<T>(
  link: string,
  id: string | number,
): Promise<T> {
  const { data } = await http.get<T>(`${link}${id}/`);
  return data;
}

export function Autocomplete<T>({
  link,
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
  clearable = true,
  extraSearchParams,
}: BaseAutocompleteFieldProps<T>) {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<T[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedOption, setSelectedOption] = useState<T | null>(null);
  const debouncedSearchTerm = useDebounce(searchTerm, preload ? 0 : 300);
  const [hasMore, setHasMore] = useState(false);
  const [page, setPage] = useState(1);

  // Fetch initial value if it exists
  useEffect(() => {
    const fetchInitialValue = async () => {
      if (value) {
        try {
          setLoading(true);
          const option = await fetchOptionById<T>(link, value);
          setSelectedOption(option);
        } catch (err) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to fetch initial value",
          );
        } finally {
          setLoading(false);
        }
      } else {
        setSelectedOption(null);
      }
    };

    fetchInitialValue();
  }, [value, link]);

  // Fetch options based on search term
  useEffect(() => {
    const loadOptions = async () => {
      try {
        setLoading(true);
        setError(null);

        const response = await fetchOptions<T>(
          link,
          debouncedSearchTerm,
          page,
          extraSearchParams,
        );

        setOptions((prev) =>
          page === 1 ? response.results : [...prev, ...response.results],
        );
        setHasMore(!!response.next);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to fetch options",
        );
      } finally {
        setLoading(false);
      }
    };

    if (open) {
      loadOptions();
    }
  }, [debouncedSearchTerm, open, page, link, extraSearchParams]);

  const handleSelect = useCallback(
    (currentValue: string) => {
      const newValue = clearable && currentValue === value ? "" : currentValue;
      const selectedOpt = options.find(
        (opt) => getOptionValue(opt).toString() === currentValue,
      );

      setSelectedOption(selectedOpt || null);
      onChange(newValue);
      if (onOptionChange) {
        onOptionChange(selectedOpt || null);
      }
      setOpen(false);
    },
    [value, onChange, onOptionChange, clearable, options, getOptionValue],
  );

  const handleScrollEnd = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      const target = e.target as HTMLDivElement;
      const scrollBuffer = 50; // pixels before bottom to trigger load
      const distanceFromBottom =
        target.scrollHeight - (target.scrollTop + target.clientHeight);

      // Check if we're near the bottom and not already loading
      if (
        !loading &&
        hasMore &&
        distanceFromBottom <= scrollBuffer &&
        distanceFromBottom >= 0 // Prevent overscroll triggering
      ) {
        setPage((prev) => prev + 1);
      }
    },
    [loading, hasMore],
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn(
            "w-full font-normal gap-2 rounded border-muted-foreground/20 bg-muted px-1.5 data-[state=open]:border-blue-600 data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-blue-600/20",
            "[&_svg]:size-4 justify-between",
            "transition-[border-color,box-shadow] duration-200 ease-in-out",
            disabled && "opacity-50 cursor-not-allowed",
            isInvalid &&
              "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20 hover:border-red-500 hover:bg-red-500/20 data-[state=open]:border-red-500 data-[state=open]:bg-red-500/20 data-[state=open]:ring-red-500/20",
            triggerClassName,
          )}
          disabled={disabled}
        >
          {selectedOption ? (
            getDisplayValue(selectedOption)
          ) : (
            <p
              className={cn(
                "text-muted-foreground",
                isInvalid && "text-red-500",
              )}
            >
              {placeholder}
            </p>
          )}
          <CaretSortIcon className="opacity-50 size-7" />
          {loading && (
            <div className="absolute right-7">
              <PulsatingDots size={1} color="foreground" />
            </div>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent
        sideOffset={7}
        className={cn(
          "p-0 rounded-md w-[var(--radix-popover-trigger-width)]",
          className,
        )}
      >
        <Command shouldFilter={false} className="overflow-hidden">
          <div className="border-b w-full">
            <CommandInput
              className="bg-transparent h-8 truncate"
              placeholder={`Search ${label.toLowerCase()}...`}
              value={searchTerm}
              onValueChange={setSearchTerm}
            />
          </div>
          <CommandList onScroll={handleScrollEnd}>
            {error && (
              <div className="p-4 text-destructive text-center">{error}</div>
            )}
            {!loading && !error && options.length === 0 && (
              <CommandEmpty>
                {noResultsMessage ?? `No ${label.toLowerCase()} found.`}
              </CommandEmpty>
            )}
            <CommandGroup>
              {options.map((option) => (
                <CommandItem
                  className="[&_svg]:size-3 cursor-pointer font-normal"
                  key={getOptionValue(option).toString()}
                  value={getOptionValue(option).toString()}
                  onSelect={handleSelect}
                >
                  {renderOption(option)}
                  <Icon
                    icon={faCheck}
                    className={cn(
                      "ml-auto",
                      value === getOptionValue(option).toString()
                        ? "opacity-100"
                        : "opacity-0",
                    )}
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
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
  ...props
}: AutocompleteFieldProps<TOption, TForm>) {
  return (
    <Controller<TForm>
      name={name}
      control={control}
      rules={rules}
      render={({ field: { onChange, value, disabled }, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
          className={className}
        >
          <Autocomplete<TOption>
            link={link}
            preload={preload}
            renderOption={renderOption}
            getDisplayValue={getDisplayValue}
            getOptionValue={getOptionValue}
            label={label}
            value={value}
            onChange={onChange}
            isInvalid={fieldState.invalid}
            onOptionChange={onOptionChange}
            disabled={disabled}
            clearable={clearable}
            extraSearchParams={extraSearchParams}
            {...props}
          />
        </FieldWrapper>
      )}
    />
  );
}
