import { useDebounce } from "@/hooks/use-debounce";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";

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
  extraSearchParams?: Record<string, string | string[]>,
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
  const commandListRef = useRef<HTMLDivElement>(null);

  // Animation frame reference for smooth scrolling
  const animationRef = useRef<number | null>(null);
  // Target scroll position for smooth scrolling
  const targetScrollRef = useRef<number | null>(null);
  // Current scroll velocity
  const velocityRef = useRef(0);

  // Memoize the fetch functions to prevent unnecessary recreation
  const fetchInitialValueFn = useCallback(async () => {
    if (value) {
      try {
        setLoading(true);
        const option = await fetchOptionById<T>(link, value);
        setSelectedOption(option);
      } catch (err) {
        setError(
          err instanceof Error ? err.message : "Failed to fetch initial value",
        );
      } finally {
        setLoading(false);
      }
    } else {
      setSelectedOption(null);
    }
  }, [value, link]);

  // Fetch initial value if it exists
  useEffect(() => {
    fetchInitialValueFn();
  }, [fetchInitialValueFn]);

  // Memoize the load options function
  const loadOptionsFn = useCallback(async () => {
    if (!open) return;

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
      setError(err instanceof Error ? err.message : "Failed to fetch options");
    } finally {
      setLoading(false);
    }
  }, [debouncedSearchTerm, open, page, link, extraSearchParams]);

  // Fetch options based on search term
  useEffect(() => {
    if (open) {
      loadOptionsFn();
    }
  }, [open, loadOptionsFn]);

  // Clean up animation frame on unmount
  useEffect(() => {
    return () => {
      if (animationRef.current !== null) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, []);

  // Smooth scrolling animation function
  const smoothScroll = useCallback(() => {
    if (!commandListRef.current || targetScrollRef.current === null) return;

    const element = commandListRef.current;
    const target = targetScrollRef.current;
    const current = element.scrollTop;

    // Distance to target
    const distance = target - current;

    // If we're very close to target, just set it and stop
    if (Math.abs(distance) < 0.5) {
      element.scrollTop = target;
      targetScrollRef.current = null;
      velocityRef.current = 0;
      return;
    }

    // Calculate spring-like animation
    // Spring factor: lower = smoother but slower
    const spring = 0.15;

    // Update velocity with spring physics
    velocityRef.current += distance * spring;
    // Apply damping to prevent oscillation
    velocityRef.current *= 0.8;

    // Apply velocity to scroll position
    element.scrollTop += velocityRef.current;

    // Continue animation
    animationRef.current = requestAnimationFrame(smoothScroll);
  }, []);

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

  // Handle wheel events with smooth scrolling
  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      if (
        commandListRef.current &&
        commandListRef.current.scrollHeight >
          commandListRef.current.clientHeight
      ) {
        const { scrollTop, scrollHeight, clientHeight } =
          commandListRef.current;
        const isScrollingDown = e.deltaY > 0;
        const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1;
        const isAtTop = scrollTop <= 0;

        // Allow parent scrolling only if we're at the boundaries and trying to scroll beyond
        if ((isAtBottom && isScrollingDown) || (isAtTop && !isScrollingDown)) {
          // Let the event propagate to parent
          return;
        }

        // Otherwise handle the scroll ourselves
        e.stopPropagation();
        e.preventDefault();

        // Apply sensitivity damping for smoother scrolling
        const scrollSensitivity = 0.6; // Lower value = less sensitive scrolling
        const deltaY = e.deltaY * scrollSensitivity;

        // Set target scroll position
        const currentScroll = commandListRef.current.scrollTop;
        targetScrollRef.current = currentScroll + deltaY;

        // Start animation if not already running
        if (animationRef.current === null) {
          animationRef.current = requestAnimationFrame(smoothScroll);
        }
      }
    },
    [smoothScroll],
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
          <CommandList
            ref={commandListRef}
            onScroll={handleScrollEnd}
            onWheel={handleWheel}
            className="max-h-[200px] overflow-y-auto scrollbar-thin scrollbar-thumb-gray-300 scrollbar-track-transparent"
          >
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
              {loading && (
                <div className="p-2 flex justify-center">
                  <PulsatingDots size={1} color="foreground" />
                </div>
              )}
              {hasMore && !loading && (
                <div className="p-2 text-xs text-center text-muted-foreground">
                  Scroll for more
                </div>
              )}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

// Create a memoized version that preserves the generic type
export const MemoizedAutocomplete = memo(Autocomplete) as typeof Autocomplete;

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
      render={({ field: { onChange, value, disabled }, fieldState }) => {
        // Memoize the wrapped component props to prevent unnecessary renders
        // eslint-disable-next-line react-hooks/rules-of-hooks
        const autocompleteProps = useMemo(
          () => ({
            link,
            preload,
            renderOption,
            getDisplayValue,
            getOptionValue,
            label,
            value,
            onChange,
            isInvalid: fieldState.invalid,
            onOptionChange,
            disabled,
            clearable,
            extraSearchParams,
            ...props,
          }),
          // eslint-disable-next-line react-hooks/exhaustive-deps
          [
            link,
            preload,
            renderOption,
            getDisplayValue,
            getOptionValue,
            label,
            value,
            onChange,
            fieldState.invalid,
            onOptionChange,
            disabled,
            clearable,
            extraSearchParams,
            props,
          ],
        );

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={!!rules?.required}
            error={fieldState.error?.message}
            className={className}
          >
            <MemoizedAutocomplete<TOption> {...autocompleteProps} />
          </FieldWrapper>
        );
      }}
    />
  );
}

// Add memoization to the FieldWrapper component
export const MemoizedAutocompleteField = memo(
  AutocompleteField,
) as typeof AutocompleteField;
