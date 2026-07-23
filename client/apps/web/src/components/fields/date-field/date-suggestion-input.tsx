import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { cn } from "@trenova/shared/lib/utils";
import { XIcon } from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
  useImperativeHandle,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

export type Suggestion = {
  date: Date;
  inputString: string;
};

function buildSuggestions(
  inputValue: string,
  defaults: string[],
  parseInput: (text: string) => Date | null,
): Suggestion[] {
  const toSuggestion = (inputString: string): Suggestion | null => {
    const date = parseInput(inputString);
    return date ? { date, inputString } : null;
  };
  const isSuggestion = (s: Suggestion | null): s is Suggestion => s !== null;

  const trimmed = inputValue.trim();
  if (!trimmed) {
    return defaults.map(toSuggestion).filter(isSuggestion);
  }

  const matchingDefaults = defaults.filter((text) =>
    text.toLowerCase().includes(trimmed.toLowerCase()),
  );
  if (matchingDefaults.length) {
    return matchingDefaults.map(toSuggestion).filter(isSuggestion);
  }

  return [toSuggestion(trimmed)].filter(isSuggestion);
}

export interface DateSuggestionInputProps
  extends Omit<React.ComponentProps<"input">, "value" | "defaultValue" | "onChange"> {
  value: Date | undefined;
  onValueChange: (date: Date | undefined) => void;
  formatValue: (date: Date) => string;
  parseInput: (text: string) => Date | null;
  defaultSuggestions: string[];
  picker?: React.ReactNode;
  isInvalid?: boolean;
  clearable?: boolean;
}

