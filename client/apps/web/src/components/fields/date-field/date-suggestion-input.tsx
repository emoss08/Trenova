import { autoUpdate, flip, hide, offset, size, useFloating } from "@floating-ui/react-dom";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useId, useImperativeHandle, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

export type Suggestion = {
  date: Date;
  inputString: string;
};

function buildSuggestions(
  query: string,
  defaults: string[],
  parseInput: (text: string) => Date | null,
): Suggestion[] {
  const toSuggestion = (inputString: string): Suggestion | null => {
    const date = parseInput(inputString);
    return date ? { date, inputString } : null;
  };
  const isSuggestion = (s: Suggestion | null): s is Suggestion => s !== null;

  const trimmed = query.trim();
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

export interface DateSuggestionInputProps extends Omit<
  React.ComponentProps<"input">,
  "value" | "defaultValue" | "onChange"
> {
  value: Date | undefined;
  onValueChange: (date: Date | undefined) => void;
  formatValue: (date: Date) => string;
  parseInput: (text: string) => Date | null;
  defaultSuggestions: string[];
  picker?: React.ReactNode;
  isInvalid?: boolean;
  clearable?: boolean;
}

/**
 * A date typed in place ("t+2", "next friday", "07/15") with shortcuts offered beneath it. The list
 * is positioned by floating-ui, so it follows the field through every scrolling ancestor, flips
 * when there is no room below, and hides while the field is scrolled out of view. Text that is not
 * a date stays where it was typed and is marked, rather than being swapped back for the old value.
 * The list opens on a click, on typing or on the down arrow, never on focus alone, so a dialog that
 * focuses its first field does not open it.
 */
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
  const t = useT();

  const listboxId = useId();
  const hintId = useId();
  const [text, setText] = useState("");
  const [edited, setEdited] = useState(false);
  const [rejected, setRejected] = useState<string | null>(null);
  const [focused, setFocused] = useState(false);
  const [open, setOpen] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);

  const inputRef = useRef<HTMLInputElement>(null);
  useImperativeHandle(ref, () => inputRef.current!, []);

  const isLocked = disabled || readOnly;
  const committedText = useMemo(() => (value ? formatValue(value) : ""), [value, formatValue]);
  const [shownCommitted, setShownCommitted] = useState(committedText);
  if (committedText !== shownCommitted) {
    setShownCommitted(committedText);
    if (committedText) setRejected(null);
  }
  const displayValue = focused ? text : (rejected ?? committedText);

  const suggestions = useMemo(
    () => buildSuggestions(edited ? text : "", defaultSuggestions, parseInput),
    [edited, text, defaultSuggestions, parseInput],
  );
  const activeIndex = suggestions.length ? Math.min(selectedIndex, suggestions.length - 1) : 0;
  const dropdownVisible = open && (suggestions.length > 0 || text.trim().length > 0);
  const showClear = !!clearable && !isLocked && displayValue.length > 0;
  const invalid = isInvalid || rejected !== null;

  const { refs, floatingStyles, middlewareData } = useFloating({
    open: dropdownVisible,
    placement: "bottom-start",
    strategy: "fixed",
    whileElementsMounted: autoUpdate,
    middleware: [
      offset(4),
      flip({ padding: 8 }),
      size({
        apply({ rects, elements }) {
          elements.floating.style.width = `${rects.reference.width}px`;
        },
      }),
      hide(),
    ],
  });
  const anchorHidden = middlewareData.hide?.referenceHidden ?? false;

  function closeDropdown() {
    setSelectedIndex(0);
    setOpen(false);
  }

  function commitDate(date: Date | undefined) {
    setText(date ? formatValue(date) : "");
    setRejected(null);
    onValueChange(date);
  }

  function commitTypedText() {
    const trimmed = text.trim();
    if (!trimmed) {
      setRejected(null);
      if (value) onValueChange(undefined);
      return;
    }
    if (trimmed === committedText) {
      setRejected(null);
      return;
    }
    const parsed = parseInput(trimmed);
    if (parsed) {
      commitDate(parsed);
      return;
    }
    setRejected(trimmed);
    if (value) onValueChange(undefined);
  }

  function selectSuggestion(suggestion: Suggestion) {
    commitDate(suggestion.date);
    closeDropdown();
  }

  function handleClear() {
    setText("");
    setRejected(null);
    setSelectedIndex(0);
    if (value) onValueChange(undefined);
  }

  function handleFocus(e: React.FocusEvent<HTMLInputElement>) {
    onFocus?.(e);
    if (isLocked) return;
    setFocused(true);
    setText(rejected ?? committedText);
    setEdited(rejected !== null);
    setSelectedIndex(0);
  }

  function handleBlur(e: React.FocusEvent<HTMLInputElement>) {
    setFocused(false);
    closeDropdown();
    if (!isLocked) commitTypedText();
    onBlur?.(e);
  }

  function handleClick(e: React.MouseEvent<HTMLInputElement>) {
    onClick?.(e);
    if (isLocked || open) return;
    setOpen(true);
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const next = e.target.value;
    setText(next);
    setEdited(true);
    setSelectedIndex(0);
    setOpen(true);
    if (!next.trim()) {
      setRejected(null);
      if (value) onValueChange(undefined);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    onKeyDown?.(e);
    if (isLocked) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (!open) {
        setOpen(true);
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
        setText(rejected ?? committedText);
        setEdited(false);
        closeDropdown();
      }
    }
  }

  useEffect(() => {
    if (!dropdownVisible) return;
    document
      .getElementById(`${listboxId}-option-${activeIndex}`)
      ?.scrollIntoView({ block: "nearest" });
  }, [activeIndex, dropdownVisible, listboxId]);

  const listed = dropdownVisible && suggestions.length > 0;

  return (
    <div ref={refs.setReference} className="relative">
      <div className="relative">
        <Input
          placeholder={placeholder ?? t("Tomorrow")}
          autoComplete="off"
          spellCheck={false}
          {...props}
          ref={inputRef}
          type="text"
          role="combobox"
          aria-expanded={dropdownVisible}
          aria-controls={listed ? listboxId : undefined}
          aria-autocomplete="list"
          aria-activedescendant={listed ? `${listboxId}-option-${activeIndex}` : undefined}
          aria-invalid={invalid}
          aria-describedby={
            cn(props["aria-describedby"], rejected !== null && !focused && hintId) || undefined
          }
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
            className="text-muted-foreground absolute top-1/2 right-8 size-5 -translate-y-1/2 [&>svg]:size-3"
          >
            <span className="sr-only">{t("Clear date")}</span>
            <XCloseIcon className="size-4" />
          </Button>
        )}
        {picker}
      </div>
      {rejected !== null && !focused ? (
        <p id={hintId} className="text-danger-foreground mt-1 text-2xs">
          {t('"{0}" is not a date. Try "t+2", "next friday", or "07/15".', rejected)}
        </p>
      ) : null}

      {dropdownVisible &&
        createPortal(
          <div
            ref={refs.setFloating}
            style={{ ...floatingStyles, visibility: anchorHidden ? "hidden" : undefined }}
            className="dark isolate z-50"
          >
            <div
              className="bg-popover text-popover-foreground animate-in fade-in-0 rounded-lg p-0 ring-1 ring-foreground/10"
              tabIndex={-1}
            >
              {suggestions.length > 0 ? (
                <ul
                  id={listboxId}
                  role="listbox"
                  aria-label={t("Date suggestions")}
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
                      <span className="text-muted-foreground shrink-0 text-xs">
                        {formatValue(suggestion.date)}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-muted-foreground px-3 py-1.5 text-xs">
                  {t('No matching date. Try "t+2", "next friday", or "07/15".')}
                </p>
              )}
            </div>
          </div>,
          document.body,
        )}
    </div>
  );
}
