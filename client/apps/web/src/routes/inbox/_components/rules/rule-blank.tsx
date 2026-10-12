import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";

export type BlankOption = {
  key: string;
  label: string;
  /** A second, quieter line: an address, a count. */
  hint?: string;
  leading?: ReactNode;
};

type RuleBlankProps = {
  /** What the blank reads as in the sentence: the chosen value, or the placeholder. */
  children: ReactNode;
  filled: boolean;
  /** Names the blank for assistive technology: "Kind of message". */
  label: string;
  options: readonly BlankOption[];
  selected: string | null;
  onSelect: (key: string) => void;
  /** Searching the options; the caller narrows them itself when it passes `onSearch`. */
  searchable?: boolean;
  search?: string;
  onSearch?: (search: string) => void;
  searchPlaceholder?: string;
  emptyText?: string;
  /** A rule shown dimmed (paused) draws its blanks quieter. */
  quiet?: boolean;
  /** A blank just filled on the add row lands with a small settle. */
  fresh?: boolean;
  disabled?: boolean;
};

/**
 * One blank of a rule's sentence: the bold words, underlined, that open a
 * short list to choose from. An empty blank is the placeholder on a dashed
 * line. The list is a combobox, so it is typed into and walked with the
 * arrow keys like any other.
 */
export function RuleBlank({
  children,
  filled,
  label,
  options,
  selected,
  onSelect,
  searchable = false,
  search,
  onSearch,
  searchPlaceholder,
  emptyText,
  quiet = false,
  fresh = false,
  disabled = false,
}: RuleBlankProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const filtersItself = onSearch !== undefined;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={disabled}
        nativeButton={false}
        render={<span role="button" tabIndex={disabled ? -1 : 0} />}
        className={cn(
          "ui-focus-ring inline cursor-pointer rounded-xs bg-bottom bg-no-repeat pb-0.5 text-left transition-[background-size,color] duration-200 [box-decoration-break:clone]",
          filled
            ? cn(
                "bg-linear-to-r from-border-strong to-border-strong bg-[length:100%_1px] font-medium hover:bg-[length:100%_100%] data-popup-open:bg-[length:100%_100%]",
                quiet ? "text-foreground-muted" : "text-foreground",
                fresh && "animate-confirm inline-block",
              )
            : "text-foreground-subtle bg-[repeating-linear-gradient(90deg,var(--border-strong)_0_4px,transparent_4px_8px)] bg-[length:100%_1.5px] font-normal hover:text-foreground-muted",
          disabled && "cursor-default",
        )}
      >
        <span className="sr-only">{t("{0}:", label)} </span>
        {children}
      </PopoverTrigger>
      <PopoverContent align="start" className="ui-lift-float w-72 gap-0 overflow-hidden p-0">
        <Command shouldFilter={!filtersItself} loop>
          {searchable && (
            <CommandInput
              value={search}
              onValueChange={onSearch}
              placeholder={searchPlaceholder ?? t("Search")}
              aria-label={searchPlaceholder ?? t("Search")}
            />
          )}
          <CommandList className="max-h-72">
            <CommandEmpty>{emptyText ?? t("Nothing matches")}</CommandEmpty>
            <CommandGroup>
              {options.map((option) => (
                <CommandItem
                  key={option.key}
                  value={filtersItself ? option.key : `${option.label} ${option.hint ?? ""}`}
                  onSelect={() => {
                    onSelect(option.key);
                    setOpen(false);
                  }}
                  className="min-h-8"
                >
                  {option.leading}
                  <span className="flex min-w-0 flex-1 flex-col leading-tight">
                    <span className="truncate">{option.label}</span>
                    {option.hint !== undefined && option.hint !== "" && (
                      <span className="text-foreground-subtle truncate text-xs">{option.hint}</span>
                    )}
                  </span>
                  {option.key === selected && (
                    <CheckIcon aria-hidden className="text-foreground size-3.5" strokeWidth={2.4} />
                  )}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