export function DateSuggestionInput({
  value,
  onValueChange,
  formatValue,
  parseInput,
  defaultSuggestions,
  picker,
  isInvalid,
  clearable,
  placeholder,
  disabled,
  readOnly,
  className,
  ref,
  onFocus,
  onBlur,
  onClick,
  onKeyDown,
  ...props
}: DateSuggestionInputProps) {
  const listboxId = useId();
  const [text, setText] = useState("");
  const [focused, setFocused] = useState(false);
  const [open, setOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  useImperativeHandle(ref, () => inputRef.current!);

  const isLocked = disabled || readOnly;
  const committedText = useMemo(() => (value ? formatValue(value) : ""), [value, formatValue]);
  const displayValue = focused ? text : committedText;

  const suggestions = useMemo(
    () => buildSuggestions(text, defaultSuggestions, parseInput),
    [text, defaultSuggestions, parseInput],
  );
  const activeIndex = suggestions.length ? Math.min(selectedIndex, suggestions.length - 1) : 0;
  const dropdownVisible = open && (suggestions.length > 0 || text.trim().length > 0);
  const showClear = !!clearable && !isLocked && displayValue.length > 0;

  function openDropdown() {
    setOpen(true);
  }

  function closeDropdown() {
    setSelectedIndex(0);
    setOpen(false);
  }

  function commitDate(date: Date | undefined) {
    setText(date ? formatValue(date) : "");
    onValueChange(date);
  }

  function commitTypedText() {
    const trimmed = text.trim();
    if (!trimmed) {
      if (value) {
        onValueChange(undefined);
      }
      return;
    }
    if (trimmed === committedText) return;
    const parsed = parseInput(trimmed);
    if (parsed) {
      commitDate(parsed);
    }
  }

  function selectSuggestion(suggestion: Suggestion) {
    commitDate(suggestion.date);
    closeDropdown();
  }

  function handleClear() {
    setText("");
    setSelectedIndex(0);
    if (value) {
      onValueChange(undefined);
    }
  }

  function handleFocus(e: React.FocusEvent<HTMLInputElement>) {
    onFocus?.(e);
    if (isLocked) return;
    setFocused(true);
    setText(committedText);
    setSelectedIndex(0);
    openDropdown();
  }

  function handleBlur(e: React.FocusEvent<HTMLInputElement>) {
    setFocused(false);
    closeDropdown();
    if (!isLocked) {
      commitTypedText();
    }
    onBlur?.(e);
  }

  function handleClick(e: React.MouseEvent<HTMLInputElement>) {
    onClick?.(e);
    if (isLocked || open) return;
    openDropdown();
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const next = e.target.value;
    setText(next);
    setSelectedIndex(0);
    openDropdown();
    if (!next.trim() && value) {
      onValueChange(undefined);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    onKeyDown?.(e);
    if (isLocked) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (!open) {
        openDropdown();
        return;
      }
      setSelectedIndex(Math.min(activeIndex + 1, Math.max(suggestions.length - 1, 0)));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex(Math.max(activeIndex - 1, 0));
    } else if (e.key === "Enter") {
      if (open && suggestions.length > 0) {
        e.preventDefault();
        selectSuggestion(suggestions[activeIndex]);
      }
    } else if (e.key === "Escape") {
      if (open) {
        e.stopPropagation();
        setText(committedText);
        closeDropdown();
      }
    }
  }

  const updateDropdownPosition = useCallback(() => {
    if (!containerRef.current || !dropdownRef.current) return;
    const rect = containerRef.current.getBoundingClientRect();
    const dropdownHeight = dropdownRef.current.offsetHeight;
    const spaceBelow = window.innerHeight - rect.bottom;
    const spaceAbove = rect.top;
    const placeAbove = spaceBelow < dropdownHeight + 8 && spaceAbove > spaceBelow;

    dropdownRef.current.style.top = placeAbove
      ? `${rect.top - dropdownHeight - 4}px`
      : `${rect.bottom + 4}px`;
    dropdownRef.current.style.left = `${rect.left}px`;
    dropdownRef.current.style.width = `${rect.width}px`;
  }, []);

  useLayoutEffect(() => {
    if (dropdownVisible) {
      updateDropdownPosition();
    }
  }, [dropdownVisible, suggestions.length, updateDropdownPosition]);

  useEffect(() => {
    if (!dropdownVisible) return;

    window.addEventListener("resize", updateDropdownPosition);
    window.addEventListener("scroll", updateDropdownPosition, true);

    return () => {
      window.removeEventListener("resize", updateDropdownPosition);
      window.removeEventListener("scroll", updateDropdownPosition, true);
    };
  }, [dropdownVisible, updateDropdownPosition]);

  useEffect(() => {
    if (!dropdownVisible) return;
    document
      .getElementById(`${listboxId}-option-${activeIndex}`)
      ?.scrollIntoView({ block: "nearest" });
  }, [activeIndex, dropdownVisible, listboxId]);

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <Input
          placeholder={placeholder ?? "Tomorrow"}
          autoComplete="off"
          spellCheck={false}
          {...props}
          ref={inputRef}
          type="text"
          role="combobox"
          aria-expanded={dropdownVisible}
          aria-controls={listboxId}
          aria-autocomplete="list"
          aria-activedescendant={
            dropdownVisible && suggestions.length > 0
              ? `${listboxId}-option-${activeIndex}`
              : undefined
          }
          aria-invalid={isInvalid}
          disabled={disabled}
          readOnly={readOnly}
          className={cn("pr-8", showClear && "pr-14", className)}
          value={displayValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={handleFocus}
          onBlur={handleBlur}
          onClick={handleClick}
        />
        {showClear && (
          <Button
            type="button"
            size="icon"
            variant="ghost"
            onMouseDown={(e) => e.preventDefault()}
            onClick={handleClear}
            className="absolute top-1/2 right-8 size-5 -translate-y-1/2 text-muted-foreground [&>svg]:size-3"
          >
            <span className="sr-only">Clear date</span>
            <XIcon className="size-4" />
          </Button>
        )}
        {picker}
      </div>

      {dropdownVisible &&
        createPortal(
          <div
            ref={dropdownRef}
            className="fixed z-[9999] animate-in rounded-md border bg-popover p-0 shadow-md fade-in-0"
            tabIndex={-1}
          >
            {suggestions.length > 0 ? (
              <ul
                id={listboxId}
                role="listbox"
                aria-label="Date suggestions"
                className="max-h-56 overflow-auto p-1"
              >
                {suggestions.map((suggestion, index) => (
                  <li
                    key={suggestion.inputString}
                    id={`${listboxId}-option-${index}`}
                    role="option"
                    aria-selected={activeIndex === index}
                    className={cn(
                      "flex cursor-pointer items-center justify-between gap-1 rounded-sm px-3 py-1.5 text-xs",
                      index === activeIndex && "bg-muted text-accent-foreground",
                    )}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => selectSuggestion(suggestion)}
                    onMouseEnter={() => setSelectedIndex(index)}
                  >
                    <span className="xs:w-auto w-[110px] truncate">{suggestion.inputString}</span>
                    <span className="shrink-0 text-xs text-muted-foreground">
                      {formatValue(suggestion.date)}
                    </span>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="px-3 py-1.5 text-xs text-muted-foreground">
                No matching date. Try &quot;t+2&quot;, &quot;next friday&quot;, or &quot;07/15&quot;.
              </p>
            )}
          </div>,
          document.body,
        )}
    </div>
  );
}
