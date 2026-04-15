"use no memo";
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
import { useDebounce } from "@/hooks/use-debounce";
import { API_BASE_URL } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { multiSelectVariants } from "@/lib/variants/async-multi-select";
import { type GenericLimitOffsetResponse } from "@/types/server";
import { CheckIcon, ChevronDownIcon, XIcon } from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
  useReducer,
  useRef,
  useState,
  type ComponentPropsWithoutRef,
} from "react";
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
  type Path,
  type RegisterOptions,
} from "react-hook-form";
import { Spinner } from "../ui/spinner";
import { FieldWrapper } from "./field-components";

export interface BaseMultiSelectAutocompleteFieldProps<TOption> {
  link: string;
  preload?: boolean;
  renderOption: (option: TOption) => React.ReactNode;
  renderBadge?: (option: TOption) => React.ReactNode;
  getOptionValue: (option: TOption) => string | number;
  getDisplayValue: (option: TOption) => string;
  label?: string;
  placeholder?: string;
  values?: (string | TOption)[];
  onChange: (values: (string | TOption)[]) => void;
  onOptionsChange?: (options: TOption[]) => void;
  disabled?: boolean;
  className?: string;
  triggerClassName?: string;
  noResultsMessage?: string;
  isInvalid?: boolean;
  maxCount?: number;
  extraSearchParams?: Record<string, string>;
  nestedValues?: boolean;
}

export interface MultiSelectAutocompleteProps<TOption>
  extends BaseMultiSelectAutocompleteFieldProps<TOption>,
    Omit<ComponentPropsWithoutRef<"button">, "onChange"> {}

export interface MultiSelectAutocompleteFieldProps<
  TOption,
  TForm extends FieldValues,
> {
  name: FieldPath<TForm>;
  control: Control<TForm>;
  rules?: RegisterOptions<TForm, Path<TForm>>;
  label?: string;
  description?: string;
  className?: string;
  link: string;
  preload?: boolean;
  renderOption: (option: TOption) => React.ReactNode;
  renderBadge?: (option: TOption) => React.ReactNode;
  getOptionValue: (option: TOption) => string | number;
  getOptionLabel?: (option: TOption) => string;
  getDisplayValue: (option: TOption) => string;
  onOptionsChange?: (options: TOption[]) => void;
  placeholder?: string;
  noResultsMessage?: string;
  triggerClassName?: string;
  maxCount?: number;
  animation?: number;
  extraSearchParams?: Record<string, string>;
  nestedValues?: boolean;
}

async function fetchOptions<T>(
  link: string,
  inputValue: string,
  page: number,
  extraSearchParams?: Record<string, string>,
): Promise<GenericLimitOffsetResponse<T>> {
  const limit = 10;
  const offset = (page - 1) * limit;

  const fetchURL = new URL(`${API_BASE_URL}${link}`, window.location.origin);
  fetchURL.searchParams.set("limit", limit.toString());
  fetchURL.searchParams.set("offset", offset.toString());
  if (inputValue) {
    fetchURL.searchParams.set("query", inputValue);
  }

  if (extraSearchParams) {
    Object.entries(extraSearchParams).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        fetchURL.searchParams.set(key, value.toString());
      }
    });
  }

  const response = await fetch(fetchURL.href, {
    credentials: "include",
  });

  return response.json();
}

