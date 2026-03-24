import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Spinner } from "@/components/ui/spinner";
import { popoutWindowManager } from "@/hooks/popout-window/popout-window";
import { useDebounce } from "@/hooks/use-debounce";
import { API_BASE_URL } from "@/lib/constants";
import { cn, pluralize, toTitleCase } from "@/lib/utils";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon } from "lucide-react";
import type React from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

export async function fetchOptions<T>(
  link: string,
  inputValue: string,
  page: number,
  initialLimit?: number,
  extraSearchParams?: Record<string, string | string[]>,
): Promise<GenericLimitOffsetResponse<T>> {
  const limit = initialLimit ?? 10;
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

function openPopoutWindow(popoutLink: string, event: React.MouseEvent<HTMLButtonElement>) {
  event.preventDefault();
  event.stopPropagation();

  popoutWindowManager.openWindow(
    popoutLink,
    {},
    {
      panelType: "create",
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
  selectedOption,
  value,
  setOpen,
  noResultsMessage,
  clearable,
  onOptionChange,
  extraSearchParams,
  onChange,
  popoutLink,
  onClear,
  initialLimit = 20,
  listboxId,
}: {
  open: boolean;
  link: string;
  preload: boolean;
  label?: string;
  clearable: boolean;
  value: string | null | undefined;
  setOpen: (open: boolean) => void;
  noResultsMessage?: string;
  onChange: (...event: any[]) => void;
  getOptionValue: (option: TOption) => string | number;
  renderOption: (option: TOption) => React.ReactNode;
  setSelectedOption: (option: TOption | null) => void;
  selectedOption: TOption | null;
  onOptionChange?: (option: TOption | null) => void;
  extraSearchParams?: Record<string, string | string[]>;
  initialLimit?: number;
  popoutLink?: string;
  onClear?: () => void;
  listboxId: string;
}) {
  const queryClient = useQueryClient();
  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearchTerm = useDebounce(searchTerm, preload ? 0 : 300);
  const [page, setPage] = useState(1);
  const commandListRef = useRef<HTMLDivElement>(null);

  const animationRef = useRef<number | null>(null);
  const targetScrollRef = useRef<number | null>(null);
  const isWheelScrollingRef = useRef(false);
  const searchQueryKey = [
    "autocomplete-search",
    link,
    debouncedSearchTerm,
    page,
    extraSearchParams,
    initialLimit,
  ];
  const getSearchQueryKey = useCallback(
    (targetPage: number) => [
      "autocomplete-search",
      link,
      debouncedSearchTerm,
      targetPage,
      extraSearchParams,
      initialLimit,
    ],
    [link, debouncedSearchTerm, extraSearchParams, initialLimit],
  );

  const { isLoading, isError, data } = useQuery({
    queryKey: searchQueryKey,
    queryFn: async () => {
      const response = await fetchOptions<TOption>(
        link,
        debouncedSearchTerm,
        page,
        initialLimit,
        extraSearchParams,
      );
      return response;
    },
    placeholderData: () => queryClient.getQueryData(searchQueryKey),
    enabled: open,
    staleTime: 2 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    retry: 1,
  });
  const hasMore = !!data?.next;
  const options = useMemo(() => {
    const aggregatedOptions: TOption[] = [];

    for (let targetPage = 1; targetPage <= page; targetPage += 1) {
      const cachedPage = queryClient.getQueryData<GenericLimitOffsetResponse<TOption>>(
        getSearchQueryKey(targetPage),
      );
      if (cachedPage?.results?.length) {
        aggregatedOptions.push(...cachedPage.results);
      }
    }

    const currentPageIsCached = queryClient.getQueryData<GenericLimitOffsetResponse<TOption>>(
      getSearchQueryKey(page),
    );
    if (!currentPageIsCached && data?.results?.length) {
      aggregatedOptions.push(...data.results);
    }

    if (value && selectedOption && open) {
      const optionExists = aggregatedOptions.some(
        (opt) => getOptionValue(opt).toString() === value,
      );
      if (!optionExists) {
        return [selectedOption, ...aggregatedOptions];
      }
    }

    return aggregatedOptions;
  }, [data, page, queryClient, value, selectedOption, open, getOptionValue, getSearchQueryKey]);

  useEffect(() => {
    if (!open) {
      setSearchTerm("");
      setPage(1);
    }
  }, [open]);

  const handleScrollEnd = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      const target = e.target as HTMLDivElement;
      const scrollBuffer = 50;
      const distanceFromBottom = target.scrollHeight - (target.scrollTop + target.clientHeight);

      if (!isWheelScrollingRef.current) {
        if (animationRef.current !== null) {
          cancelAnimationFrame(animationRef.current);
          animationRef.current = null;
        }
        targetScrollRef.current = null;
      }

      if (!isLoading && hasMore && distanceFromBottom <= scrollBuffer && distanceFromBottom >= 0) {
        setPage((prev) => prev + 1);
      }
    },
    [isLoading, hasMore],
  );

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
      return;
    }

    const easeFactor = 0.25;

    const movement = distance * easeFactor;

    element.scrollTop += movement;

    animationRef.current = requestAnimationFrame(smoothScrollFn);
  }, []);

  const handleSelect = useCallback(
    (currentValue: string) => {
      const isClearing = clearable && currentValue === value;
      const newValue = isClearing ? null : currentValue;
      const selectedOpt = isClearing
        ? null
        : options.find((opt) => getOptionValue(opt).toString() === currentValue);

      setSelectedOption(selectedOpt || null);
      onChange(newValue);
      if (onOptionChange) {
        onOptionChange(selectedOpt || null);
      }
      if (isClearing && onClear) {
        onClear();
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
      onClear,
    ],
  );

  const smoothScrollRef = useRef(smoothScroll);
  smoothScrollRef.current = smoothScroll;

  const commandListCallbackRef = useCallback((node: HTMLDivElement | null) => {
    commandListRef.current = node;
  }, []);

  useEffect(() => {
    const el = commandListRef.current;
    if (!el) return;

    function handleWheel(e: WheelEvent) {
      if (!commandListRef.current) return;

      const { scrollTop, scrollHeight, clientHeight } = commandListRef.current;
      const isScrollingDown = e.deltaY > 0;
      const isAtBottom = scrollTop + clientHeight >= scrollHeight - 1;
      const isAtTop = scrollTop <= 0;

      if ((isAtBottom && isScrollingDown) || (isAtTop && !isScrollingDown)) {
        return;
      }

      e.stopPropagation();
      e.preventDefault();

      isWheelScrollingRef.current = true;

      const scrollSensitivity = 0.8;
      const delta = e.deltaY * scrollSensitivity;
      const currentScroll = commandListRef.current.scrollTop;

      targetScrollRef.current = currentScroll + delta;

      if (animationRef.current === null) {
        animationRef.current = requestAnimationFrame(smoothScrollRef.current);
      }

      setTimeout(() => {
        isWheelScrollingRef.current = false;
      }, 50);
    }

    el.addEventListener("wheel", handleWheel, { passive: false });
    return () => el.removeEventListener("wheel", handleWheel);
  });

  return (
    <Command shouldFilter={false} className="overflow-hidden">
      <CommandInput
        className="h-7 truncate bg-transparent"
        placeholder={`Search ${label?.toLowerCase()}...`}
        value={searchTerm}
        onValueChange={(nextValue) => {
          setSearchTerm(nextValue);
          setPage(1);
        }}
      />
      <CommandList
        id={listboxId}
        ref={commandListCallbackRef}
        onScroll={handleScrollEnd}
        className="scrollbar-thin scrollbar-thumb-gray-300 scrollbar-track-transparent max-h-[250px] overflow-y-auto"
      >
        {isError && <div className="p-4 text-center text-destructive">Failed to fetch options</div>}
        {!isLoading && data && options.length === 0 && (
          <div className="flex size-full flex-col items-center justify-center gap-2 p-4">
            <CommandEmpty className="p-0">
              {noResultsMessage ??
                `No ${pluralize(toTitleCase(label ?? ""), options.length)} found.`}
            </CommandEmpty>
            <span className="text-center text-2xs text-muted-foreground">
              We can&apos;t find any {label?.toLowerCase()} in your organization.
            </span>
            {popoutLink && (
              <Button size="sm" onClick={(event) => openPopoutWindow(popoutLink, event)}>
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
          {isLoading && options.length === 0 && (
            <div className="flex justify-center p-8">
              <Spinner className="size-4" />
            </div>
          )}
          {isLoading && options.length > 0 && (
            <div className="flex justify-center p-2">
              <Spinner className="size-4" />
            </div>
          )}
          {hasMore && !isLoading && (
            <div className="p-2 text-center text-xs text-muted-foreground">Scroll for more</div>
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
  value: string | null | undefined;
}) {
  const optionValue = getOptionValue(option).toString();
  const isSelected = value === optionValue;

  return (
    <CommandItem
      className="cursor-pointer [&_svg]:size-3"
      key={getOptionValue(option).toString()}
      value={getOptionValue(option).toString()}
      onSelect={handleSelect}
    >
      {renderOption(option)}
      <CheckIcon
        className={cn(
          "size-3",
          "absolute top-1/2 right-2 -translate-y-1/2",
          isSelected ? "opacity-100" : "opacity-0",
        )}
      />
    </CommandItem>
  );
}
