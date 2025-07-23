/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { useDebounce } from "@/hooks/use-debounce";
import { useCallback, useEffect, useRef, useState } from "react";

import { Badge } from "@/components/ui/badge";
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
import { Separator } from "@/components/ui/separator";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { multiSelectVariants } from "@/lib/variants/multi-select";
import {
  BaseMultiSelectAutocompleteFieldProps,
  MultiSelectAutocompleteFieldProps,
} from "@/types/fields";
import { LimitOffsetResponse } from "@/types/server";
import { faCheck } from "@fortawesome/pro-regular-svg-icons";
import { faXmark } from "@fortawesome/pro-solid-svg-icons";
import { ChevronDownIcon, Cross2Icon } from "@radix-ui/react-icons";
import { Controller, FieldValues } from "react-hook-form";
import { Icon } from "../ui/icons";
import { PulsatingDots } from "../ui/pulsating-dots";
import { FieldWrapper } from "./field-components";

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

async function fetchOptionsByIds<T>(link: string, ids: string[]) {
  if (!ids.length) return [];

  // Make individual requests for each ID
  const promises = ids.map(async (id) => {
    try {
      const { data } = await http.get<T>(`${link}${id}/`);
      return data;
    } catch (error) {
      console.error(`Failed to fetch option with id ${id}:`, error);
      return null;
    }
  });

  // Wait for all requests to complete and filter out any failed requests
  const results = await Promise.all(promises);
  return results.filter((result) => result !== null);
}