async function fetchOptionsByIds(
  link: string,
  ids: string[],
  extraSearchParams?: Record<string, string>,
) {
  if (!ids.length) {
    return [];
  }

  const promises = ids.map(async (id) => {
    try {
      const fetchURL = new URL(
        `${API_BASE_URL}${link}${id}`,
        window.location.origin,
      );

      if (extraSearchParams) {
        Object.entries(extraSearchParams).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            fetchURL.searchParams.set(key, value.toString());
          }
        });
      }

      const response = await fetch(fetchURL.href, {
        credentials: "include",
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch option with id ${id}`);
      }

      return response.json();
    } catch (error) {
      console.error(`Failed to fetch option with id ${id}:`, error);
      return null;
    }
  });

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
  type AsyncState = {
    options: T[];
    loading: boolean;
    error: string | null;
    hasMore: boolean;
  };
  type AsyncAction =
    | { type: "fetchStart" }
    | {
        type: "fetchSuccess";
        results: T[];
        append: boolean;
        hasMore: boolean;
      }
    | { type: "fetchError"; error: string }
    | { type: "fetchEnd" };

  const asyncStateReducer = (
    state: AsyncState,
    action: AsyncAction,
  ): AsyncState => {
    switch (action.type) {
      case "fetchStart":
        return { ...state, loading: true, error: null };
      case "fetchSuccess":
        return {
          ...state,
          options: action.append
            ? [...state.options, ...action.results]
            : action.results,
          hasMore: action.hasMore,
        };
      case "fetchError":
        return { ...state, error: action.error };
      case "fetchEnd":
        return { ...state, loading: false };
      default:
        return state;
    }
  };

  const listboxId = useId();
  const [open, setOpen] = useState(false);
  const [asyncState, dispatchAsyncState] = useReducer(asyncStateReducer, {
    options: [],
    loading: false,
    error: null,
    hasMore: false,
  });
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedOptions, setSelectedOptions] = useState<T[]>([]);
  const debouncedSearchTerm = useDebounce(searchTerm, preload ? 0 : 300);
  const [page, setPage] = useState(1);
  const commandListRef = useRef<HTMLDivElement>(null);
  const { options, loading, error, hasMore } = asyncState;

  const animationRef = useRef<number | null>(null);
  const targetScrollRef = useRef<number | null>(null);
  const velocityRef = useRef(0);
  const prevValuesRef = useRef<string>("");

  useEffect(() => {
    const fetchInitialValues = async () => {
      if (values.length === 0) {
        if (prevValuesRef.current !== "") {
          setSelectedOptions([]);
          prevValuesRef.current = "";
        }
        return;
      }

      const currentValuesKey = JSON.stringify(
        values.map((v) =>
          typeof v === "string" ? v : getOptionValue(v as T).toString(),
        ),
      );

      if (prevValuesRef.current === currentValuesKey) {
        return;
      }

      prevValuesRef.current = currentValuesKey;
      dispatchAsyncState({ type: "fetchStart" });

      const resolveInitialOptions = nestedValues
        ? (async () => {
            const objects = values.filter((v): v is T => typeof v === "object");
            const strings = values.filter(
              (v): v is string => typeof v === "string",
            );
            const additionalOptionsPromise =
              strings.length > 0
                ? fetchOptionsByIds(link, strings, extraSearchParams)
                : Promise.resolve([] as T[]);

            const additionalOptions = await additionalOptionsPromise;
            return [...objects, ...additionalOptions];
          })()
        : fetchOptionsByIds(
            link,
            values.map((v) =>
              typeof v === "string" ? v : getOptionValue(v as T).toString(),
            ),
            extraSearchParams,
          );

      await resolveInitialOptions
        .then((fetchedOptions) => {
          const seen = new Set<string>();
          const deduped = fetchedOptions.filter((opt) => {
            const key = getOptionValue(opt as T).toString();
            if (seen.has(key)) return false;
            seen.add(key);
            return true;
          });
          setSelectedOptions(deduped);
        })
        .catch((err) => {
          dispatchAsyncState({
            type: "fetchError",
            error:
              err instanceof Error
                ? err.message
                : "Failed to fetch initial values",
          });
        })
        .finally(() => {
          dispatchAsyncState({ type: "fetchEnd" });
        });
    };

    void fetchInitialValues();
  }, [values, link, nestedValues, getOptionValue, extraSearchParams]);

  useEffect(() => {
    const loadOptions = async () => {
      dispatchAsyncState({ type: "fetchStart" });

      await fetchOptions<T>(link, debouncedSearchTerm, page, extraSearchParams)
        .then((response) => {
          dispatchAsyncState({
            type: "fetchSuccess",
            results: response.results,
            append: page !== 1,
            hasMore: !!response.next,
          });
        })
        .catch((err) => {
          console.error(link, err);
          dispatchAsyncState({
            type: "fetchError",
            error:
              err instanceof Error ? err.message : "Failed to fetch options",
          });
        })
        .finally(() => {
          dispatchAsyncState({ type: "fetchEnd" });
        });
    };

    if (open) {
      void loadOptions();
    }
  }, [debouncedSearchTerm, open, page, link, extraSearchParams]);

  useEffect(() => {
    return () => {
      if (animationRef.current !== null) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, []);

  const smoothScroll = useCallback(function smoothScrollFn() {
    if (!commandListRef.current || targetScrollRef.current === null) return;

    const element = commandListRef.current;
    const target = targetScrollRef.current;
    const current = element.scrollTop;

    const distance = target - current;

    if (Math.abs(distance) < 0.5) {
      element.scrollTop = target;
      targetScrollRef.current = null;
      velocityRef.current = 0;
      return;
    }

    const spring = 0.15;

    velocityRef.current += distance * spring;
    velocityRef.current *= 0.8;
    element.scrollTop += velocityRef.current;
    animationRef.current = requestAnimationFrame(smoothScrollFn);
  }, []);

  const handleSelect = useCallback(
    (currentValue: string) => {
      const optionToToggle = options.find(
        (opt) => getOptionValue(opt).toString() === currentValue,
      );

      if (!optionToToggle) return;

      const isAlreadySelected = selectedOptions.some(
        (opt) => getOptionValue(opt).toString() === currentValue,
      );

      const newSelectedOptions = isAlreadySelected
        ? selectedOptions.filter(
            (opt) => getOptionValue(opt).toString() !== currentValue,
          )
        : [
            ...selectedOptions.filter(
              (opt) => getOptionValue(opt).toString() !== currentValue,
            ),
            optionToToggle,
          ];

      setSelectedOptions(newSelectedOptions);

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
      const scrollBuffer = 50;
      const distanceFromBottom =
        target.scrollHeight - (target.scrollTop + target.clientHeight);

      if (
        !loading &&
        hasMore &&
        distanceFromBottom <= scrollBuffer &&
        distanceFromBottom >= 0
      ) {
        setPage((prev) => prev + 1);
      }
    },
    [loading, hasMore],
  );

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

        if ((isAtBottom && isScrollingDown) || (isAtTop && !isScrollingDown)) {
          return;
        }

        e.stopPropagation();
        e.preventDefault();

        const scrollSensitivity = 0.6;
        const deltaY = e.deltaY * scrollSensitivity;

        const currentScroll = commandListRef.current.scrollTop;
        targetScrollRef.current = currentScroll + deltaY;

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
        <PopoverTrigger
          render={
            <Button
              variant="outline"
              role="combobox"
              aria-expanded={open}
              aria-controls={listboxId}
              className={cn(
                "h-auto min-h-7 w-full cursor-auto gap-2 rounded-md border-muted-foreground/20 bg-muted px-2 py-1 font-normal",
                "data-pressed:border-brand data-pressed:ring-4 data-pressed:ring-brand/20 data-pressed:outline-hidden",
                "cursor-pointer justify-between hover:bg-muted-foreground/20 [&_svg]:size-3 [&_svg]:shrink-0",
                "transition-all duration-200 ease-in-out",
                "cursor-default whitespace-nowrap",
                disabled && "cursor-not-allowed opacity-50",
                isInvalid &&
                  "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 hover:border-red-500 hover:bg-red-500/20 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20 data-[state=open]:border-red-500 data-[state=open]:bg-red-500/20 data-[state=open]:ring-red-500/20",
                triggerClassName,
              )}
              disabled={disabled}
            >
              {selectedOptions.length > 0 ? (
                <div className="flex w-full flex-wrap items-center justify-between gap-2 py-1">
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
                          <XIcon />
                        </span>
                      </span>
                    ))}
                    {selectedOptions.length > maxCount && (
                      <Badge
                        className={cn(
                          "h-auto max-h-5 border-foreground/10 bg-transparent text-foreground hover:bg-transparent",
                          multiSelectVariants({ variant: "default" }),
                        )}
                      >
                        {`+ ${selectedOptions.length - maxCount} more`}
                      </Badge>
                    )}
                  </div>

                  <div className="flex items-center">
                    {selectedOptions.length > 0 && !loading && (
                      <div className="flex-rowitems-center flex justify-center gap-0.5">
                        <span
                          title="Clear all"
                          className="mr-1 size-4 cursor-pointer items-center justify-center text-muted-foreground transition-all duration-200 ease-in-out hover:bg-muted-foreground/20 hover:text-foreground"
                          onClick={(event) => {
                            event.stopPropagation();
                            handleClearAll();
                          }}
                        >
                          <XIcon />
                        </span>
                        <Separator
                          orientation="vertical"
                          className="h-full min-h-4 bg-foreground/10"
                        />
                        <span
                          title="Toggle dropdown"
                          className="mr-0.5 size-3 cursor-pointer items-center justify-center text-muted-foreground transition-all duration-200 ease-in-out"
                          onClick={(event) => {
                            event.stopPropagation();
                            setOpen(!open);
                          }}
                        >
                          <ChevronDownIcon
                            className={cn(
                              "ml-1 size-3 shrink-0 cursor-pointer text-muted-foreground transition-all duration-200 ease-in-out",
                              open && "-rotate-180 transform",
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
                    className="mr-0.5 size-3 cursor-pointer items-center justify-center text-muted-foreground transition-all duration-200 ease-in-out"
                    onClick={(event) => {
                      event.stopPropagation();
                      setOpen(!open);
                    }}
                  >
                    <ChevronDownIcon
                      className={cn(
                        "ml-1 size-3 flex-shrink-0 cursor-pointer opacity-50 transition-all duration-200 ease-in-out",
                        open && "-rotate-180 transform",
                      )}
                    />
                  </span>
                </>
              )}
              {loading && (
                <div className="absolute right-7">
                  <Spinner />
                </div>
              )}
            </Button>
          }
        />
        <PopoverContent
          sideOffset={7}
          className={cn("dark w-(--anchor-width) rounded-md p-0", className)}
        >
          <Command shouldFilter={false} className="overflow-hidden">
            <CommandInput
              className="h-7 truncate bg-transparent [&_[cmdk-input]]:h-7"
              placeholder={`Search ${label?.toLowerCase()}...`}
              value={searchTerm}
              onValueChange={setSearchTerm}
            />
            <CommandList
              id={listboxId}
              ref={commandListRef}
              onScroll={handleScrollEnd}
              onWheel={handleWheel}
              className="scrollbar-thin scrollbar-thumb-gray-300 scrollbar-track-transparent max-h-[200px] overflow-y-auto"
            >
              {error && (
                <div className="p-4 text-center text-destructive">{error}</div>
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
                      <CheckIcon className="size-4" />
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
                      className="cursor-pointer font-normal [&_svg]:size-3"
                      key={optionValue}
                      value={optionValue}
                      onSelect={handleSelect}
                    >
                      {renderOption(option)}
                      <CheckIcon
                        className={cn(
                          "ml-auto size-2",
                          isSelected ? "opacity-100" : "opacity-0",
                        )}
                      />
                    </CommandItem>
                  );
                })}
                {loading && (
                  <div className="flex justify-center p-2">
                    <Spinner />
                  </div>
                )}
                {hasMore && !loading && (
                  <div className="p-2 text-center text-xs text-muted-foreground">
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
