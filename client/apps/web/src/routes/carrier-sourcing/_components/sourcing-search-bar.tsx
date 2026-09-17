import { useT } from "@trenova/shared/i18n/use-t";
import { cityStateLabel } from "@/lib/carrier-intelligence";
import {
  detectSourcingIntent,
  SOURCING_TEXT_MAX_LENGTH,
  type SourcingIntent,
} from "@/lib/carrier-sourcing";
import {
  CARRIER_SOURCING_AUTOCOMPLETE_KEY,
  fetchCarrierSourcingAutocomplete,
  type CarrierSourcingSuggestion,
} from "@/lib/graphql/carrier-sourcing";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useQuery } from "@tanstack/react-query";
import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "@trenova/shared/components/ui/command";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { cn } from "@trenova/shared/lib/utils";
import { Command as CommandPrimitive } from "cmdk";
import { ScanSearchIcon, SearchIcon, TruckIcon, XIcon } from "lucide-react";
import { useRef, useState, type KeyboardEvent } from "react";

export const AUTOCOMPLETE_MIN_CHARS = 2;
export const AUTOCOMPLETE_LIMIT = 8;
export const AUTOCOMPLETE_DEBOUNCE_MS = 250;

export type SourcingSearchBarProps = {
  value: string;
  onValueChange: (value: string) => void;
  onSubmit: (value: string) => void;
  onClear: () => void;
  onPickSuggestion: (suggestion: CarrierSourcingSuggestion) => void;
  autocompleteEnabled: boolean;
  isBusy: boolean;
};

function useSubmitLabel() {
  const t = useT();
  return (intent: SourcingIntent): string => {
    switch (intent.kind) {
      case "dot":
        return t("Look up USDOT {0}", intent.dotNumber);
      case "mc":
        return t("Look up MC {0}", intent.docketNumber);
      case "ein":
        return t("Search by EIN {0}", intent.text);
      case "vin":
        return t("Search by VIN {0}", intent.text);
      case "name":
        return t("Search carriers for “{0}”", intent.text);
      default:
        return "";
    }
  };
}