export function MultiSelectAutocomplete<T>({
  link,
  preload = false,
  renderOption,
  renderBadge,
  getOptionValue,
  getDisplayValue,
  label,
  placeholder = "Select options...",
  values = [],
  onChange,
  disabled = false,
  className,
  triggerClassName,
  noResultsMessage,
  onOptionsChange,
  isInvalid,
  maxCount = 1,
  extraSearchParams,
  nestedValues = false,
}: BaseMultiSelectAutocompleteFieldProps<T>) {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<T[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedOptions, setSelectedOptions] = useState<T[]>([]);
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

  // Fetch initial values if they exist
  useEffect(() => {
    const fetchInitialValues = async () => {
      if (values && values.length > 0) {
        try {
          setLoading(true);

          if (nestedValues) {
            // If values are already objects, use them directly
            const objects = values.filter((v): v is T => typeof v === "object");
            const strings = values.filter(
              (v): v is string => typeof v === "string",
            );

            let fetchedOptions: T[] = [...objects];

            // Fetch any string IDs that need to be converted to objects
            if (strings.length > 0) {
              const additionalOptions = await fetchOptionsByIds<T>(
                link,
                strings,
              );
              fetchedOptions = [...fetchedOptions, ...additionalOptions];
            }

            setSelectedOptions(fetchedOptions);
          } else {
            // Original behavior for string arrays
            const stringValues = values.map((v) =>
              typeof v === "string" ? v : getOptionValue(v as T).toString(),
            );
            const fetchedOptions = await fetchOptionsByIds<T>(
              link,
              stringValues,
            );
            setSelectedOptions(fetchedOptions);
          }
        } catch (err) {
          setError(
            err instanceof Error
              ? err.message
              : "Failed to fetch initial values",
          );
        } finally {
          setLoading(false);
        }
      } else {
        setSelectedOptions([]);
      }
    };

    fetchInitialValues();
  }, [values, link, nestedValues, getOptionValue]);

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
      const optionToToggle = options.find(
        (opt) => getOptionValue(opt).toString() === currentValue,
      );

      if (!optionToToggle) return;

      // Check if the option is already selected
      const isAlreadySelected = selectedOptions.some(
        (opt) => getOptionValue(opt).toString() === currentValue,
      );

      // Update selected options
      const newSelectedOptions = isAlreadySelected
        ? selectedOptions.filter(
            (opt) => getOptionValue(opt).toString() !== currentValue,
          )
        : [...selectedOptions, optionToToggle];

      setSelectedOptions(newSelectedOptions);

      // Update values passed to parent
      const newValues = nestedValues
        ? (newSelectedOptions as (string | T)[])
        : (newSelectedOptions.map((opt) => getOptionValue(opt).toString()) as (
            | string
            | T
          )[]);
      onChange(newValues);

      if (onOptionsChange) {
        onOptionsChange(newSelectedOptions);
      }
    },
    [
      options,
      selectedOptions,
      onChange,
      onOptionsChange,
      getOptionValue,
      nestedValues,
    ],
  );

  const removeOption = useCallback(
    (valueToRemove: string) => {
      const newSelectedOptions = selectedOptions.filter(
        (opt) => getOptionValue(opt).toString() !== valueToRemove,
      );

      setSelectedOptions(newSelectedOptions);

      // Update values passed to parent
      const newValues = nestedValues
        ? (newSelectedOptions as (string | T)[])
        : (newSelectedOptions.map((opt) => getOptionValue(opt).toString()) as (
            | string
            | T
          )[]);
      onChange(newValues);

      if (onOptionsChange) {
        onOptionsChange(newSelectedOptions);
      }
    },
    [selectedOptions, onChange, onOptionsChange, getOptionValue, nestedValues],
  );

  const handleClearAll = useCallback(() => {
    setSelectedOptions([]);
    onChange([]);
    if (onOptionsChange) {
      onOptionsChange([]);
    }
  }, [onChange, onOptionsChange]);

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

  const toggleSelectAll = useCallback(() => {
    if (selectedOptions.length === options.length) {
      handleClearAll();
    } else {
      const newSelectedOptions = [...options];
      setSelectedOptions(newSelectedOptions);

      // Update values passed to parent
      const newValues = nestedValues
        ? (newSelectedOptions as (string | T)[])
        : (newSelectedOptions.map((opt) => getOptionValue(opt).toString()) as (
            | string
            | T
          )[]);
      onChange(newValues);

      if (onOptionsChange) {
        onOptionsChange(newSelectedOptions);
      }
    }
  }, [
    options,
    selectedOptions,
    onChange,
    onOptionsChange,
    handleClearAll,
    getOptionValue,
    nestedValues,
  ]);

  return (
    <div className="flex flex-col gap-1">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className={cn(
              "w-full font-normal gap-2 cursor-auto rounded-md border-muted-foreground/20 bg-muted px-2 py-1.5 h-7",
              "data-[state=open]:border-blue-600 data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-blue-600/20",
              "justify-between [&_svg]:size-3 [&_svg]:shrink-0 cursor-pointer hover:bg-muted",
              "transition-[border-color,box-shadow] duration-200 ease-in-out",
              "whitespace-nowrap",
              disabled && "opacity-50 cursor-not-allowed",
              isInvalid &&
                "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20 hover:border-red-500 hover:bg-red-500/20 data-[state=open]:border-red-500 data-[state=open]:bg-red-500/20 data-[state=open]:ring-red-500/20",
              triggerClassName,
            )}
            disabled={disabled}
          >
            {selectedOptions.length > 0 ? (
              <div className="flex flex-wrap gap-1 py-1 w-full items-center justify-between">
                <div className="flex flex-wrap gap-1">
                  {selectedOptions.slice(0, maxCount).map((option) => (
                    <span
                      key={getOptionValue(option).toString()}
                      className={cn(
                        multiSelectVariants({ variant: "default" }),
                      )}
                    >
                      {renderBadge
                        ? renderBadge(option)
                        : getDisplayValue(option)}
                      <span
                        className="size-4 cursor-pointer"
                        onClick={(event) => {
                          event.stopPropagation();
                          removeOption(getOptionValue(option).toString());
                        }}
                      >
                        <Icon icon={faXmark} />
                      </span>
                    </span>
                  ))}
                  {selectedOptions.length > maxCount && (
                    <Badge
                      withDot={false}
                      className={cn(
                        "max-h-5 h-auto bg-transparent text-foreground border-foreground/10 hover:bg-transparent",
                        multiSelectVariants({ variant: "default" }),
                      )}
                    >
                      {`+ ${selectedOptions.length - maxCount} more`}
                    </Badge>
                  )}
                </div>
                <div className="flex items-center">
                  {selectedOptions.length > 0 && (
                    <div className="flex items-center gap-0.5">
                      <span
                        title="Clear all"
                        className="size-3 text-muted-foreground duration-200 ease-in-out transition-all mx-1 cursor-pointer"
                        onClick={(event) => {
                          event.stopPropagation();
                          handleClearAll();
                        }}
                      >
                        <Cross2Icon />
                      </span>
                      <Separator
                        orientation="vertical"
                        className="min-h-4 h-full bg-foreground/10"
                      />
                      <span
                        title="Toggle dropdown"
                        className="size-3 text-muted-foreground duration-200 ease-in-out transition-all cursor-pointer items-center justify-center mr-0.5"
                        onClick={(event) => {
                          event.stopPropagation();
                          setOpen(!open);
                        }}
                      >
                        <ChevronDownIcon
                          className={cn(
                            "size-3 flex-shrink-0 ml-1 text-muted-foreground duration-200 ease-in-out transition-all cursor-pointer",
                            open && "transform -rotate-180",
                          )}
                        />
                      </span>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <>
                <p
                  className={cn(
                    "text-muted-foreground",
                    isInvalid && "text-red-500",
                  )}
                >
                  {placeholder}
                </p>
                <span
                  className="size-3 text-muted-foreground duration-200 ease-in-out transition-all cursor-pointer items-center justify-center mr-0.5"
                  onClick={(event) => {
                    event.stopPropagation();
                    setOpen(!open);
                  }}
                >
                  <ChevronDownIcon
                    className={cn(
                      "size-3 opacity-50 flex-shrink-0 ml-1 duration-200 ease-in-out transition-all cursor-pointer",
                      open && "transform -rotate-180",
                    )}
                  />
                </span>
              </>
            )}
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
                className="bg-transparent h-7 [&_[cmdk-input]]:h-7 truncate"
                placeholder={`Search ${label?.toLowerCase()}...`}
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
                  {noResultsMessage ?? `No ${label?.toLowerCase()} found.`}
                </CommandEmpty>
              )}
              <CommandGroup>
                {options.length > 0 && (
                  <CommandItem
                    key="select-all"
                    onSelect={() => toggleSelectAll()}
                    className="cursor-pointer"
                  >
                    <div
                      className={cn(
                        "flex size-4 items-center justify-center rounded-sm border border-primary",
                        selectedOptions.length === options.length &&
                          options.length > 0
                          ? "bg-primary text-primary-foreground"
                          : "opacity-50 [&_svg]:invisible",
                      )}
                    >
                      <Icon icon={faCheck} className="size-2" />
                    </div>
                    <span>(Select All)</span>
                  </CommandItem>
                )}
                {options.map((option) => {
                  const optionValue = getOptionValue(option).toString();
                  const isSelected = selectedOptions.some(
                    (selected) =>
                      getOptionValue(selected).toString() === optionValue,
                  );

                  return (
                    <CommandItem
                      className="[&_svg]:size-3 cursor-pointer font-normal"
                      key={optionValue}
                      value={optionValue}
                      onSelect={handleSelect}
                    >
                      {renderOption(option)}
                      <Icon
                        icon={faCheck}
                        className={cn(
                          "ml-auto size-2",
                          isSelected ? "opacity-100" : "opacity-0",
                        )}
                      />
                    </CommandItem>
                  );
                })}
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
    </div>
  );
}

export function MultiSelectAutocompleteField<
  TOption,
  TForm extends FieldValues,
>({
  label,
  name,
  control,
  rules,
  className,
  link,
  preload,
  renderOption,
  renderBadge,
  description,
  getOptionValue,
  getDisplayValue,
  onOptionsChange,
  extraSearchParams,
  ...props
}: MultiSelectAutocompleteFieldProps<TOption, TForm>) {
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
          <MultiSelectAutocomplete<TOption>
            link={link}
            preload={preload}
            renderOption={renderOption}
            renderBadge={renderBadge}
            getDisplayValue={getDisplayValue}
            getOptionValue={getOptionValue}
            label={label}
            values={Array.isArray(value) ? value : []}
            onChange={onChange}
            isInvalid={fieldState.invalid}
            onOptionsChange={onOptionsChange}
            disabled={disabled}
            extraSearchParams={extraSearchParams}
            {...props}
          />
        </FieldWrapper>
      )}
    />
  );
}
