import { Button } from "@trenova/shared/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { popoutWindowManager } from "@/hooks/popout-window/popout-window";
import { useDebounce } from "@/hooks/use-debounce";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import {
  fetchGraphQLSelectOptions,
  selectOptionFiltersFromSearchParams,
  type GraphQLSelectOptionsConfig,
} from "@/lib/graphql/select-options";
import { cn, pluralize, toTitleCase } from "@trenova/shared/lib/utils";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { CheckIcon } from "lucide-react";
import type React from "react";
import { useCallback, useMemo, useState } from "react";

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
      if (value === undefined || value === null) return;
      if (Array.isArray(value)) {
        value.forEach((entry) => fetchURL.searchParams.append(key, entry));
      } else {
        fetchURL.searchParams.set(key, value);
      }
    });
  }

  const response = await fetch(fetchURL.href, {
    credentials: "include",
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch options (${response.status})`);
  }

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
  graphql,
  onChange,
  popoutLink,
  onClear,
  initialLimit = 20,
  listboxId,
  filterOption,
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
  graphql?: GraphQLSelectOptionsConfig;
  initialLimit?: number;
  popoutLink?: string;
  onClear?: () => void;
  listboxId: string;
  filterOption?: (option: TOption) => boolean;
}) {
  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearchTerm = useDebounce(searchTerm, preload ? 0 : 300);

  const { data, isLoading, isError, isFetching, isFetchingNextPage, hasNextPage, fetchNextPage, refetch } =
    useInfiniteQuery({
      queryKey: [
        "autocomplete-search",
        link,
        debouncedSearchTerm,
        extraSearchParams,
        graphql,
        initialLimit,
      ],
      queryFn: async ({ pageParam }) => {
        if (graphql) {
          return (await fetchGraphQLSelectOptions({
            resource: graphql.resource,
            query: debouncedSearchTerm,
            page: pageParam,
            initialLimit,
            filters: {
              ...selectOptionFiltersFromSearchParams(extraSearchParams),
              ...graphql.filters,
            },
          })) as GenericLimitOffsetResponse<TOption>;
        }

        return fetchOptions<TOption>(
          link,
          debouncedSearchTerm,
          pageParam,
          initialLimit,
          extraSearchParams,
        );
      },
      initialPageParam: 1,
      getNextPageParam: (lastPage, allPages) => (lastPage.next ? allPages.length + 1 : undefined),
      placeholderData: keepPreviousData,
      enabled: open,
      staleTime: 2 * 60 * 1000,
      gcTime: 10 * 60 * 1000,
      refetchOnMount: false,
      refetchOnWindowFocus: false,
      retry: 1,
    });

  const options = useMemo(() => {
    const aggregatedOptions = data?.pages.flatMap((page) => page.results ?? []) ?? [];
    const filteredOptions = filterOption
      ? aggregatedOptions.filter(filterOption)
      : aggregatedOptions;

    if (value && selectedOption) {
      const optionExists = filteredOptions.some((opt) => getOptionValue(opt).toString() === value);
      if (!optionExists) {
        return [selectedOption, ...filteredOptions];
      }
    }

    return filteredOptions;
  }, [data, filterOption, value, selectedOption, getOptionValue]);

  const handleScroll = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      const target = e.currentTarget;
      const scrollBuffer = 50;
      const distanceFromBottom = target.scrollHeight - (target.scrollTop + target.clientHeight);

      if (distanceFromBottom <= scrollBuffer && hasNextPage && !isFetching) {
        void fetchNextPage();
      }
    },
    [hasNextPage, isFetching, fetchNextPage],
  );

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

  return (
    <Command shouldFilter={false} className="overflow-hidden">
      <CommandInput
        className="h-7 truncate bg-transparent"
        placeholder={label ? `Search ${label.toLowerCase()}...` : "Search..."}
        value={searchTerm}
        onValueChange={setSearchTerm}
      />
      <CommandList
        id={listboxId}
        onScroll={handleScroll}
        className="max-h-62.5 scrollbar-thin scrollbar-thumb-gray-300 scrollbar-track-transparent overflow-y-auto overscroll-contain"
      >
        {isError && (
          <div className="flex flex-col items-center gap-2 p-4">
            <p className="text-center text-xs text-destructive">Failed to load options.</p>
            <Button size="sm" variant="outline" onClick={() => refetch()}>
              Retry
            </Button>
          </div>
        )}
        {!isError && !isFetching && data && options.length === 0 && (
          <div className="flex size-full flex-col items-center justify-center gap-2 p-4">
            <CommandEmpty className="p-0 text-center">
              {noResultsMessage ??
                `No ${pluralize(toTitleCase(label ?? ""), options.length)} found.`}
            </CommandEmpty>
            <span className="text-center text-2xs text-muted-foreground">
              We can&apos;t find any {label ? label.toLowerCase() : "results"} in your organization.
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
          {(isFetchingNextPage || (isFetching && !isLoading && options.length > 0)) && (
            <div className="flex justify-center p-2">
              <Spinner className="size-4" />
            </div>
          )}
          {hasNextPage && !isFetchingNextPage && (
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
      value={optionValue}
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
