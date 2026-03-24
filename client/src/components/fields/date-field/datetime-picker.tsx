import { parseDate } from "@/lib/chrono";
import { useEffect, useImperativeHandle, useRef, useState } from "react";
import { createPortal } from "react-dom";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { generateDateTime, generateDateTimeString } from "@/lib/date";
import { cn } from "@/lib/utils";
import { CalendarIcon, XIcon } from "lucide-react";
import type { Suggestion } from "./date-picker";
import { DateTimePickerPopover } from "./datetime-picker-popover";

const defaultSuggestions = [
  "t 0800",
  "t 1200",
  "t 1700",
  "t+1 0800",
  "t+1 1200",
  "t+1 1700",
];

function generateSuggestions(inputValue: string, suggestion: Suggestion | null): Suggestion[] {
  if (!inputValue.length) {
    return defaultSuggestions
      .map((suggestion) => ({
        date: generateDateTime(suggestion),
        inputString: suggestion,
      }))
      .filter((s): s is Suggestion => s.date !== null);
  }

  const filteredDefaultSuggestions = defaultSuggestions.filter((suggestion) =>
    suggestion.toLowerCase().includes(inputValue.toLowerCase()),
  );
  if (filteredDefaultSuggestions.length) {
    return filteredDefaultSuggestions
      .map((suggestion) => ({
        date: generateDateTime(suggestion),
        inputString: suggestion,
      }))
      .filter((s): s is Suggestion => s.date !== null);
  }

  return [suggestion].filter((suggestion) => suggestion !== null);
}

export interface DateTimePickerProps extends React.InputHTMLAttributes<HTMLInputElement> {
  dateTime: Date | undefined;
  setDateTime: (date: Date | undefined) => void;
  isInvalid?: boolean;
  placeholder?: string;
  clearable?: boolean;
  label?: string;
  description?: string;
  ref?: React.Ref<HTMLInputElement>;
}

