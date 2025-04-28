import { Button } from "@/components/ui/button";
import { PulsatingDots } from "@/components/ui/pulsating-dots";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import { ChevronDownIcon, Cross2Icon } from "@radix-ui/react-icons";
import { useQuery } from "@tanstack/react-query";
import type React from "react";

type AutocompleteTriggerProps<TOption> = {
  open: boolean;
  disabled: boolean;
  triggerClassName: string | undefined;
  clearable: boolean;
  value: string;
  selectedOption: TOption | null;
  getDisplayValue: (option: TOption) => React.ReactNode;
  placeholder: string;
  handleClear: () => void;
  isInvalid?: boolean;
  setSelectedOption: (option: TOption | null) => void;
  link: string;
} & React.ComponentProps<"button">;

async function fetchOptionById<T>(
  link: string,
  id: string | number,
): Promise<T> {
  const url = link.endsWith("/") ? `${link}${id}/` : `${link}/${id}/`;

  const { data } = await http.get<T>(url);
  return data;
}

export function AutocompleteTrigger<TOption>({
  open,
  disabled,
  isInvalid,
  triggerClassName,
  clearable,
  value,
  selectedOption,
  getDisplayValue,
  placeholder,
  handleClear,
  setSelectedOption,
  link,
  ...props
}: AutocompleteTriggerProps<TOption>) {
  const { isLoading, isError } = useQuery({
    queryKey: ["autocomplete-item", link, value],
    queryFn: async () => {
      const option = await fetchOptionById<TOption>(link, value);

      // * Set the selected option
      setSelectedOption(option);

      return option;
    },
    enabled: !!value && !selectedOption,
    retry: 1,
    staleTime: 0, // Always consider data stale
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
  });

  return (
    <Button
      type="button"
      variant="outline"
      role="combobox"
      aria-expanded={open}
      className={cn(
        "w-full font-normal gap-2 rounded border-muted-foreground/20 text-xs bg-muted px-1.5 data-[state=open]:border-blue-600 data-[state=open]:outline-hidden data-[state=open]:ring-4 data-[state=open]:ring-blue-600/20",
        "[&_svg]:size-3 justify-between hover:bg-muted",
        "transition-[border-color,box-shadow] duration-200 ease-in-out",
        disabled && "opacity-50 cursor-not-allowed",
        isInvalid &&
          "border-red-500 bg-red-500/20 ring-0 ring-red-500 placeholder:text-red-500 focus:outline-hidden focus-visible:border-red-600 focus-visible:ring-4 focus-visible:ring-red-400/20 hover:border-red-500 hover:bg-red-500/20 data-[state=open]:border-red-500 data-[state=open]:bg-red-500/20 data-[state=open]:ring-red-500/20",
        triggerClassName,
      )}
      disabled={disabled}
      {...props}
    >
      <AutocompleteInputInner
        selectedOption={selectedOption}
        getDisplayValue={getDisplayValue}
        isInvalid={isInvalid || isError}
        placeholder={placeholder}
      />
      <AutocompleteInputActions
        clearable={clearable}
        value={value}
        handleClear={handleClear}
        loading={isLoading}
        open={open}
      />
    </Button>
  );
}

export function AutocompleteInputInner<TOption>({
  selectedOption,
  getDisplayValue,
  placeholder,
  isInvalid,
}: {
  selectedOption: TOption | null;
  getDisplayValue: (option: TOption) => React.ReactNode;
  placeholder: string;
  isInvalid?: boolean;
}) {
  return selectedOption ? (
    getDisplayValue(selectedOption)
  ) : (
    <p className={cn("text-muted-foreground", isInvalid && "text-red-500")}>
      {placeholder}
    </p>
  );
}

export function AutocompleteInputActions({
  clearable,
  value,
  handleClear,
  loading,
  open,
}: {
  clearable: boolean;
  value: string;
  handleClear: () => void;
  loading: boolean;
  open: boolean;
}) {
  return (
    <div className="flex items-center gap-1 ml-auto">
      {clearable && value && (
        <span
          onClick={(e) => {
            e.stopPropagation();
            handleClear();
          }}
          className="[&>svg]:size-3 size-5 rounded-md flex items-center justify-center hover:bg-muted-foreground/30 text-muted-foreground hover:text-foreground transition-colors duration-200 ease-in-out cursor-pointer"
        >
          <span className="sr-only">Clear</span>
          <Cross2Icon className="size-4" />
        </span>
      )}
      {loading ? (
        <div className="mr-1">
          <PulsatingDots size={1} color="foreground" />
        </div>
      ) : (
        <ChevronDownIcon
          className={cn(
            "opacity-50 size-7 duration-200 ease-in-out transition-all",
            open && "-rotate-180",
          )}
        />
      )}
    </div>
  );
}
