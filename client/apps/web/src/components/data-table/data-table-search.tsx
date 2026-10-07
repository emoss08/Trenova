"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { HelpCircleIcon, SearchLgIcon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTableSearchSuggestions } from "@trenova/shared/types/data-table";
import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { useEffect, useId, useRef, useState } from "react";

type DataTableSearchProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  suggestions?: DataTableSearchSuggestions;
  shortcut?: string;
};

const SEARCH_DEBOUNCE_MS = 300;

const SEARCH_SYNTAX_EXAMPLES = [
  {
    syntax: "word",
    description: "Match word",
    example: "invoice",
  },
  {
    syntax: '"phrase"',
    description: "Exact phrase match",
    example: '"pending approval"',
  },
  {
    syntax: "word1 OR word2",
    description: "Match either word",
    example: "active OR pending",
  },
  {
    syntax: "-word",
    description: "Exclude word",
    example: "invoice -draft",
  },
  {
    syntax: "word1 word2",
    description: "Match both words (AND)",
    example: "customer order",
  },
];

export default function DataTableSearch({
  value,
  onChange,
  placeholder,
  suggestions,
  shortcut,
}: DataTableSearchProps) {
  const t = useT();
  const listId = useId();
  const anchorRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [focused, setFocused] = useState(false);
  const [isEmpty, setIsEmpty] = useState(value === "");
  const [activeIndex, setActiveIndex] = useState(-1);

  const items = suggestions?.items ?? [];
  const open = !!suggestions && focused && isEmpty && items.length > 0;
  const onSuggestionsOpenChange = suggestions?.onOpenChange;

  useEffect(() => {
    if (inputRef.current && inputRef.current.value !== value) {
      inputRef.current.value = value;
    }
    setIsEmpty(value === "");
  }, [value]);

  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    onSuggestionsOpenChange?.(open);
    if (!open) setActiveIndex(-1);
  }, [open, onSuggestionsOpenChange]);

  useEffect(() => {
    if (!shortcut) return;
    const focusOnShortcut = (event: KeyboardEvent) => {
      if (event.key !== shortcut || event.metaKey || event.ctrlKey || event.altKey) return;
      if (isTypingTarget(event.target) || isWithinDialog(event.target)) return;
      event.preventDefault();
      inputRef.current?.focus();
    };
    window.addEventListener("keydown", focusOnShortcut);
    return () => window.removeEventListener("keydown", focusOnShortcut);
  }, [shortcut]);

  const handleChange = (nextValue: string) => {
    setIsEmpty(nextValue === "");
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    debounceTimerRef.current = setTimeout(() => {
      onChange(nextValue);
    }, SEARCH_DEBOUNCE_MS);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      event.currentTarget.blur();
      return;
    }
    if (!open) return;
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setActiveIndex((index) => (index + 1) % items.length);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((index) => (index <= 0 ? items.length - 1 : index - 1));
    } else if (event.key === "Enter" && activeIndex >= 0) {
      event.preventDefault();
      items[activeIndex]?.onSelect();
    }
  };

  const optionId = (index: number) => `${listId}-option-${index}`;

  return (
    <div ref={anchorRef} className="w-48">
      <Input
        ref={inputRef}
        type="text"
        defaultValue={value}
        role={suggestions ? "combobox" : undefined}
        aria-label={placeholder ?? t("Search")}
        aria-expanded={suggestions ? open : undefined}
        aria-controls={suggestions && open ? listId : undefined}
        aria-autocomplete={suggestions ? "list" : undefined}
        aria-activedescendant={open && activeIndex >= 0 ? optionId(activeIndex) : undefined}
        onChange={(e) => handleChange(e.target.value)}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder ?? t("Search...")}
        className="h-7 text-sm"
        leftElement={<SearchLgIcon className="text-muted-foreground size-3.5 shrink-0" />}
        rightElement={
          <span className="flex items-center gap-1">
            {shortcut && isEmpty && !focused ? <Kbd>{shortcut}</Kbd> : null}
            <SearchSyntaxHelper />
          </span>
        }
      />
      {suggestions && open ? (
        <Popover open>
          <PopoverContent
            anchor={anchorRef}
            align="start"
            initialFocus={false}
            finalFocus={false}
            className="w-60 gap-0 p-1"
          >
            <div
              id={listId}
              role="listbox"
              aria-label={suggestions.title}
              className="flex flex-col"
            >
              <div className="text-muted-foreground px-2 py-1 text-xs" aria-hidden>
                {suggestions.title}
              </div>
              {items.map((item, index) => (
                <div
                  key={item.key}
                  id={optionId(index)}
                  role="option"
                  aria-selected={item.selected ?? false}
                  data-active={index === activeIndex || undefined}
                  className="hover:bg-surface-hover data-active:bg-surface-hover flex h-7 cursor-pointer items-center gap-2 rounded-md px-2 text-sm"
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={item.onSelect}
                >
                  {item.dotClassName ? (
                    <span aria-hidden className={cn("size-1.5 rounded-full", item.dotClassName)} />
                  ) : null}
                  <span className="flex-1">{item.label}</span>
                  {item.count != null ? (
                    <span className="text-muted-foreground font-mono text-xs tabular-nums">
                      {item.count.toLocaleString()}
                    </span>
                  ) : null}
                </div>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      ) : null}
    </div>
  );
}

function SearchSyntaxHelper() {
  const t = useT();

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className="text-muted-foreground hover:text-foreground size-6 cursor-help"
          >
            <HelpCircleIcon className="size-3.5" />
          </Button>
        }
      />
      <PopoverContent className="w-80" align="end">
        <div className="space-y-3">
          <div>
            <h4 className="font-medium">{t("Search syntax")}</h4>
            <p className="text-muted-foreground text-sm">
              {t("Use these patterns to refine your search results.")}
            </p>
          </div>
          <div className="space-y-2">
            {SEARCH_SYNTAX_EXAMPLES.map((item) => (
              <div key={item.syntax} className="grid grid-cols-[100px_1fr] gap-2 text-sm">
                <code className="bg-muted rounded-md px-1.5 py-0.5 font-mono text-xs">
                  {item.syntax}
                </code>
                <div>
                  <p className="text-foreground">{t(item.description)}</p>
                  <p className="text-muted-foreground text-xs">
                    {t("e.g.,")} <code className="font-mono">{item.example}</code>
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
