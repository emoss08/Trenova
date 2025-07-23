/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
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
import { popoutWindowManager } from "@/hooks/popout-window/popout-window";
import { http } from "@/lib/http-client";
import { cn, toTitleCase } from "@/lib/utils";
import type { LimitOffsetResponse } from "@/types/server";
import { faGhost } from "@fortawesome/pro-duotone-svg-icons";
import { faCheck } from "@fortawesome/pro-regular-svg-icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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

function openPopoutWindow(
  popoutLink: string,
  event: React.MouseEvent<HTMLButtonElement>,
) {
  event.preventDefault();
  event.stopPropagation();

  popoutWindowManager.openWindow(
    popoutLink,
    {},
    {
      modalType: "create",
      width: 800,
      height: 800,
      hideAside: true,
      rememberPosition: true,
    },
  );
}

export function AutocompleteCommandContent<TOption>({
  open,
  link,
  preload,
  label,
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
  popoutLink,
}: {
  open: boolean;
  link: string;
  preload: boolean;
  label?: string;
  clearable: boolean;
  value: string;
  setOpen: (open: boolean) => void;
  noResultsMessage?: string;
  onChange: (...event: any[]) => void;
  getOptionValue: (option: TOption) => string | number;
  renderOption: (option: TOption) => React.ReactNode;
  setSelectedOption: (option: TOption | null) => void;
  onOptionChange?: (option: TOption | null) => void;
  extraSearchParams?: Record<string, string | string[]>;
  popoutLink?: string;
}) {
  const queryClient = useQueryClient();
  const [options, setOptions] = useState<TOption[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearchTerm = useDebounce(searchTerm, preload ? 0 : 300);
  const [hasMore, setHasMore] = useState(false);
  const [page, setPage] = useState(1);
  const commandListRef = useRef<HTMLDivElement>(null);

  // * Animation frame reference for smooth scrolling
  const animationRef = useRef<number | null>(null);
  // * Target scroll position for smooth scrolling
  const targetScrollRef = useRef<number | null>(null);

  const { isLoading, isError } = useQuery({
    queryKey: [
      "autocomplete",
      link,
      debouncedSearchTerm,
      page,
      extraSearchParams,
    ],
    queryFn: async () => {
      const response = await fetchOptions<TOption>(
        link,
        debouncedSearchTerm,
        page,
        extraSearchParams,
      );

      // * Update options state
      setOptions((prev) =>
        page === 1 ? response.results : [...prev, ...response.results],
      );
      setHasMore(!!response.next);

      return response;
    },
    placeholderData: () => {
      return queryClient.getQueryData([
        "autocomplete",
        link,
        debouncedSearchTerm,
        page,
        extraSearchParams,
      ]);
    },
    enabled: open,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
  });

  // * Reset the page when the search term changes
  useEffect(() => {
    setPage(1);
  }, [debouncedSearchTerm]);

  const handleScrollEnd = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      const target = e.target as HTMLDivElement;
      const scrollBuffer = 50; // pixels before bottom to trigger load
      const distanceFromBottom =
        target.scrollHeight - (target.scrollTop + target.clientHeight);

      // * Check if we're near the bottom and not already loading
      if (
        !isLoading &&
        hasMore &&
        distanceFromBottom <= scrollBuffer &&
        distanceFromBottom >= 0 // Prevent overscroll triggering
      ) {
        setPage((prev) => prev + 1);
      }
    },
    [isLoading, hasMore],
  );

  // * Clean up animation frame on unmount
  useEffect(() => {
    return () => {
      if (animationRef.current !== null) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, []);

  // * Smooth scrolling animation function
  const smoothScroll = useCallback(() => {
    if (!commandListRef.current || targetScrollRef.current === null) return;

    const element = commandListRef.current;
    const target = targetScrollRef.current;
    const current = element.scrollTop;

    // * Distance to target
    const distance = target - current;

    // * If we're very close to target, just set it and stop
    if (Math.abs(distance) < 0.5) {
      element.scrollTop = target;
      targetScrollRef.current = null;
      return;
    }

    // * Gentle easing - much less springy than before
    // * This gives a smooth feel without the bounciness
    const easeFactor = 0.25;

    // * Move a percentage of the distance each frame
    const movement = distance * easeFactor;

    // * Apply movement
    element.scrollTop += movement;

    // * Continue animation
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

  // * Handle wheel events with better smooth scrolling
  const handleWheel = useCallback(
    (e: React.WheelEvent) => {
      if (!commandListRef.current) return;

      const { scrollTop, scrollHeight, clientHeight } = commandListRef.current;
      const isScrollingDown = e.deltaY > 0;
      const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1;
      const isAtTop = scrollTop <= 0;

      // * Allow parent scrolling only if we're at the boundaries and trying to scroll beyond
      if ((isAtBottom && isScrollingDown) || (isAtTop && !isScrollingDown)) {
        return; // * Let the event propagate to parent
      }

      // * Otherwise handle the scroll ourselves
      e.stopPropagation();
      e.preventDefault();

      // * Adjust sensitivity - higher is more responsive but less smooth
      const scrollSensitivity = 0.8;

      // * Apply the scroll delta with sensitivity adjustment
      const delta = e.deltaY * scrollSensitivity;

      // * Get current scroll position
      const currentScroll = commandListRef.current.scrollTop;

      // * Set target scroll position with momentum
      // * This creates a smoother feeling without being too springy
      targetScrollRef.current = currentScroll + delta;

      // * Start animation if not already running
      if (animationRef.current === null) {
        animationRef.current = requestAnimationFrame(smoothScroll);
      }
    },
    [smoothScroll],
  );

  // * Add non-passive wheel event listener
  useEffect(() => {
    const commandList = commandListRef.current;
    if (!commandList) return;

    // * Add wheel event listener with { passive: false } option
    const wheelHandler = (e: WheelEvent) => {
      handleWheel(e as unknown as React.WheelEvent);
    };

    // Using non-passive listener to allow preventDefault() for custom scroll behavior
    // This is necessary to prevent parent scrolling while implementing momentum scrolling
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
          placeholder={`Search ${label?.toLowerCase()}...`}
          value={searchTerm}
          onValueChange={setSearchTerm}
        />
      </div>
      <CommandList
        ref={commandListRef}
        onScroll={handleScrollEnd}
        className="max-h-[250px] overflow-y-auto scrollbar-thin scrollbar-thumb-gray-300 scrollbar-track-transparent"
      >
        {isError && (
          <div className="p-4 text-destructive text-center">
            Failed to fetch options
          </div>
        )}
        {!isLoading && options.length === 0 && (
          <div className="flex flex-col items-center p-4 justify-center size-full gap-2">
            <div className="flex items-center justify-center p-4 rounded-full bg-blue-600/20 border border-blue-600/50">
              <Icon icon={faGhost} className="size-10 text-blue-600" />
            </div>

            <CommandEmpty>
              {noResultsMessage ?? `No ${toTitleCase(label ?? "")} found.`}
            </CommandEmpty>
            <span className="text-2xs text-muted-foreground text-center">
              We can&apos;t find any {label?.toLowerCase()} in your
              organization.
            </span>
            {popoutLink && (
              <Button
                size="sm"
                onClick={(event) => openPopoutWindow(popoutLink, event)}
              >
                Add New
              </Button>
            )}
          </div>
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
          {isLoading && (
            <div className="p-2 flex justify-center">
              <PulsatingDots size={1} color="foreground" />
            </div>
          )}
          {hasMore && !isLoading && (
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
          "size-3",
          "absolute right-2 top-1/2 -translate-y-1/2",
          value === getOptionValue(option).toString()
            ? "opacity-100"
            : "opacity-0",
        )}
      />
    </CommandItem>
  );
}
