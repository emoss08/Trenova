/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */
import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { useLocationAutoComplete } from "@/hooks/useQueries";
import { cn } from "@/lib/utils";
import { GoogleAutoCompleteResult } from "@/types/location";
import { faGoogle } from "@fortawesome/free-brands-svg-icons";
import { faSpinner } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { debounce } from "lodash-es";
import { AlertTriangle } from "lucide-react";
import * as React from "react";
import { useInteractOutside } from "react-aria";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { ErrorMessage, Input, InputProps } from "../common/fields/input";
import { Label } from "../common/fields/label";

type AutocompleteResultsProps = {
  searchResults: GoogleAutoCompleteResult[] | null;
  onSelectResult: (address: string) => void;
  ref: React.RefObject<HTMLDivElement>;
};

const AutocompleteResults = React.forwardRef<
  HTMLDivElement,
  AutocompleteResultsProps
>(({ searchResults, onSelectResult }, ref) => {
  if (!searchResults || searchResults.length === 0) {
    return (
      <div className="bg-popover border-border p-2 shadow-lg" ref={ref}>
        <p className="text-muted-foreground text-sm">No results found.</p>
      </div>
    );
  }

  return (
    <div
      className="z-100 border-border absolute w-auto rounded-md border shadow-lg"
      ref={ref}
    >
      <Command className="bg-popover">
        <CommandList>
          <CommandGroup>
            {searchResults.map((result) => (
              <CommandItem
                key={result.placeId}
                onSelect={() => onSelectResult(result.address)}
              >
                <div className="flex flex-1 items-center justify-between truncate">
                  <div className="flex-1 truncate text-sm">
                    <p className="font-mono text-sm">{result.name}</p>
                    <p className="text-muted-foreground text-xs">
                      {result.address}
                    </p>
                  </div>
                </div>
              </CommandItem>
            ))}
          </CommandGroup>
        </CommandList>
        <div className="bg-card border-border flex select-none items-center justify-between border-t p-2">
          <p className="text-muted-foreground text-xs">
            {searchResults.length} results
          </p>
          <p className="text-muted-foreground text-xs">
            Powered by <FontAwesomeIcon icon={faGoogle} />
          </p>
        </div>
      </Command>
    </div>
  );
});

type LocationAutoCompleteProps<T extends FieldValues> = Omit<
  InputProps,
  "name"
> & {
  label?: string;
  description?: string;
  ref?: React.ForwardedRef<HTMLInputElement>;
} & UseControllerProps<T>;

export function LocationAutoComplete<T extends FieldValues>({
  ...props
}: LocationAutoCompleteProps<T>) {
  const [showResults, setShowResults] = React.useState<boolean>(false);
  const [inputValue, setInputValue] = React.useState<string>("");
  const popoverRef = React.useRef<HTMLDivElement>(null);
  const { field, fieldState } = useController(props);
  const [debouncedInputValue, setDebouncedInputValue] =
    React.useState<string>("");

  const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = event.target.value;
    setInputValue(newValue);
    field.onChange(newValue);
    setShowResults(newValue.trim().length > 0);
  };

  // Use useEffect to correctly implement the debouncing effect
  React.useEffect(() => {
    const debouncer = debounce(() => setDebouncedInputValue(inputValue), 500);
    debouncer();
    return () => {
      debouncer.cancel();
    };
  }, [inputValue]);

  // TODO: Remove auto complete functionality if the organization does not have a Google API key or allow it.
  const { searchResultError, searchResults, isSearchLoading } =
    useLocationAutoComplete(debouncedInputValue);

  const isFieldError = fieldState.invalid || searchResultError;

  const onSelectResult = (address: string) => {
    field.onChange(address);
    setShowResults(false);
  };

  useInteractOutside({
    ref: popoverRef,
    onInteractOutside: () => {
      setShowResults(false);
    },
  });

  return (
    <>
      {props.label && (
        <Label
          className={cn(
            "text-sm font-medium",
            props.rules?.required && "required",
          )}
        >
          {props.label}
        </Label>
      )}
      <div className="relative">
        {isSearchLoading && (
          <div className="pointer-events-none absolute right-2 mt-2.5 flex items-center pl-3">
            <FontAwesomeIcon icon={faSpinner} className="animate-spin" />
          </div>
        )}
        <Input
          {...field}
          className={cn(
            isSearchLoading && "cursor-wait pr-10",
            isFieldError &&
              "ring-1 ring-inset ring-red-500 placeholder:text-red-500 focus:ring-red-500",
          )}
          onChange={handleInputChange}
          {...props}
        />
        {fieldState.invalid && (
          <>
            <div className="pointer-events-none absolute inset-y-0 right-0 mr-2.5 mt-2.5">
              <AlertTriangle size={15} className="text-red-500" />
            </div>
            <ErrorMessage formError={fieldState.error?.message} />
          </>
        )}
        {props.description && !fieldState.invalid && (
          <p className="text-foreground/70 text-xs">{props.description}</p>
        )}
      </div>
      {showResults && (
        <AutocompleteResults
          searchResults={searchResults as GoogleAutoCompleteResult[]}
          onSelectResult={onSelectResult}
          ref={popoverRef}
        />
      )}
    </>
  );
}