export function DateTimePicker({
  dateTime,
  setDateTime,
  isInvalid,
  placeholder,
  clearable,
  ref,
  ...props
}: DateTimePickerProps) {
  const [suggestion, setSuggestion] = useState<Suggestion | null>(null);
  const [inputValue, setInputValue] = useState("");
  const [isOpen, setIsOpen] = useState(false);
  const [isClosing, setClosing] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);

  const [isCleared, setIsCleared] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useImperativeHandle(ref, () => inputRef.current!);

  const suggestions = generateSuggestions(inputValue, suggestion);
  const externalInputValue = dateTime && !isCleared ? generateDateTimeString(dateTime) : "";
  const displayInputValue = isOpen ? inputValue : externalInputValue;

  const dateTimeMs = dateTime?.getTime();
  useEffect(() => {
    if (!dateTime) {
      setInputValue("");
    } else {
      setIsCleared(false);
    }
    // oxlint-disable-next-line eslint-plugin-react-hooks/exhaustive-deps
  }, [dateTimeMs]);

  const updateDropdownPosition = () => {
    if (containerRef.current && dropdownRef.current) {
      const rect = containerRef.current.getBoundingClientRect();
      const dropdownHeight = dropdownRef.current.offsetHeight;
      const spaceBelow = window.innerHeight - rect.bottom;
      const spaceAbove = rect.top;
      const placeAbove = spaceBelow < dropdownHeight + 8 && spaceAbove > spaceBelow;

      if (placeAbove) {
        dropdownRef.current.style.top = `${rect.top - dropdownHeight - 4}px`;
      } else {
        dropdownRef.current.style.top = `${rect.bottom + 4}px`;
      }
      dropdownRef.current.style.left = `${rect.left}px`;
      dropdownRef.current.style.width = `${rect.width}px`;
    }
  };

  useEffect(() => {
    if (!isOpen) return;

    const updatePosition = () => updateDropdownPosition();

    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);

    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [isOpen]);

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const { value } = e.target;
    setInputValue(value);

    if (value.length > 0) {
      setIsOpen(true);
    } else {
      setIsOpen(false);
      setIsCleared(true);
      setDateTime(undefined);
    }

    setSelectedIndex(0);

    const result = parseDate(value);
    if (result) {
      setSuggestion({ date: result, inputString: value });
    } else {
      setSuggestion(null);
    }
  }

  function closeDropdown() {
    setClosing(true);
    setSelectedIndex(0);
    setTimeout(() => {
      setIsOpen(false);
      setClosing(false);
    }, 200);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((prevIndex) =>
        prevIndex < suggestions.length - 1 ? prevIndex + 1 : prevIndex,
      );
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((prevIndex) => (prevIndex > 0 ? prevIndex - 1 : 0));
    } else if (e.key === "Enter" && isOpen && suggestions.length > 0) {
      e.preventDefault();
      const dateStr = generateDateTimeString(suggestions[selectedIndex].date);
      setInputValue(dateStr);
      setIsCleared(false);
      setDateTime(suggestions[selectedIndex].date);
      closeDropdown();
    } else if (e.key === "Escape" || e.key === "Tab") {
      closeDropdown();
    }
  }

  function handleClear() {
    setInputValue("");
    setIsCleared(true);
    setDateTime(undefined);
    closeDropdown();
    // inputRef.current?.focus();
  }

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        closeDropdown();
      }
    }

    document.addEventListener("mousedown", handleClickOutside);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <Input
          placeholder={placeholder || "Tomorrow"}
          {...props}
          ref={inputRef}
          aria-invalid={isInvalid}
          type="text"
          value={displayInputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={() => {
            if (!isCleared) {
              setInputValue(externalInputValue);
            }
            setIsOpen(true);
          }}
          onClick={() => {
            if (!isCleared) {
              setInputValue(externalInputValue);
            }
            setIsOpen(true);
          }}
        />
        {clearable && inputValue && (
          <Button
            onClick={(e) => {
              e.stopPropagation();
              handleClear();
            }}
            type="button"
            size="icon"
            className="absolute top-1/2 right-8 size-5 -translate-y-1/2 rounded-sm bg-transparent text-muted-foreground hover:bg-foreground/10"
          >
            <span className="sr-only">Clear date</span>
            <XIcon className="size-4" />
          </Button>
        )}
        <DateTimePickerPopover
          onOpen={() => setSuggestion(null)}
          dateTime={dateTime}
          setDateTime={setDateTime}
          setInputValue={setInputValue}
        >
          <Button
            type="button"
            size="icon"
            variant="outline"
            className="absolute top-1/2 right-2 size-5 -translate-y-1/2 rounded-sm bg-transparent text-muted-foreground hover:bg-foreground/10 [&>svg]:size-4"
          >
            <span className="sr-only">Open normal date time picker</span>
            <CalendarIcon className="size-4" />
          </Button>
        </DateTimePickerPopover>
      </div>

      {isOpen &&
        suggestions.length > 0 &&
        createPortal(
          <div
            ref={(node) => {
              dropdownRef.current = node;
              if (node) {
                updateDropdownPosition();
              }
            }}
            role="dialog"
            className={cn(
              "fixed z-[9999] animate-in rounded-md border bg-popover p-0 shadow-md transition-all fade-in-0 zoom-in-95 slide-in-from-top-2",
              isClosing && "animate-out duration-300 fade-out-0 zoom-out-95",
            )}
            tabIndex={-1}
            aria-label="Suggestions"
          >
            <ul role="listbox" aria-label="Suggestions" className="max-h-56 overflow-auto p-1">
              {suggestions.map((suggestion, index) => (
                <li
                  key={suggestion.inputString}
                  role="option"
                  aria-selected={selectedIndex === index}
                  className={cn(
                    "flex cursor-pointer items-center justify-between gap-1 rounded-sm px-3 py-1.5 text-xs",
                    index === selectedIndex && "bg-muted text-accent-foreground",
                  )}
                  onClick={() => {
                    const dateStr = generateDateTimeString(suggestion.date);
                    setInputValue(dateStr);
                    setIsCleared(false);
                    setDateTime(suggestion.date);
                    closeDropdown();
                    inputRef.current?.focus();
                  }}
                  onMouseEnter={() => setSelectedIndex(index)}
                >
                  <span className="xs:w-auto w-[110px] truncate">{suggestion.inputString}</span>
                  <span className="shrink-0 text-xs text-muted-foreground">
                    {generateDateTimeString(suggestion.date)}
                  </span>
                </li>
              ))}
            </ul>
          </div>,
          document.body,
        )}
    </div>
  );
}
