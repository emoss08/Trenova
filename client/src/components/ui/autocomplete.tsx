import {
  Command,
  CommandGroup,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { useLocationAutoComplete } from "@/hooks/useQueries";
import { cn } from "@/lib/utils";
import { GoogleAutoCompleteResult } from "@/types/location";
import { faSpinner } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { debounce } from "lodash-es";
import * as React from "react";
import { useInteractOutside } from "react-aria";
import {
  FieldValues,
  UseControllerProps,
  useController,
} from "react-hook-form";
import { FieldErrorMessage } from "../common/fields/error-message";
import { Input, InputProps } from "../common/fields/input";
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
      <div className="border-border bg-popover p-2 shadow-lg" ref={ref}>
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
                key={result.placeId || result.name}
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
        <div className="border-border bg-card flex select-none items-center justify-between border-t p-2">
          <p className="text-muted-foreground text-xs">
            {searchResults.length} results
          </p>
          <p className="fill-muted-foreground text-muted-foreground size-4 text-xs">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 488 512"
              className="fill-muted-foreground text-muted-foreground size-4 text-xs"
            >
              <path d="M488 261.8C488 403.3 391.1 504 248 504 110.8 504 0 393.2 0 256S110.8 8 248 8c66.8 0 123 24.5 166.3 64.9l-67.5 64.9C258.5 52.6 94.3 116.6 94.3 256c0 86.5 69.1 156.6 153.7 156.6 98.2 0 135-70.4 140.8-106.9H248v-85.3h236.1c2.3 12.7 3.9 24.9 3.9 41.4z" />
            </svg>
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
  completionAllowed?: boolean;
  ref?: React.ForwardedRef<HTMLInputElement>;
} & UseControllerProps<T>;

export function LocationAutoComplete<T extends FieldValues>({
  completionAllowed = false, // Destructure the new prop
  ...props
}: LocationAutoCompleteProps<T>) {
  const [showResults, setShowResults] = React.useState<boolean>(false);
  const [inputValue, setInputValue] = React.useState<string>("");
  const popoverRef = React.useRef<HTMLDivElement>(null);
  const { field, fieldState } = useController(props);
  const [debouncedInputValue, setDebouncedInputValue] =
    React.useState<string>("");

  const { label, rules } = props;

  const handleInputChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = event.target.value;
    setInputValue(newValue);
    field.onChange(newValue);
    setShowResults(completionAllowed && newValue.trim().length > 0); // Use the flag here
  };

  React.useEffect(() => {
    const debouncer = debounce(() => setDebouncedInputValue(inputValue), 500);
    debouncer();
    return () => {
      debouncer.cancel();
    };
  }, [inputValue]);

  const { searchResultError, searchResults, isSearchLoading } =
    useLocationAutoComplete(completionAllowed ? debouncedInputValue : ""); // Conditional API call based on the flag

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
      <span className="space-x-1">
        {label && <Label className="text-sm font-medium">{label}</Label>}
        {rules?.required && <span className="text-red-500">*</span>}
      </span>
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
          <FieldErrorMessage formError={fieldState.error?.message} />
        )}
        {props.description && !fieldState.invalid && (
          <p className="text-foreground/70 text-xs">{props.description}</p>
        )}
      </div>
      {showResults && completionAllowed && (
        <AutocompleteResults
          searchResults={searchResults as GoogleAutoCompleteResult[]}
          onSelectResult={onSelectResult}
          ref={popoverRef}
        />
      )}
    </>
  );
}