export function SourcingSearchBar({
  value,
  onValueChange,
  onSubmit,
  onClear,
  onPickSuggestion,
  autocompleteEnabled,
  isBusy,
}: SourcingSearchBarProps) {
  const t = useT();
  const inputRef = useRef<HTMLInputElement>(null);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const submitLabel = useSubmitLabel();

  const intent = detectSourcingIntent(value);
  const debounced = useDebounce(value.trim(), AUTOCOMPLETE_DEBOUNCE_MS);
  const debouncedIntent = detectSourcingIntent(debounced);
  const open = focused && !dismissed && intent.kind !== "empty";
  const suggestionsReady =
    autocompleteEnabled &&
    open &&
    debouncedIntent.kind === "name" &&
    debounced.length >= AUTOCOMPLETE_MIN_CHARS;

  const suggestionsQuery = useQuery({
    queryKey: [CARRIER_SOURCING_AUTOCOMPLETE_KEY, debounced, AUTOCOMPLETE_LIMIT],
    queryFn: ({ signal }) =>
      fetchCarrierSourcingAutocomplete(debounced, AUTOCOMPLETE_LIMIT, { signal }),
    enabled: suggestionsReady,
    staleTime: 5 * 60_000,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const suggestions =
    suggestionsReady && intent.kind === "name" ? (suggestionsQuery.data ?? []) : [];

  useHotkey(
    "/",
    () => {
      inputRef.current?.focus();
      inputRef.current?.select();
    },
    { ignoreInputs: true, preventDefault: true },
  );

  const submit = () => {
    setDismissed(true);
    onSubmit(value);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      if (open) {
        event.preventDefault();
        setDismissed(true);
      } else if (value !== "") {
        event.preventDefault();
        onClear();
      } else {
        inputRef.current?.blur();
      }
      return;
    }
    if (event.key === "Enter" && !open) {
      event.preventDefault();
      submit();
      return;
    }
    if (event.key === "ArrowDown" && !open && intent.kind !== "empty") {
      setDismissed(false);
    }
  };

  return (
    <Command
      shouldFilter={false}
      loop
      label={t("Search carriers")}
      className="relative h-auto overflow-visible bg-transparent"
    >
      <div className="relative">
        <SearchIcon
          className="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
          aria-hidden
        />
        <CommandPrimitive.Input
          ref={inputRef}
          value={value}
          onValueChange={(next) => {
            onValueChange(next.slice(0, SOURCING_TEXT_MAX_LENGTH));
            setDismissed(false);
          }}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          onKeyDown={handleKeyDown}
          maxLength={SOURCING_TEXT_MAX_LENGTH}
          autoComplete="off"
          spellCheck={false}
          placeholder={t("Search carriers by name, USDOT, MC, EIN or VIN")}
          aria-label={t("Search carriers by name, USDOT, MC, EIN or VIN")}
          className={cn(
            "border-input bg-muted placeholder:text-muted-foreground h-10 w-full rounded-md border pr-20 pl-9 text-sm outline-none",
            "focus-visible:border-brand focus-visible:ring-brand/30 transition-[border-color,box-shadow] duration-200 focus-visible:ring-4",
          )}
        />
        <div className="absolute inset-y-0 right-2 flex items-center gap-1.5">
          {isBusy || suggestionsQuery.isFetching ? (
            <Spinner className="text-muted-foreground size-3.5" aria-label={t("Loading")} />
          ) : null}
          {value !== "" ? (
            <button
              type="button"
              onClick={() => {
                onClear();
                inputRef.current?.focus();
              }}
              aria-label={t("Clear search")}
              className="text-muted-foreground hover:text-foreground flex size-6 items-center justify-center rounded-sm"
            >
              <XIcon className="size-3.5" aria-hidden />
            </button>
          ) : focused ? null : (
            <Kbd aria-hidden>/</Kbd>
          )}
        </div>
      </div>
      {open ? (
        <CommandList
          onMouseDown={(event) => event.preventDefault()}
          className="bg-popover text-popover-foreground absolute top-full right-0 left-0 z-50 mt-1 rounded-md border shadow-md"
        >
          <CommandGroup>
            <CommandItem value="__submit" onSelect={submit}>
              {intent.kind === "dot" || intent.kind === "mc" ? (
                <ScanSearchIcon className="size-3.5" aria-hidden />
              ) : (
                <SearchIcon className="size-3.5" aria-hidden />
              )}
              <span className="truncate">{submitLabel(intent)}</span>
              <CommandShortcut>↵</CommandShortcut>
            </CommandItem>
          </CommandGroup>
          {suggestions.length > 0 ? (
            <CommandGroup heading={t("Carriers")}>
              {suggestions.map((suggestion) => {
                const location = cityStateLabel(suggestion.city, suggestion.state);
                return (
                  <CommandItem
                    key={suggestion.dotNumber}
                    value={`dot:${suggestion.dotNumber}`}
                    onSelect={() => {
                      setDismissed(true);
                      onPickSuggestion(suggestion);
                    }}
                  >
                    <TruckIcon className="size-3.5" aria-hidden />
                    <span className="flex min-w-0 flex-1 items-baseline gap-2">
                      <span className="truncate">
                        {suggestion.legalName || t("USDOT {0}", suggestion.dotNumber)}
                      </span>
                      {suggestion.dbaName ? (
                        <span className="text-muted-foreground truncate text-xs">
                          {suggestion.dbaName}
                        </span>
                      ) : null}
                    </span>
                    <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                      {[t("USDOT {0}", suggestion.dotNumber), location].filter(Boolean).join(" · ")}
                    </span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
          ) : null}
        </CommandList>
      ) : null}
    </Command>
  );
}
