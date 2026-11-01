import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandItemCheck,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { popoutWindowManager } from "@/hooks/popout-window/popout-window";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import {
  fetchGraphQLSelectOptions,
  selectOptionFiltersFromSearchParams,
  type GraphQLSelectOptionsConfig,
} from "@/lib/graphql/select-options";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import type React from "react";
import { useCallback, useMemo, useState } from "react";

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
  createOption,
}: {
  open: boolean;
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
  graphql: GraphQLSelectOptionsConfig;
  initialLimit?: number;
  popoutLink?: string;
  onClear?: () => void;
  listboxId: string;
  filterOption?: (option: TOption) => boolean;
  createOption?: (text: string) => TOption;
}) {
  const t = useT();

  const [searchTerm, setSearchTerm] = useState("");
  const debouncedSearchTerm = useDebounce(searchTerm, preload ? 0 : 300);

  const {
    data,
    isLoading,
    isError,
    isFetching,
    isPlaceholderData,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
    refetch,
  } = useInfiniteQuery({
    queryKey: [
      "autocomplete-search",
      debouncedSearchTerm,
      extraSearchParams,
      graphql,
      initialLimit,
    ],
    queryFn: async ({ pageParam, signal }) =>
      (await fetchGraphQLSelectOptions(
        {
          resource: graphql.resource,
          query: debouncedSearchTerm,
          page: pageParam,
          initialLimit,
          filters: {
            ...selectOptionFiltersFromSearchParams(extraSearchParams),
            ...graphql.filters,
          },
        },
        { signal },
      )) as GenericLimitOffsetResponse<TOption>,
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

  const typedText = debouncedSearchTerm.trim();
  const createdOption = useMemo(() => {
    if (!createOption || !typedText || !data || isPlaceholderData) return null;
    const candidate = createOption(typedText);
    const candidateValue = getOptionValue(candidate).toString().toLowerCase();
    const taken = options.some(
      (option) => getOptionValue(option).toString().toLowerCase() === candidateValue,
    );
    return taken ? null : candidate;
  }, [createOption, typedText, data, isPlaceholderData, options, getOptionValue]);

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
        : (options.find((opt) => getOptionValue(opt).toString() === currentValue) ??
          (createdOption && getOptionValue(createdOption).toString() === currentValue
            ? createdOption
            : undefined));

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
      createdOption,
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
        placeholder={label ? t("Search {0}...", label.toLowerCase()) : t("Search...")}
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
            <p className="text-destructive text-center text-xs">{t("Failed to load options.")}</p>
            <Button size="sm" variant="outline" onClick={() => refetch()}>
              {t("Retry")}
            </Button>
          </div>
        )}
        {!isError && !isFetching && data && options.length === 0 && !createdOption && (
          <div className="flex size-full flex-col items-center justify-center gap-2 p-4">
            <CommandEmpty className="p-0 text-center">
              {noResultsMessage ?? t("No results found.")}
            </CommandEmpty>
            <span className="text-2xs text-muted-foreground text-center">
              {label
                ? t("We can't find any {0} in your organization.", label.toLowerCase())
                : t("We can't find any results in your organization.")}
            </span>
            {popoutLink && (
              <Button size="sm" onClick={(event) => openPopoutWindow(popoutLink, event)}>
                {t("Add new")}
              </Button>
            )}
          </div>
        )}
        <CommandGroup>
          {createdOption && (
            <CommandItem
              className="cursor-pointer"
              value={getOptionValue(createdOption).toString()}
              onSelect={handleSelect}
            >
              <PlusIcon className="text-muted-foreground size-3.5 shrink-0" />
              <span className="min-w-0 flex-1 truncate">{t('Use "{0}"', typedText)}</span>
            </CommandItem>
          )}
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
            <div className="text-muted-foreground p-2 text-center text-xs">
              {t("Scroll for more")}
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
  value: string | null | undefined;
}) {
  const optionValue = getOptionValue(option).toString();
  const isSelected = value === optionValue;

  return (
    <CommandItem className="cursor-pointer" value={optionValue} onSelect={handleSelect}>
      <div className="min-w-0 flex-1 overflow-hidden">{renderOption(option)}</div>
      <CommandItemCheck checked={isSelected} />
    </CommandItem>
  );
}
