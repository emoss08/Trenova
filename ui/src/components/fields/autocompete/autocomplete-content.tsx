import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Icon } from "@/components/ui/icons";
import { PulsatingDots } from "@/components/ui/pulsating-dots";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import type { LimitOffsetResponse } from "@/types/server";
import { faCheck } from "@fortawesome/pro-regular-svg-icons";
import { useDebounce } from "@uidotdev/usehooks";
import type React from "react";
import { useCallback, useEffect, useRef, useState } from "react";

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

export function AutocompleteCommandContent<TOption>({
  link,
  preload,
  label,
  loading,
  setLoading,
  getOptionValue,
  renderOption,
  setSelectedOption,
  value,
  setOpen,
  noResultsMessage,
  clearable,
  onOptionChange,
  extraSearchParams,
  onChange,
  error,
  setError,
}: {
  link: string;
  preload: boolean;
  label: string;
  clearable: boolean;
  loading: boolean;
  setLoading: (loading: boolean) => void;
  value: string;
  setOpen: (open: boolean) => void;
  noResultsMessage?: string;
  onChange: (...event: any[]) => void;
  getOptionValue: (option: TOption) => string | number;
  renderOption: (option: TOption) => React.ReactNode;
  setSelectedOption: (option: TOption | null) => void;
  onOptionChange?: (option: TOption | null) => void;
  error: string | null;
  setError: (error: string | null) => void;
  extraSearchParams?: Record<string, string | string[]>;
}) {
  const [options, setOptions] = useState<TOption[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearchTerm = useDebounce(searchTerm, preload ? 0 : 300);
  const [hasMore, setHasMore] = useState(false);
  const [page, setPage] = useState(1);
  const commandListRef = useRef<HTMLDivElement>(null);

  // Animation frame reference for smooth scrolling
  const animationRef = useRef<number | null>(null);
  // Target scroll position for smooth scrolling
  const targetScrollRef = useRef<number | null>(null);
  // Scroll acceleration and velocity tracking
  const velocityRef = useRef(0);
  // Last wheel event timestamp for inertia calculation
  const lastWheelTimeRef = useRef(0);

  // Memoize the load options function
  const loadOptionsFn = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetchOptions<TOption>(
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
  }, [
    debouncedSearchTerm,
    page,
    link,
    extraSearchParams,
    setLoading,
    setError,
  ]);

  // Fetch options based on search term
  useEffect(() => {
    loadOptionsFn();
  }, [loadOptionsFn]);

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

    // Gentle easing - much less springy than before
    // This gives a smooth feel without the bounciness
    const easeFactor = 0.25;

    // Move a percentage of the distance each frame
    const movement = distance * easeFactor;

    // Apply movement
    element.scrollTop += movement;

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
    [
      value,
      onChange,
      onOptionChange,
      clearable,
      options,
      getOptionValue,
      setSelectedOption,
      setOpen,
    ],
  );

  // Handle wheel events with better smooth scrolling
  const handleWheel = useCallback(
    (e: React.WheelEvent) => {
      if (!commandListRef.current) return;

      const { scrollTop, scrollHeight, clientHeight } = commandListRef.current;
      const isScrollingDown = e.deltaY > 0;
      const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1;
      const isAtTop = scrollTop <= 0;

      // Allow parent scrolling only if we're at the boundaries and trying to scroll beyond
      if ((isAtBottom && isScrollingDown) || (isAtTop && !isScrollingDown)) {
        return; // Let the event propagate to parent
      }

      // Otherwise handle the scroll ourselves
      e.stopPropagation();
      e.preventDefault();

      // Calculate scroll speed with gentle acceleration/deceleration
      const now = performance.now();
      lastWheelTimeRef.current = now;

      // Adjust sensitivity - higher is more responsive but less smooth
      const scrollSensitivity = 0.8;

      // Apply the scroll delta with sensitivity adjustment
      const delta = e.deltaY * scrollSensitivity;

      // Get current scroll position
      const currentScroll = commandListRef.current.scrollTop;

      // Set target scroll position with momentum
      // This creates a smoother feeling without being too springy
      targetScrollRef.current = currentScroll + delta;

      // Start animation if not already running
      if (animationRef.current === null) {
        animationRef.current = requestAnimationFrame(smoothScroll);
      }
    },
    [smoothScroll],
  );

  // Add non-passive wheel event listener
  useEffect(() => {
    const commandList = commandListRef.current;
    if (!commandList) return;

    // Add wheel event listener with { passive: false } option
    const wheelHandler = (e: WheelEvent) => {
      handleWheel(e as unknown as React.WheelEvent);
    };

    commandList.addEventListener("wheel", wheelHandler, { passive: false });

    return () => {
      commandList.removeEventListener("wheel", wheelHandler);
      if (animationRef.current !== null) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [handleWheel]);

  return (
    <Command shouldFilter={false} className="overflow-hidden">
      <div className="w-full">
        <CommandInput
          className="bg-transparent h-7 truncate"
          placeholder={`Search ${label.toLowerCase()}...`}
          value={searchTerm}
          onValueChange={setSearchTerm}
        />
      </div>
      <CommandList
        ref={commandListRef}
        onScroll={handleScrollEnd}
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
            <AutocompleteCommandOption
              key={getOptionValue(option).toString()}
              option={option}
              getOptionValue={getOptionValue}
              renderOption={renderOption}
              handleSelect={handleSelect}
              value={value}
            />
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
  );
}

export function AutocompleteCommandOption<TOption>({
  option,
  getOptionValue,
  renderOption,
  handleSelect,
  value,
}: {
  option: TOption;
  getOptionValue: (option: TOption) => string | number;
  renderOption: (option: TOption) => React.ReactNode;
  handleSelect: (value: string) => void;
  value: string;
}) {
  return (
    <CommandItem
      className="[&_svg]:size-3 cursor-pointer"
      key={getOptionValue(option).toString()}
      value={getOptionValue(option).toString()}
      onSelect={handleSelect}
    >
      {renderOption(option)}
      <Icon
        icon={faCheck}
        className={cn(
          "ml-auto size-3",
          value === getOptionValue(option).toString()
            ? "opacity-100"
            : "opacity-0",
        )}
      />
    </CommandItem>
  );
}
