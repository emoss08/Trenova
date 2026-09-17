import { useT } from "@trenova/shared/i18n/use-t";
import { FieldWrapper } from "@/components/fields/field-components";
import { cityStateLabel } from "@/lib/carrier-sourcing";
import {
  CARRIER_SOURCING_AUTOCOMPLETE_KEY,
  fetchCarrierSourcingAutocomplete,
  type CarrierSourcingSuggestion,
} from "@/lib/graphql/carrier-sourcing";
import { useQuery } from "@tanstack/react-query";
import { Input } from "@trenova/shared/components/ui/input";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { cn } from "@trenova/shared/lib/utils";
import { SearchIcon } from "lucide-react";
import { useId, useState, type KeyboardEvent } from "react";
import { Controller, type Control } from "react-hook-form";
import type { SourcingSearchFormValues } from "./sourcing-schema";

export const AUTOCOMPLETE_MIN_CHARS = 2;
export const AUTOCOMPLETE_LIMIT = 8;
export const AUTOCOMPLETE_DEBOUNCE_MS = 300;

export type SourcingTextAutocompleteProps = {
  control: Control<SourcingSearchFormValues>;
  autocompleteEnabled: boolean;
  onPickSuggestion: (suggestion: CarrierSourcingSuggestion) => void;
};

export function SourcingTextAutocomplete({
  control,
  autocompleteEnabled,
  onPickSuggestion,
}: SourcingTextAutocompleteProps) {
  const t = useT();
  const listboxId = useId();

  return (
    <Controller
      control={control}
      name="text"
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={t("Name, EIN or VIN")}
          description={t(
            "A 9-digit EIN or a 17-character VIN searches by that identifier; anything else matches carrier names.",
          )}
          error={fieldState.error?.message}
        >
          <SuggestionInput
            value={field.value}
            onChange={field.onChange}
            onBlur={field.onBlur}
            name={field.name}
            invalid={fieldState.invalid}
            listboxId={listboxId}
            autocompleteEnabled={autocompleteEnabled}
            onPickSuggestion={onPickSuggestion}
          />
        </FieldWrapper>
      )}
    />
  );
}

type SuggestionInputProps = {
  value: string;
  onChange: (value: string) => void;
  onBlur: () => void;
  name: string;
  invalid: boolean;
  listboxId: string;
  autocompleteEnabled: boolean;
  onPickSuggestion: (suggestion: CarrierSourcingSuggestion) => void;
};

function SuggestionInput({
  value,
  onChange,
  onBlur,
  name,
  invalid,
  listboxId,
  autocompleteEnabled,
  onPickSuggestion,
}: SuggestionInputProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const debounced = useDebounce(value.trim(), AUTOCOMPLETE_DEBOUNCE_MS);
  const queryReady = autocompleteEnabled && debounced.length >= AUTOCOMPLETE_MIN_CHARS;

  const suggestionsQuery = useQuery({
    queryKey: [CARRIER_SOURCING_AUTOCOMPLETE_KEY, debounced, AUTOCOMPLETE_LIMIT],
    queryFn: ({ signal }) =>
      fetchCarrierSourcingAutocomplete(debounced, AUTOCOMPLETE_LIMIT, { signal }),
    enabled: queryReady && open,
    staleTime: 5 * 60_000,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const suggestions = queryReady ? (suggestionsQuery.data ?? []) : [];
  const showList = open && queryReady && (suggestions.length > 0 || suggestionsQuery.isFetching);

  const pick = (suggestion: CarrierSourcingSuggestion) => {
    setOpen(false);
    setActiveIndex(-1);
    onPickSuggestion(suggestion);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (!showList || suggestions.length === 0) {
      if (event.key === "ArrowDown" && queryReady) {
        setOpen(true);
      }
      return;
    }
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        setActiveIndex((index) => (index + 1) % suggestions.length);
        break;
      case "ArrowUp":
        event.preventDefault();
        setActiveIndex((index) => (index <= 0 ? suggestions.length - 1 : index - 1));
        break;
      case "Enter":
        if (activeIndex >= 0 && activeIndex < suggestions.length) {
          event.preventDefault();
          pick(suggestions[activeIndex]);
        } else {
          setOpen(false);
        }
        break;
      case "Escape":
        event.preventDefault();
        setOpen(false);
        setActiveIndex(-1);
        break;
      default:
        break;
    }
  };

  return (
    <div className="relative">
      <Input
        name={name}
        value={value}
        role="combobox"
        aria-expanded={showList}
        aria-controls={listboxId}
        aria-autocomplete="list"
        aria-activedescendant={
          showList && activeIndex >= 0 ? `${listboxId}-option-${activeIndex}` : undefined
        }
        aria-invalid={invalid}
        autoComplete="off"
        placeholder={t("Blue Ridge Freight, 12-3456789 or a VIN")}
        maxLength={200}
        leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
        rightElement={
          suggestionsQuery.isFetching && open ? <Spinner className="mr-1 size-3.5" /> : undefined
        }
        onChange={(event) => {
          onChange(event.target.value);
          setOpen(true);
          setActiveIndex(-1);
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => {
          onBlur();
          setOpen(false);
          setActiveIndex(-1);
        }}
        onKeyDown={handleKeyDown}
      />
      {showList ? (
        <ul
          id={listboxId}
          role="listbox"
          aria-label={t("Carrier suggestions")}
          className="bg-popover text-popover-foreground absolute top-full right-0 left-0 z-50 mt-1 max-h-72 overflow-y-auto rounded-md border p-1 shadow-md"
        >
          {suggestions.length === 0 ? (
            <li className="text-muted-foreground px-2 py-1.5 text-xs" aria-disabled>
              {t("Looking for matches...")}
            </li>
          ) : (
            suggestions.map((suggestion, index) => {
              const location = cityStateLabel(suggestion.city, suggestion.state);
              return (
                <li
                  key={suggestion.dotNumber}
                  id={`${listboxId}-option-${index}`}
                  role="option"
                  aria-selected={index === activeIndex}
                  className={cn(
                    "flex cursor-pointer flex-col rounded-sm px-2 py-1.5 text-sm",
                    index === activeIndex ? "bg-accent" : "hover:bg-accent",
                  )}
                  onMouseDown={(event) => event.preventDefault()}
                  onMouseEnter={() => setActiveIndex(index)}
                  onClick={() => pick(suggestion)}
                >
                  <span className="truncate font-medium">
                    {suggestion.legalName || t("USDOT {0}", suggestion.dotNumber)}
                  </span>
                  <span className="text-muted-foreground truncate text-2xs">
                    {[
                      t("USDOT {0}", suggestion.dotNumber),
                      suggestion.dbaName ? t("DBA {0}", suggestion.dbaName) : null,
                      location,
                    ]
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                </li>
              );
            })
          )}
        </ul>
      ) : null}
    </div>
  );
}
